/*
 * Copyright (C) 2017 Google Inc.
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
package com.google.gapid.util;

import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;

import org.eclipse.swt.SWT;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Widget;

import java.util.concurrent.Callable;
import java.util.concurrent.Future;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * Helper class to manage user selection and model selection events.
 */
public abstract class SelectionHandler<W extends Widget> {
  private final Logger log;
  protected final W widget;
  private final AtomicInteger lastSelectionEventId = new AtomicInteger();
  private boolean handlingUiSelection;
  private Future<?> lastSelectionFuture = Futures.immediateFuture(null);

  public SelectionHandler(Logger log, W widget) {
    this.log = log;
    this.widget = widget;

    widget.addListener(SWT.Selection, e -> updateSelectionFromUi(e));
  }

  private void updateSelectionFromUi(Event e) {
    handlingUiSelection = true;
    try {
      updateModel(e);
    } finally {
      handlingUiSelection = false;
    }
  }

  public <T> void updateSelectionFromModel(Callable<T> onBgThread, Consumer<T> onUiThread) {
    if (handlingUiSelection) {
      return;
    }

    int currentSelection = lastSelectionEventId.incrementAndGet();
    lastSelectionFuture.cancel(true);
    ListenableFuture<T> future = Scheduler.EXECUTOR.submit(onBgThread);
    lastSelectionFuture = future;

    MoreFutures.addCallback(future, new LoggingCallback<T>(log) {
      @Override
      public void onSuccess(T result) {
        if (result != null && isCurrentSelection(currentSelection)) {
          scheduleIfNotDisposed(widget, () -> {
            if (isCurrentSelection(currentSelection)) {
              onUiThread.accept(result);
            }
          });
        }
      }
    });
  }

  protected boolean isCurrentSelection(int selection) {
    return lastSelectionEventId.get() == selection;
  }

  protected abstract void updateModel(Event e);
}
