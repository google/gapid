/*
 * Copyright (C) 2018 Google Inc.
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

import com.google.common.base.Function;
import com.google.common.collect.Lists;
import com.google.common.util.concurrent.AsyncFunction;
import com.google.common.util.concurrent.FutureCallback;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.Uninterruptibles;

import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Additional utilities for {@link ListenableFuture ListenableFutures}.
 */
public class MoreFutures {
  private MoreFutures() {
  }

  public static <V> void addCallback(
      ListenableFuture<V> future, FutureCallback<? super V> callback) {
    Futures.addCallback(future, callback, Scheduler.EXECUTOR);
  }

  public static <I, O> ListenableFuture<O> transform(
      ListenableFuture<I> input, Function<? super I, ? extends O> function) {
    return Futures.transform(input, function, Scheduler.EXECUTOR);
  }

  public static <I, O> ListenableFuture<O> transformAsync(
      ListenableFuture<I> input,
      AsyncFunction<? super I, ? extends O> function) {
    return Futures.transformAsync(input, function, Scheduler.EXECUTOR);
  }

  public static <T> ListenableFuture<T> logFailure(Logger log, ListenableFuture<T> future) {
    return logFailure(log, future, false);
  }

  public static <T> ListenableFuture<T> logFailureIgnoringCancel(
      Logger log, ListenableFuture<T> future) {
    return logFailure(log, future, true);
  }

  private static <T> ListenableFuture<T> logFailure(
      Logger log, ListenableFuture<T> future, boolean ignoreCancel) {
    addCallback(future, new LoggingCallback<Object>(log, ignoreCancel) {
      @Override
      public void onSuccess(Object result) {
        // Ignore.
      }
    });
    return future;
  }

  public static <I, O> ListenableFuture<O> combine(
      Iterable<ListenableFuture<I>> futures, Combiner<List<Result<I>>, O> fun) {
    return Futures.whenAllComplete(futures).call(() -> {
      List<Result<I>> results = Lists.newArrayList();
      for (ListenableFuture<I> future : futures) {
        results.add(Result.getUninterruptibly(future));
      }
      return fun.apply(results);
    }, Scheduler.EXECUTOR);
  }

  public static <I, O> ListenableFuture<O> combineAsync(
      Iterable<ListenableFuture<I>> futures, Combiner<List<Result<I>>, ListenableFuture<O>> fun) {
    return Futures.whenAllComplete(futures).callAsync(() -> {
      List<Result<I>> results = Lists.newArrayList();
      for (ListenableFuture<I> future : futures) {
        results.add(Result.getUninterruptibly(future));
      }
      return fun.apply(results);
    }, Scheduler.EXECUTOR);
  }

  public static interface Combiner<I, O> {
    public O apply(I input) throws ExecutionException;
  }

  public static class Result<T> {
    public final T result;
    public final ExecutionException error;

    private Result(T result, ExecutionException error) {
      this.result = result;
      this.error = error;
    }

    public static <T> Result<T> getUninterruptibly(ListenableFuture<T> future) {
      try {
        return new Result<T>(Uninterruptibles.getUninterruptibly(future), null);
      } catch (ExecutionException e) {
        return new Result<T>(null, e);
      }
    }

    public boolean hasFailed() {
      return error != null;
    }

    public boolean succeeded() {
      return error == null;
    }

    public Throwable getCause() {
      return (error == null) ? null : error.getCause();
    }
  }
}
