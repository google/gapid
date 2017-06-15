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

import org.eclipse.swt.widgets.Widget;

import java.util.logging.Logger;

/**
 * An {@link UiCallback} that handles errors.
 */
public abstract class UiErrorCallback<T, U, E>
    extends UiCallback<T, UiErrorCallback.ResultOrError<U, E>> {
  public UiErrorCallback(Widget widget, Logger logger) {
    super(widget, logger);
  }

  @Override
  protected final void onUiThread(ResultOrError<U, E> result) {
    if (result.hasResult()) {
      onUiThreadSuccess(result.getResult());
    } else {
      onUiThreadError(result.getError());
    }
  }

  protected abstract void onUiThreadSuccess(U result);

  protected abstract void onUiThreadError(E error);

  public static <U, E> ResultOrError<U, E> success(U result) {
    return new ResultOrError<U, E>(result, null);
  }

  public static <U, E> ResultOrError<U, E> error(E error) {
    return new ResultOrError<U, E>(null, error);
  }

  public static class ResultOrError<T, E> {
    private final T result;
    private final E error;

    ResultOrError(T result, E error) {
      this.result = result;
      this.error = error;
    }

    public T getResult() {
      return result;
    }

    public E getError() {
      return error;
    }

    public boolean hasResult() {
      return result != null;
    }

    public boolean hasError() {
      return error != null;
    }
  }
}
