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

import static java.util.logging.Level.INFO;
import static java.util.logging.Level.SEVERE;

import com.google.gapid.Server.GapisInitException;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.UiErrorCallback;

import org.eclipse.swt.widgets.Shell;

import java.io.File;
import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

public class Capture {
  private static final Logger LOG = Logger.getLogger(Capture.class.getName());

  private final Events.ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private final FutureController rpcController = new SingleInFlight();
  private final Shell shell;
  private final Client client;
  private final Settings settings;
  private Path.Capture path;
  private String name = "";

  public Capture(Shell shell, Client client, Settings settings) {
    this.shell = shell;
    this.client = client;
    this.settings = settings;
  }

  public boolean isLoaded() {
    return path != null;
  }

  public Path.Capture getCapture() {
    return path;
  }

  public String getName() {
    return name;
  }

  public void loadCapture(File file) {
    LOG.log(INFO, "Loading capture " + file + "...");
    path = null;
    name = file.getName();
    listeners.fire().onCaptureLoadingStart();
    if (file.length() == 0) {
      fireError(new GapisInitException(
          GapisInitException.MESSAGE_TRACE_FILE_EMPTY + file, "empty file"));
      return;
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

      fireError(new GapisInitException(
          GapisInitException.MESSAGE_TRACE_FILE_LOAD_FAILED + file, "Loading trace failed", e));
      return;
    }

    Rpc.listen(client.loadCapture(canonicalPath), rpcController,
        new UiErrorCallback<Path.Capture, Path.Capture, GapisInitException>(shell, LOG) {
      @Override
      protected ResultOrError<Path.Capture, GapisInitException> onRpcThread(
          Result<Path.Capture> result) throws RpcException, ExecutionException {
        try {
          Path.Capture capturePath = result.get();
          if (capturePath == null) {
            return error(new GapisInitException(
                GapisInitException.MESSAGE_TRACE_FILE_BROKEN + file, "Invalid/Corrupted"));
          } else {
            return success(capturePath);
          }
        } catch (ExecutionException | RpcException e) {
          return error(new GapisInitException(
              GapisInitException.MESSAGE_TRACE_FILE_LOAD_FAILED + file, "Loading trace failed", e));
        }
      }

      @Override
      protected void onUiThreadSuccess(Path.Capture result) {
        setCapture(result);
      }

      @Override
      protected void onUiThreadError(GapisInitException error) {
        fireError(error);
      }
    });
  }

  protected void fireError(GapisInitException error) {
    LOG.log(SEVERE, "Failed to load capture", error); // TODO show to user.
    listeners.fire().onCaptureLoaded(error);
  }

  protected void setCapture(Path.Capture path) {
    this.path = path;
    listeners.fire().onCaptureLoaded(null);
  }

  public void updateCapture(Path.Capture newPath, String newName) {
    if (newName == null) {
      if (!name.startsWith("*")) {
        name = "*" + name;
      }
    } else {
      name = newName;
    }
    listeners.fire().onCaptureLoadingStart();
    setCapture(newPath);
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    public default void onCaptureLoadingStart() { /* empty */ }
    public default void onCaptureLoaded(GapisInitException error) { /* empty */ }
  }
}
