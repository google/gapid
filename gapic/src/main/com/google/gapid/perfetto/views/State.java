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

import com.google.common.collect.HashMultimap;
import com.google.common.math.DoubleMath;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.models.CpuInfo;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.ThreadInfo;
import com.google.gapid.perfetto.models.Track;
import com.google.gapid.perfetto.models.TrackConfig;
import com.google.gapid.perfetto.models.VSync;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Events;

import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.widgets.Widget;

import java.math.RoundingMode;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * Represents the current UI state.
 */
public abstract class State {
  public static final long MAX_ZOOM_SPAN_NSEC = MICROSECONDS.toNanos(100);

  private static final Logger LOG = Logger.getLogger(State.class.getName());

  private static final double MIN_WIDTH = 32;

  private final Widget owner;
  private TimeSpan traceTime;
  private TimeSpan visibleTime;
  private double scrollOffset = 0;
  private double width;
  private double maxScrollOffset = 0;
  private double nanosPerPx;
  private long resolution;
  private Selection.MultiSelection selection;
  private final AtomicInteger lastSelectionUpdateId = new AtomicInteger(0);
  private HashMultimap<Long, Long> selectedThreads;     // upid -> utids
  private TimeSpan highlight = TimeSpan.ZERO;

  private final Events.ListenerCollection<Listener> listeners = Events.listeners(Listener.class);

  public State(Widget owner) {
    this.owner = owner;
    this.traceTime = TimeSpan.ZERO;
    this.visibleTime = TimeSpan.ZERO;
    this.width = 0;
    this.selection = null;
    this.selectedThreads = HashMultimap.create();
  }

  public void update(TimeSpan newTraceTime) {
    this.traceTime = newTraceTime;
    this.visibleTime = newTraceTime;
    this.selection = null;
    this.selectedThreads = HashMultimap.create();
    this.highlight = TimeSpan.ZERO;
    update();
    listeners.fire().onDataChanged();
  }

  public TimeSpan getVisibleTime() {
    return visibleTime;
  }

  public double getScrollOffset() {
    return scrollOffset;
  }

  public double getMaxScrollOffset() {
    return maxScrollOffset;
  }

  public TimeSpan getTraceTime() {
    return traceTime;
  }

  public double timeToPx(long time) {
    return (time - visibleTime.start) / nanosPerPx;
  }

  public long pxToTime(double px) {
    return Math.round(visibleTime.start + px * nanosPerPx);
  }

  public double durationToDeltaPx(long time) {
    return time / nanosPerPx;
  }

  public long deltaPxToDuration(double px) {
    return Math.round(px * nanosPerPx);
  }

  public abstract CpuInfo getCpuInfo();
  public abstract ProcessInfo getProcessInfo(long id);
  public abstract ThreadInfo getThreadInfo(long id);

  public Selection.MultiSelection getSelection() {
    return selection;
  }

  public <T extends Selection<T>> T getSelection(Selection.Kind type) {
    if (selection == null) {
      return Selection.emptySelection();
    } else {
      return selection.getSelection(type);
    }
  }

  public RGBA getSliceColorForThread(ThreadInfo thread) {
    if (selectedThreads.isEmpty() || selectedThreads.containsValue(thread.utid)) {
      return thread.getColor().base;
    } else if (selectedThreads.containsKey(thread.upid)) {
      return thread.getColor().alternate;
    } else {
      return thread.getColor().disabled;
    }
  }

  public TimeSpan getHighlight() {
    return highlight;
  }

  public void setWidth(double width) {
    width = Math.max(MIN_WIDTH, width);
    if (this.width != width) {
      this.width = width;
      update();
    }
  }

  public Track.DataRequest toRequest() {
    return new Track.DataRequest(visibleTime, resolution);
  }

  public void setMaxScrollOffset(double maxScrollOffset) {
    this.maxScrollOffset = Math.max(0, maxScrollOffset);
    scrollOffset = Math.min(this.maxScrollOffset, scrollOffset);
    listeners.fire().onVisibleAreaChanged();
  }

  public boolean setVisibleTime(TimeSpan visibleTime) {
    // TODO: this is not optimal: when zooming in on the right side, then zooming out on the left,
    // the zoom out will hardly zoom.
    visibleTime = visibleTime.boundedBy(traceTime);
    if (!this.visibleTime.equals(visibleTime)) {
      this.visibleTime = visibleTime;
      update();
      listeners.fire().onVisibleAreaChanged();
      return true;
    }
    return false;
  }

  public boolean dragX(TimeSpan atDragStart, double dx) {
    long dt = deltaPxToDuration(dx);
    return setVisibleTime(atDragStart.move(-dt).boundedByPreservingDuration(traceTime));
  }

  public boolean dragY(double dy) {
    return scrollToY(scrollOffset - dy);
  }

  public boolean scrollToX(long t) {
    return setVisibleTime(visibleTime.moveTo(t).boundedByPreservingDuration(traceTime));
  }

  public boolean scrollToY(double y) {
    double newScrollOffset = Math.max(0, Math.min(maxScrollOffset, y));
    if (newScrollOffset != scrollOffset) {
      scrollOffset = newScrollOffset;
      listeners.fire().onVisibleAreaChanged();
      return true;
    }
    return false;
  }

  /* Return true if selection state changed. */
  public boolean resetSelections() {
    boolean hasDeselection = selection != null || selectedThreads.size() > 0;
    setSelection((Selection.MultiSelection)null);
    return hasDeselection;
  }

  public void addSelection(Selection.Kind type, ListenableFuture<? extends Selection<?>> futureSel) {
    int myId = lastSelectionUpdateId.incrementAndGet();
    thenOnUiThread(futureSel, newSelection -> {
      if (lastSelectionUpdateId.get() == myId) {
        addSelection(type, newSelection);
      }
    });
  }

  public void addSelection(Selection.Kind type, Selection<?> newSel) {
    if (selection == null) {
      setSelection(type, newSel);
    } else {
      selection.addSelection(type, newSel);
      listeners.fire().onSelectionChanged(selection);
    }
  }

  public void addSelection(ListenableFuture<Selection.MultiSelection> futureSel) {
    int myId = lastSelectionUpdateId.incrementAndGet();
    thenOnUiThread(futureSel, newSelection -> {
      if (lastSelectionUpdateId.get() == myId) {
        if (selection == null) {
          setSelection(newSelection);
        } else {
          selection.addSelection(newSelection);
          listeners.fire().onSelectionChanged(selection);
        }
      }
    });
  }

  public void setSelection(Selection.Kind type, ListenableFuture<? extends Selection<?>> futureSel) {
    int myId = lastSelectionUpdateId.incrementAndGet();
    thenOnUiThread(futureSel, newSelection -> {
      if (lastSelectionUpdateId.get() == myId) {
        setSelection(new Selection.MultiSelection(type, newSelection));
      }
    });
  }

  public void setSelection(Selection.Kind type, Selection<?> selection) {
    setSelection(new Selection.MultiSelection(type, selection));
  }

  public void setSelection(ListenableFuture<Selection.MultiSelection> futureSel) {
    int myId = lastSelectionUpdateId.incrementAndGet();
    thenOnUiThread(futureSel, newSelection -> {
      if (lastSelectionUpdateId.get() == myId) {
        setSelection(newSelection);
      }
    });
  }

  public void setSelection(Selection.MultiSelection selection) {
    lastSelectionUpdateId.incrementAndGet();
    this.selection = selection;
    // If selection is cleared or set to a non-cpu one, don't do color grouping for cpu slices.
    if (selection == null || selection.getSelection(Selection.Kind.Cpu).isEmpty()) {
      clearSelectedThreads();
    }
    listeners.fire().onSelectionChanged(selection);
  }

  public void clearSelectedThreads() {
    selectedThreads = HashMultimap.create();
  }

  public void addSelectedThread(ThreadInfo threadInfo) {
    selectedThreads.put(threadInfo.upid, threadInfo.utid);
  }

  public void setSelectedThread(ThreadInfo threadInfo) {
    clearSelectedThreads();
    selectedThreads.put(threadInfo.upid, threadInfo.utid);
  }

  public void setHighlight(TimeSpan highlight) {
    this.highlight = highlight.boundedBy(traceTime);
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
    if (width <= 0 || nanosPerPx <= 0) {
      nanosPerPx = 0;
      resolution = 0;
    } else {
      resolution = 1l << DoubleMath.log2(nanosPerPx, RoundingMode.FLOOR);
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
    public default void onVisibleAreaChanged() { /* do nothing */ }
    public default void onSelectionChanged(Selection.MultiSelection selection) { /* do nothing */}
  }

  public static class ForSystemTrace extends State {
    private Perfetto.Data data;
    private final PinnedTracks pinnedTracks;

    public ForSystemTrace(Widget owner) {
      super(owner);
      this.data = null; // TODO: null object
      this.pinnedTracks = new PinnedTracks();
    }

    public void update(Perfetto.Data newData) {
      this.data = newData;
      this.pinnedTracks.clear();
      super.update((data == null) ? TimeSpan.ZERO : data.traceTime);
    }

    public boolean hasData() {
      return data != null;
    }

    @Override
    public CpuInfo getCpuInfo() {
      return data.cpu;
    }

    @Override
    public ProcessInfo getProcessInfo(long id) {
      return data.processes.get(id);
    }

    @Override
    public ThreadInfo getThreadInfo(long id) {
      return data.threads.get(id);
    }

    public VSync getVSync() {
      return data.vsync;
    }

    public TrackConfig getTracks() {
      return data.tracks;
    }

    public PinnedTracks getPinnedTracks() {
      return pinnedTracks;
    }
  }
}
