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

import com.google.common.base.Preconditions;
import com.google.common.util.concurrent.ListenableFuture;

import java.util.concurrent.atomic.AtomicReference;

public class SingleInFlight {
  private final AtomicReference<Context> active = new AtomicReference<Context>();

  public SingleInFlight() {
  }

  public Context start() {
    Context result = new Context() {
      private ListenableFuture<?> future = null;
      private boolean cancelled = false;

      @Override
      public synchronized <T> void listen(ListenableFuture<T> f, Rpc.Callback<T> callback) {
        Preconditions.checkState(future == null);
        if (cancelled) {
          future.cancel(true);
        }
        future = f;
        Rpc.listen(f, callback);
      }

      @Override
      public synchronized void cancel() {
        cancelled = true;
        if (future != null) {
          future.cancel(true);
        }
      }
    };
    Context previous = active.getAndSet(result);
    if (previous != null) {
      previous.cancel();
    }
    return result;
  }

  public interface Context {
    public <T> void listen(ListenableFuture<T> future, Rpc.Callback<T> callback);
    public void cancel();
  }
}
