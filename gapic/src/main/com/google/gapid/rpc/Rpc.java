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
import com.google.common.util.concurrent.MoreExecutors;
import com.google.common.util.concurrent.Uninterruptibles;

import java.util.concurrent.CancellationException;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Executor;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

/**
 * Holds static method helpers for getting RPC call results from {@link ListenableFuture}s
 * created from RPC calls.
 */
public class Rpc {
  private static final Executor EXECUTOR = MoreExecutors.directExecutor();

  /**
   * Blocks and waits for the result of the RPC call, or throws an exception if the RPC call was not
   * successful.
   *
   * <p>{@link RpcException}s packed in {@link ExecutionException}s thrown by the
   * {@link ListenableFuture} are unpacked and rethrown so they can be explicitly
   * handled using {@code catch} clauses by the caller.
   *
   * @param <V> the result value type.
   * @return the result value.
   * @throws RpcException          if there was an error raised by the server.
   * @throws ExecutionException    if there was a non-{@link RpcException} thrown by the
   *                               {@link ListenableFuture}.
   * @throws CancellationException if the computation was cancelled.
   */
  public static <V> V get(final ListenableFuture<V> future, long timeout, TimeUnit unit)
      throws RpcException, TimeoutException, ExecutionException {
    try {
      return Uninterruptibles.getUninterruptibly(future, timeout, unit);
    } catch (ExecutionException e) {
      Throwable cause = e.getCause();
      if (cause instanceof RpcException) {
        throw (RpcException)cause;
      } else if (cause instanceof RejectedExecutionException) {
        throw (RejectedExecutionException)cause;
      }
      throw e;
    }
  }

  /**
   * Calls {@link Callback#onFinish} with the {@link Result} once the {@link ListenableFuture} RPC
   * call has either successfully completed or thrown an exception.
   *
   * @param <V>      the RPC result type.
   * @param future   the {@link ListenableFuture} returned by the invoking the RPC call.
   * @param callback the {@link Callback} to handle {@link Callback#onFinish} events.
   */
  public static <V> void listen(ListenableFuture<V> future, Callback<V> callback) {
    future.addListener(new Runnable() {
      @Override
      public void run() {
        callback.onFinish(new Result<V>(future));
      }
    }, EXECUTOR);
  }

  /**
   * Callback for the {@link #listen} function.
   *
   * @param <V> the RPC result type.
   */
  public static interface Callback <V> {
    /**
     * Called once the RPC call has a result (success or failure).
     *
     * <p>Call {@link Result#get()} to get the RPC result.
     */
    public void onFinish(Result<V> result);
  }

  /**
   * Result wraps the {@link ListenableFuture} passed to {@link #listen}, providing a single
   * {@link #get} method for accessing the result of the RPC call.
   *
   * @param <V> the RPC result type.
   */
  public static class Result <V> {
    private final ListenableFuture<V> future;

    public Result(ListenableFuture<V> future) {
      this.future = future;
    }

    /**
     * Returns the result of the RPC call, or throws an exception if the RPC call was not
     * successful.
     *
     * <p>{@link RpcException}s packed in {@link ExecutionException}s thrown by the
     * {@link ListenableFuture} are unpacked and rethrown so they can be explicitly
     * handled using {@code catch} clauses by the caller.
     *
     * @return the result value.
     * @throws RpcException          if there was an error raised by the server.
     * @throws ExecutionException    if there was a non-{@link RpcException} thrown by the
     *                               {@link ListenableFuture}.
     * @throws CancellationException if the computation was cancelled.
     */
    public V get() throws RpcException, ExecutionException {
      try {
        return Uninterruptibles.getUninterruptibly(future);
      } catch (ExecutionException e) {
        Throwable cause = e.getCause();
        if (cause instanceof RpcException) {
          throw (RpcException)cause;
        } else if (cause instanceof RejectedExecutionException) {
          throw (RejectedExecutionException)cause;
        }
        throw e;
      }
    }
  }
}
