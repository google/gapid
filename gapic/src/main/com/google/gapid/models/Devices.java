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

import static com.google.gapid.util.Logging.throttleLogRpcError;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Paths;

import org.eclipse.swt.widgets.Shell;

import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Model containing information about capture and replay devices.
 */
public class Devices {
  protected static final Logger LOG = Logger.getLogger(Devices.class.getName());

  private final Events.ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private final SingleInFlight rpcController = new SingleInFlight();
  private final Shell shell;
  private final Client client;
  private Path.Device replayDevice;
  private List<Device.Instance> devices;

  public Devices(Shell shell, Client client, Capture capture) {
    this.shell = shell;
    this.client = client;

    capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
        resetReplayDevice();
      }

      @Override
      public void onCaptureLoaded(Loadable.Message error) {
        if (error == null) {
          loadReplayDevice(capture.getData());
        }
      }
    });
  }

  protected void resetReplayDevice() {
    replayDevice = null;
  }

  protected void loadReplayDevice(Path.Capture capturePath) {
    rpcController.start().listen(client.getDevicesForReplay(capturePath),
        new UiErrorCallback<List<Path.Device>, Path.Device, Void>(shell, LOG) {
      @Override
      protected ResultOrError<Path.Device, Void> onRpcThread(Rpc.Result<List<Path.Device>> result) {
        try {
          List<Path.Device> devs = result.get();
          return (devs == null || devs.isEmpty()) ? error(null) : success(devs.get(0));
        } catch (RpcException | ExecutionException e) {
          throttleLogRpcError(LOG, "LoadData error", e);
          return error(null);
        }
      }

      @Override
      protected void onUiThreadSuccess(Path.Device result) {
        updateReplayDevice(result);
      }

      @Override
      protected void onUiThreadError(Void error) {
        updateReplayDevice(null);
      }
    });
  }

  protected void updateReplayDevice(Path.Device newDevice) {
    replayDevice = newDevice;
    listeners.fire().onReplayDeviceChanged();
  }

  public boolean hasReplayDevice() {
    return replayDevice != null;
  }

  public Path.Device getReplayDevice() {
    return replayDevice;
  }

  public void loadDevices() {
    rpcController.start().listen(Futures.transformAsync(client.getDevices(), paths -> {
      List<ListenableFuture<Service.Value>> results = Lists.newArrayList();
      for (Path.Device path : paths) {
        results.add(client.get(Paths.toAny(path)));
      }
      return Futures.allAsList(results);
    }), new UiErrorCallback<List<Service.Value>, List<Device.Instance>, Void>(shell, LOG) {
      @Override
      protected ResultOrError<List<Device.Instance>, Void> onRpcThread(
          Rpc.Result<List<Service.Value>> result) throws RpcException, ExecutionException {
        try {
          return success(result.get().stream().map(Service.Value::getDevice).collect(toList()));
        } catch (RpcException | ExecutionException e) {
          throttleLogRpcError(LOG, "LoadData error", e);
          return error(null);
        }
      }

      @Override
      protected void onUiThreadSuccess(List<Device.Instance> result) {
        updateDevices(result);
      }

      @Override
      protected void onUiThreadError(Void error) {
        updateDevices(null);
      }
    });
  }

  protected void updateDevices(List<Device.Instance> newDevices) {
    devices = newDevices;
    listeners.fire().onCaptureDevicesLoaded();
  }

  public boolean isLoaded() {
    return devices != null;
  }

  public List<Device.Instance> getAllDevices() {
    return devices;
  }

  public List<Device.Instance> getCaptureDevices() {
    // Only return Android devices.
    return (devices == null) ? null : devices.stream().filter(Devices::isAndroid).collect(toList());
  }

  private static boolean isAndroid(Device.Instance device) {
    return device.getConfiguration().getOS().getKind() == Device.OSKind.Android;
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the selected replay device has changed.
     */
    public default void onReplayDeviceChanged() { /* empty */ }

    /**
     * Event indicating that the capture devices have been loaded.
     */
    public default void onCaptureDevicesLoaded() { /* empty */ }
  }
}
