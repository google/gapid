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
import static com.google.gapid.views.ErrorDialog.showErrorDialog;
import static java.util.logging.Level.INFO;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.UnsupportedVersionException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;

import org.eclipse.swt.widgets.Shell;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Model containing information about the currently loaded trace.
 */
public class Capture extends ModelBase<Path.Capture, File, Loadable.Message, Capture.Listener> {
  protected static final Logger LOG = Logger.getLogger(Capture.class.getName());

  private final Settings settings;
  private String name = "";

  public Capture(Shell shell, Client client, Settings settings) {
    super(LOG, shell, client, Listener.class);
    this.settings = settings;
  }

  public String getName() {
    return name;
  }

  public void loadCapture(File file) {
    LOG.log(INFO, "Loading capture " + file + "...");
    name = file.getName();
    load(file, true);
  }

  @Override
  protected ListenableFuture<Path.Capture> doLoad(File file) {
    if (!file.exists() || !file.canRead()) {
      return Futures.immediateFailedFuture(
          new Exception("Trace file does not exist or is not accessible"));
    } else if (file.length() == 0) {
      return Futures.immediateFailedFuture(new Exception("Trace file is empty!"));
    }

    String canonicalPath;
    try {
      File canonicalFile = file.getCanonicalFile();
      canonicalPath = canonicalFile.getAbsolutePath();
      if (canonicalFile.getParentFile() != null) {
        settings.lastOpenDir = canonicalFile.getParentFile().getAbsolutePath();
      }
    } catch (IOException e) {
      if (file.getParentFile() != null) {
        settings.lastOpenDir = file.getParentFile().getAbsolutePath();
      }

      return Futures.immediateFailedFuture(new Exception("Failed to load trace!", e));
    }

    settings.addToRecent(canonicalPath);
    return client.loadCapture(canonicalPath);
  }

  @Override
  protected ResultOrError<Path.Capture, Loadable.Message> processResult(
      Rpc.Result<Path.Capture> result) {
    try {
      Path.Capture capturePath = result.get();
      if (capturePath == null) {
        return error(Loadable.Message.error("Invalid/Corrupted trace file!"));
      } else {
        return success(capturePath);
      }
    } catch (UnsupportedVersionException e) {
      return error(Loadable.Message.error(e.getMessage()));
    } catch (RpcException e) {
      return error(Loadable.Message.error(e));
    } catch (ExecutionException e) {
      throttleLogRpcError(LOG, "Failed to load trace", e);
      return error(Loadable.Message.error(e.getCause().getMessage()));
    }
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onCaptureLoadingStart(false);
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onCaptureLoaded(null);
  }

  public void saveCapture(File file) {
    LOG.log(INFO, "Saving capture " + file + "...");
    name = file.getName();

    // TODO: refactor out duplicate code from loadCapture.
    String canonicalPath;
    try {
      File canonicalFile = file.getCanonicalFile();
      canonicalPath = canonicalFile.getAbsolutePath();
      if (canonicalFile.getParentFile() != null) {
        settings.lastOpenDir = canonicalFile.getParentFile().getAbsolutePath();
      }
    } catch (IOException e) {
      if (file.getParentFile() != null) {
        settings.lastOpenDir = file.getParentFile().getAbsolutePath();
      }

      LOG.log(Level.WARNING, "Failed to save trace", e);
      showErrorDialog(shell, "Failed to save trace:\n  " + e.getMessage(), e);
      return;
    }

    settings.addToRecent(canonicalPath);

    rpcController.start().listen(client.exportCapture(getData()),
        new UiErrorCallback<byte[], Boolean, Exception>(shell, LOG) {
      @Override
      protected ResultOrError<Boolean, Exception> onRpcThread(Rpc.Result<byte[]> result)
          throws RpcException, ExecutionException {
        try {
          byte[] data = result.get();
          try (FileOutputStream fos = new FileOutputStream(file)) {
            fos.write(data);
          }
          return success(true);
        } catch (ExecutionException | RpcException | IOException e) {
          return error(e);
        }
      }

      @Override
      protected void onUiThreadSuccess(Boolean unused) {
        LOG.log(INFO, "Trace saved.");
      }

      @Override
      protected void onUiThreadError(Exception error) {
        throttleLogRpcError(LOG, "Couldn't save trace", error);
        showErrorDialog(shell, "Failed to save trace:\n  " + error.getMessage(), error);
      }
    });
  }

  @Override
  protected void updateError(Loadable.Message error) {
    listeners.fire().onCaptureLoaded(error);
  }

  public void updateCapture(Path.Capture newPath, String newName) {
    if (newName == null) {
      if (!name.startsWith("*")) {
        name = "*" + name;
      }
    } else {
      name = newName;
    }
    listeners.fire().onCaptureLoadingStart(newName == null);
    updateSuccess(newPath);
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the capture is currently being loaded.
     * @param maintainState whether listeners should attempt to maintain their state from a
     *     previous capture.
     */
    public default void onCaptureLoadingStart(boolean maintainState) { /* empty */ }

    /**
     * Event indicating that the capture has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onCaptureLoaded(Loadable.Message error) { /* empty */ }
  }
}
