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
import static com.google.gapid.util.Paths.capture;
import static com.google.gapid.views.ErrorDialog.showErrorDialog;
import static java.util.logging.Level.INFO;
import static java.util.logging.Level.WARNING;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.InternalServerErrorException;
import com.google.gapid.server.Client.UnsupportedVersionException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Experimental;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.MoreFutures;

import org.eclipse.swt.widgets.Shell;

import java.io.File;
import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Model containing information about the currently loaded trace.
 */
public class Capture extends ModelBase<Capture.Data, File, Loadable.Message, Capture.Listener> {
  protected static final Logger LOG = Logger.getLogger(Capture.class.getName());

  // Don't try to open files with 16 or less bytes. An empty graphics trace, without the
  // capture header is already 16 bytes.
  private static final int MIN_FILE_SIZE = 16;

  private final Settings settings;
  private String name = "";

  public Capture(Shell shell, Analytics analytics, Client client, Settings settings) {
    super(LOG, shell, analytics, client, Listener.class);
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

  public boolean isGraphics() {
    return isLoaded() && getData().isGraphics();
  }

  public boolean isPerfetto() {
    return isLoaded() && getData().isPerfetto();
  }

  @Override
  protected ListenableFuture<Data> doLoad(File file) {
    if (!file.exists() || !file.canRead()) {
      return Futures.immediateFailedFuture(
          new BadCaptureException("Trace file does not exist or is not accessible!"));
    } else if (file.length() == 0) {
      return Futures.immediateFailedFuture(
          new BadCaptureException("Trace file is empty!"));
    } else if (file.length() <= MIN_FILE_SIZE) {
      return Futures.immediateFailedFuture(new BadCaptureException(
          "Trace file is empty! Try capturing with buffering disabled."));
    }

    String canonicalPath;
    try {
      File canonicalFile = file.getCanonicalFile();
      canonicalPath = canonicalFile.getAbsolutePath();
      if (canonicalFile.getParentFile() != null) {
        settings.writeFiles().setLastOpenDir(canonicalFile.getParentFile().getAbsolutePath());
      }
    } catch (IOException e) {
      if (file.getParentFile() != null) {
        settings.writeFiles().setLastOpenDir(file.getParentFile().getAbsolutePath());
      }

      return Futures.immediateFailedFuture(
          new BadCaptureException("Failed to read trace file: " + e.getMessage(), e));
    }

    settings.addToRecent(canonicalPath);
    return MoreFutures.transformAsync(client.loadCapture(canonicalPath), path ->
      MoreFutures.transform(client.get(
          capture(path.getID(), true), Path.Device.getDefaultInstance()),
          val -> new Data(path, val.getCapture())));
  }

  @Override
  protected ResultOrError<Data, Loadable.Message> processResult(Rpc.Result<Data> result) {
    try {
      Data data = result.get();
      if (data == null || data.path == null) {
        return error(Loadable.Message.error("Invalid/Corrupted trace file!"));
      } else if (data.isGraphics() && !Experimental.enableVulkanTracing(settings)) {
        return error(Loadable.Message.error(
            "The experimental graphics trace feature is currently disabled.\n" +
            "Enable it via the --experimental-enable-vulkan-tracing command line flag."));
      } else {
        return success(data);
      }
    } catch (UnsupportedVersionException e) {
      return error(Loadable.Message.error(e));
    } catch (BadCaptureException e) {
      throttleLogRpcError(LOG, "Failed to load trace file", e);
      return error(Loadable.Message.error(e));
    } catch (InternalServerErrorException e) {
      analytics.reportException(e);
      if (e.getMessage().contains("Cause: Failed to convert")) {
        LOG.log(WARNING, "Invalid capture format load error:", e);
        return error(Loadable.Message.error(
            "Failed to load capture: file contains unsupported, outdated, or corrupted trace data."));
      } else {
        return error(Loadable.Message.error(e));
      }
    } catch (RpcException e) {
      analytics.reportException(e);
      return error(Loadable.Message.error(e));
    } catch (ExecutionException e) {
      analytics.reportException(e);
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
        settings.writeFiles().setLastOpenDir(canonicalFile.getParentFile().getAbsolutePath());
      }
    } catch (IOException e) {
      if (file.getParentFile() != null) {
        settings.writeFiles().setLastOpenDir(file.getParentFile().getAbsolutePath());
      }

      LOG.log(Level.WARNING, "Failed to save trace", e);
      showErrorDialog(shell, analytics, "Failed to save trace:\n  " + e.getMessage(), e);
      return;
    }

    settings.addToRecent(canonicalPath);

    rpcController.start().listen(client.saveCapture(getData().path, canonicalPath),
        new UiErrorCallback<Void, Boolean, Exception>(shell, LOG) {
      @Override
      protected ResultOrError<Boolean, Exception> onRpcThread(Rpc.Result<Void> result)
          throws RpcException, ExecutionException {
        try {
          result.get();
          return success(true);
        } catch (ExecutionException | RpcException e) {
          return error(e);
        }
      }

      @Override
      protected void onUiThreadSuccess(Boolean unused) {
        LOG.log(INFO, "Trace saved.");
      }

      @Override
      protected void onUiThreadError(Exception error) {
        analytics.reportException(error);
        throttleLogRpcError(LOG, "Couldn't save trace", error);
        showErrorDialog(shell, analytics, "Failed to save trace:\n  " + error.getMessage(), error);
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
    // TODO: maybe reload the capture?
    updateSuccess(new Data(newPath, getData().capture));
  }

  public static class Data {
    public final Path.Capture path;
    public final Service.Capture capture;

    public Data(Path.Capture path, Service.Capture capture) {
      this.path = path;
      this.capture = capture;
    }

    public boolean isGraphics() {
      return capture != null && capture.getType() == Service.TraceType.Graphics;
    }

    public boolean isPerfetto() {
      return capture != null && capture.getType() == Service.TraceType.Perfetto;
    }
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

  private static class BadCaptureException extends RpcException {
    public BadCaptureException(String message) {
      super(message);
    }

    public BadCaptureException(String message, Throwable cause) {
      super(message, cause);
    }
  }
}
