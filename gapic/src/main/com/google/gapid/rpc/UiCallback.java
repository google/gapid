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
package com.google.gapid.rpc;

import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.logging.Level.FINE;

import com.google.gapid.util.Logging;

import org.eclipse.swt.widgets.Widget;

import java.util.concurrent.CancellationException;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * A {@link com.google.gapid.rpc.Rpc.Callback} that will execute part of the callback on
 * the UI thread.
 */
public abstract class UiCallback<T, U> implements Rpc.Callback<T> {
  private final Widget widget;
  private final Logger logger;

  public UiCallback(Widget widget, Logger logger) {
    this.widget = widget;
    this.logger = logger;
  }

  @Override
  public final void onFinish(Rpc.Result<T> result) {
    try {
      U value = onRpcThread(result);
      scheduleIfNotDisposed(widget, () -> onUiThread(value));
    } catch (CancellationException cancel) {
      logger.log(FINE, "RPC future was cancelled: " + cancel.getMessage());
      // Not an error, don't log.
    } catch (Exception e) {
      Logging.throttleLogRpcError(logger, "error in " + getClass().getName() + ".onRpcThread", e);
    }
  }

  /**
   * Executed on the executor passed to {@link Rpc}'s listen call. The result is then passed to
   * {@link #onUiThread(Object)} which is run on the SWT event dispatch thread.
   */
  protected abstract U onRpcThread(Rpc.Result<T> result) throws RpcException, ExecutionException;

  /**
   * Invoked on the SWT event dispatch thread with the returned value from
   * {@link #onRpcThread(Rpc.Result)}.
   */
  protected abstract void onUiThread(U result);
}
