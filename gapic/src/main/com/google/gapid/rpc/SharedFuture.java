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

import com.google.common.util.concurrent.ListenableFuture;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.Executor;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Wraps a {link {@link ListenableFuture} with a reference count. Calling {@link #cancel(boolean)}
 * will only cancel the underlying future, once the reference count hits 0.
 */
public class SharedFuture<V> implements ListenableFuture<V> {
  private final ListenableFuture<V> delegate;
  private final AtomicInteger refCount = new AtomicInteger(1);

  public SharedFuture(ListenableFuture<V> delegate) {
    this.delegate = delegate;
  }

  public static <V> SharedFuture<V> shared(ListenableFuture<V> future) {
    return new SharedFuture<V>(future);
  }

  public ListenableFuture<V> share() {
    refCount.incrementAndGet();
    return this;
  }

  @Override
  public boolean cancel(boolean mayInterruptIfRunning) {
    if (refCount.decrementAndGet() == 0) {
      return delegate.cancel(mayInterruptIfRunning);
    }
    return false;
  }

  @Override
  public boolean isCancelled() {
    return delegate.isCancelled();
  }

  @Override
  public boolean isDone() {
    return delegate.isDone();
  }

  @Override
  public V get() throws InterruptedException, ExecutionException {
    return delegate.get();
  }

  @Override
  public V get(long timeout, TimeUnit unit)
      throws InterruptedException, ExecutionException, TimeoutException {
    return delegate.get(timeout, unit);
  }

  @Override
  public void addListener(Runnable listener, Executor executor) {
    delegate.addListener(listener, executor);
  }
}
