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

import com.google.common.util.concurrent.FutureCallback;

import java.util.concurrent.CancellationException;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * {@link FutureCallback} implementation that simply logs errors, but otherwise ignores them.
 */
public abstract class LoggingCallback<T> implements FutureCallback<T> {
  private final Logger log;
  private final boolean ignoreCancel;

  public LoggingCallback(Logger log) {
    this(log, false);
  }

  public LoggingCallback(Logger log, boolean ignoreCancel) {
    this.log = log;
    this.ignoreCancel = ignoreCancel;
  }

  @Override
  public void onFailure(Throwable t) {
    if (!ignoreCancel || !(t instanceof CancellationException)) {
      log.log(Level.WARNING, "Unexpected and unhandled exception in async processing.", t);
    }
  }
}
