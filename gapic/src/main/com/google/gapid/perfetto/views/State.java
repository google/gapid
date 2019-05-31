/*
 * Copyright (C) 2019 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.google.gapid.perfetto.views;

import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.concurrent.TimeUnit.MICROSECONDS;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.QueryEngine;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.ThreadInfo;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Events;

import org.eclipse.swt.widgets.Widget;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * Represents the current UI state.
 */
public class State {
  public static final long MAX_ZOOM_SPAN_NSEC = MICROSECONDS.toNanos(100);

  private static final Logger LOG = Logger.getLogger(State.class.getName());

  private final Widget owner;
  private Perfetto.Data data;
  private TimeSpan visibleTime;
  private double width;
  private double nanosPerPx;
  private long resolution;
  private Selection selection;
  private final AtomicInteger lastSelectionUpdateId = new AtomicInteger(0);

  private final Events.ListenerCollection<Listener> listeners = Events.listeners(Listener.class);

  public State(Widget owner) {
    this.data = null; // TODO: zero value
    this.owner = owner;
    this.visibleTime = TimeSpan.ZERO;
    this.width = 0;
    this.selection = null;
  }

  public void update(Perfetto.Data newData) {
    this.data = newData;
    this.visibleTime = (newData == null) ? TimeSpan.ZERO : data.traceTime;
    this.selection = null;
    update();
    listeners.fire().onDataChanged();
  }

  public QueryEngine getQueryEngine() {
    return data.qe;
  }

  public Perfetto.Data getData() {
    return data;
  }

  public TimeSpan getVisibleTime() {
    return visibleTime;
  }

  public TimeSpan getTraceTime() {
    return data.traceTime;
  }

  public long getResolution() {
    return resolution;
  }

  public double timeToPx(long time) {
    return (time - visibleTime.start) / nanosPerPx;
  }

  public long pxToTime(double px) {
    return Math.round(visibleTime.start + px * nanosPerPx);
  }

  public long deltaPxToDuration(double px) {
    return Math.round(px * nanosPerPx);
  }

  public ProcessInfo getProcessInfo(long id) {
    return data.processes.get(id);
  }

  public ThreadInfo getThreadInfo(long id) {
    return data.threads.get(id);
  }

  public Selection getSelection() {
    return selection;
  }

  public void setWidth(double width) {
    if (this.width != width) {
      this.width = width;
      update();
    }
  }

  public boolean setVisibleTime(TimeSpan visibleTime) {
    // TODO: this is not optimal: when zooming in on the right side, then zooming out on the left,
    // the zoom out will hardly zoom.
    visibleTime = visibleTime.boundedBy((data == null) ? TimeSpan.ZERO : data.traceTime);
    if (!this.visibleTime.equals(visibleTime)) {
      this.visibleTime = visibleTime;
      update();
      listeners.fire().onVisibleTimeChanged();
      return true;
    }
    return false;
  }

  public boolean drag(TimeSpan atDragStart, double dx) {
    long dt = deltaPxToDuration(dx);
    return setVisibleTime(atDragStart.move(-dt)
        .boundedByPreservingDuration((data == null) ? TimeSpan.ZERO : data.traceTime));
  }

  public boolean scrollTo(long t) {
    return setVisibleTime(visibleTime.moveTo(t)
        .boundedByPreservingDuration((data == null) ? TimeSpan.ZERO : data.traceTime));
  }

  public void setSelection(ListenableFuture<? extends Selection> futureSel) {
    int myId = lastSelectionUpdateId.incrementAndGet();
    thenOnUiThread(futureSel, newSelection -> {
      if (lastSelectionUpdateId.get() == myId) {
        setSelection(newSelection);
      }
    });
  }

  public void setSelection(Selection selection) {
    lastSelectionUpdateId.incrementAndGet();
    this.selection = selection;
    listeners.fire().onSelectionChanged(selection);
  }

  public <T> void thenOnUiThread(ListenableFuture<T> future, Consumer<T> callback) {
    Rpc.listen(future, new UiCallback<T, T>(owner, LOG) {
      @Override
      protected T onRpcThread(Result<T> result) throws RpcException, ExecutionException {
        return result.get();
      }

      @Override
      protected void onUiThread(T result) {
        callback.accept(result);
      }
    });
  }

  public ListenableFuture<?> onUiThread(Runnable run) {
    SettableFuture<?> result = SettableFuture.create();
    scheduleIfNotDisposed(owner, () -> {
      try {
        run.run();
        result.set(null);
      } catch (Exception e) {
        result.setException(e);
      }
    });
    return result;
  }

  private void update() {
    nanosPerPx = visibleTime.getDuration() / width;
    if (nanosPerPx <= 0) {
      nanosPerPx = 0;
      resolution = 0;
    } else {
      resolution = Math.round(Math.pow(10, Math.floor(Math.log10(nanosPerPx))));
    }
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    public default void onDataChanged() { /* do nothing */ }
    public default void onVisibleTimeChanged() { /* do nothing */ }
    public default void onSelectionChanged(Selection selection) { /* do nothing */}
  }
}
