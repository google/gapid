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

import static com.google.gapid.util.UiErrorCallback.error;
import static com.google.gapid.util.UiErrorCallback.success;
import static java.util.logging.Level.SEVERE;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.proto.service.Service.Value;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.util.PathStore;
import com.google.gapid.util.UiErrorCallback;
import com.google.gapid.util.UiErrorCallback.ResultOrError;

import org.eclipse.swt.widgets.Shell;

import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Base class for models that depend on a capture. I.e. models that will trigger a load whenever
 * the capture changes and require a capture to be loaded.
 */
abstract class CaptureDependentModel<T> {
  private final Logger log;
  protected final Shell shell;
  protected final Client client;
  private final PathStore pathStore = new PathStore();
  private final FutureController rpcController = new SingleInFlight();
  private T data;

  public CaptureDependentModel(Logger log, Shell shell, Client client, Capture capture) {
    this.log = log;
    this.shell = shell;
    this.client = client;

    capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart() {
        reset();
      }

      @Override
      public void onCaptureLoaded(GapisInitException error) {
        if (error == null) {
          load(getPath(capture.getCapture()));
        } else {
          reset();
        }
      }
    });
  }

  protected abstract Path.Any getPath(Path.Capture capturePath);

  protected void load(Path.Any path) {
    if (pathStore.updateIfNotNull(path)) {
      Rpc.listen(doLoad(pathStore.getPath()), rpcController,
          new UiErrorCallback<Value, T, Void>(shell, log) {
        @Override
        protected ResultOrError<T, Void> onRpcThread(Result<Value> result) {
          return processResult(result);
        }

        @Override
        protected void onUiThreadSuccess(T result) {
          update(result);
        }

        @Override
        protected void onUiThreadError(Void error) {
          update(null);
        }
      });
    }
  }

  protected ListenableFuture<Value> doLoad(Path.Any path) {
    return client.get(path);
  }

  protected ResultOrError<T, Void> processResult(Result<Value> result) {
    try {
      return success(unbox(result.get()));
    } catch (RpcException | ExecutionException | IOException e) {
      if (!shell.isDisposed()) {
        log.log(SEVERE, "LoadData error", e);
      }
      return error(null);
    }
  }

  protected abstract T unbox(Value value) throws IOException;

  protected void reset() {
    pathStore.update(null);
    data = null;
  }

  protected void update(T newData) {
    data = newData;
    fireLoadEvent();
  }

  protected abstract void fireLoadEvent();

  public boolean isLoaded() {
    return data != null;
  }

  public Path.Any getPath() {
    return pathStore.getPath();
  }

  public T getData() {
    return data;
  }
}
