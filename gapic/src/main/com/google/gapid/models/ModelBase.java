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
package com.google.gapid.models;

import static com.google.gapid.rpc.UiErrorCallback.error;
import static com.google.gapid.rpc.UiErrorCallback.success;
import static com.google.gapid.util.Logging.throttleLogRpcError;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.ObjectStore;

import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Base class for models.
 *
 * @param <T> the type of this model's data, once loaded
 * @param <S> the source used to load the data (typically a path, see {@link ForPath}
 * @param <E> the error type, when overriding error handling (otherwise use {@code Void})
 * @param <L> the listener type for this model
 */
abstract class ModelBase<T, S, E, L extends Events.Listener> {
  private final Logger log;
  protected final Shell shell;
  protected final Client client;
  private final ObjectStore<S> sourceStore = ObjectStore.create();
  protected final SingleInFlight rpcController = new SingleInFlight();
  protected final Events.ListenerCollection<L> listeners;
  private T data;

  public ModelBase(Logger log, Shell shell, Client client, Class<L> listenerClass) {
    this.log = log;
    this.shell = shell;
    this.client = client;
    this.listeners = Events.listeners(listenerClass);
  }

  protected void load(S source, boolean force) {
    if (sourceStore.updateIfNotNull(source) || force) {
      data = null;
      fireLoadStartEvent();
      rpcController.start().listen(doLoad(source), new UiErrorCallback<T, T, E>(shell, log) {
        @Override
        protected ResultOrError<T, E> onRpcThread(Rpc.Result<T> result) {
          return processResult(result);
        }

        @Override
        protected void onUiThreadSuccess(T result) {
          updateSuccess(result);
        }

        @Override
        protected void onUiThreadError(E error) {
          updateError(error);
        }
      });
    }
  }

  protected ResultOrError<T, E> processResult(Rpc.Result<T> result) {
    try {
      return success(result.get());
    } catch (RpcException | ExecutionException e) {
      if (!shell.isDisposed()) {
        throttleLogRpcError(log, "LoadData error", e);
      }
      return error(null);
    }
  }

  protected void updateSuccess(T result) {
    data = result;
    fireLoadedEvent();
  }

  /**
   * @param error the error as returned by {@link #processResult(Rpc.Result)}.
   */
  protected void updateError(E error) {
    data = null;
    fireLoadedEvent();
  }

  protected abstract void fireLoadStartEvent();
  protected abstract void fireLoadedEvent();

  protected abstract ListenableFuture<T> doLoad(S source);

  public boolean isLoaded() {
    return data != null;
  }

  public S getSource() {
    return sourceStore.get();
  }

  public T getData() {
    return data;
  }

  public void reset() {
    sourceStore.update(null);
    data = null;
  }

  public void addListener(L listener) {
    listeners.addListener(listener);
  }

  public void removeListener(L listener) {
    listeners.removeListener(listener);
  }

  /**
   * A {@link ModelBase} that uses a path as the source.
   */
  public abstract static class ForPath<T, E, L extends Events.Listener>
    extends ModelBase<T, Path.Any, E, L> {

    public ForPath(Logger log, Shell shell, Client client, Class<L> listenerClass) {
      super(log, shell, client, listenerClass);
    }
  }
}
