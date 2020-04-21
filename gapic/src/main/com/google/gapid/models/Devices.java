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

import static com.google.common.util.concurrent.Futures.immediateFuture;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.SettingsProto.DeviceValidation;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.device.Device.Instance;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.Value;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.MoreFutures;
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
  protected final Analytics analytics;
  private final Client client;
  private List<Device.Instance> replayDevices;
  private Device.Instance selectedReplayDevice;
  private List<DeviceCaptureInfo> devices;
  private DeviceValidationInfo deviceValidationInfo;

  public static final Flag<Boolean> skipDeviceValidation = Flags.value("skip-device-validation", false,
      "Skips the device validation process. " +
      "Device validation verifies that the GPU events emitted are within the acceptable threshold.", true);

  public Devices(Shell shell, Analytics analytics, Client client, Capture capture, Settings settings) {
    this.shell = shell;
    this.analytics = analytics;
    this.client = client;
    deviceValidationInfo = new DeviceValidationInfo(client, settings);

    capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
        if (!maintainState) {
          resetReplayDevice();
        }
      }

      @Override
      public void onCaptureLoaded(Loadable.Message error) {
        if (error == null && capture.isGraphics()) {
          loadReplayDevices(capture.getData().path);
        }
      }
    });
  }

  protected void resetReplayDevice() {
    replayDevices = null;
    selectedReplayDevice = null;
  }

  public void loadReplayDevices(Path.Capture capturePath) {
    rpcController.start().listen(MoreFutures.transformAsync(client.getDevicesForReplay(capturePath),
        devs -> Futures.allAsList(devs.stream()
            .map(dev -> client.get(Paths.device(dev), dev))
            .collect(toList()))),
        new UiErrorCallback<List<Service.Value>, List<Device.Instance>, Void>(shell, LOG) {
      @Override
      protected ResultOrError<List<Device.Instance>, Void> onRpcThread(Result<List<Value>> result) {
        try {
          List<Device.Instance> devs = result.get().stream()
              .map(v -> v.getDevice())
              .collect(toList());
          return success(devs);
        } catch (RpcException | ExecutionException e) {
          analytics.reportException(e);
          throttleLogRpcError(LOG, "LoadData error", e);
          return error(null);
        }
      }

      @Override
      protected void onUiThreadSuccess(List<Instance> devs) {
        updateReplayDevices(devs);
      }

      @Override
      protected void onUiThreadError(Void error) {
        updateReplayDevices(null);
      }
    });
  }

  protected void updateReplayDevices(List<Device.Instance> devs) {
    replayDevices = devs;
    listeners.fire().onReplayDevicesLoaded();
  }

  public boolean hasReplayDevice() {
    return selectedReplayDevice != null;
  }

  public List<Device.Instance> getReplayDevices() {
    return replayDevices;
  }

  public Device.Instance getSelectedReplayDevice() {
    return selectedReplayDevice;
  }

  public Path.Device getReplayDevicePath() {
    return (selectedReplayDevice == null) ? null : Paths.device(selectedReplayDevice.getID());
  }

  public void selectReplayDevice(Device.Instance dev) {
    selectedReplayDevice = dev;
    listeners.fire().onReplayDeviceChanged(dev);
  }

  public ListenableFuture<DeviceValidationResult> validateDevice(Device.Instance device) {
    return deviceValidationInfo.doValidation(device);
  }

  public DeviceValidationResult getValidationStatus(Device.Instance device) {
    return deviceValidationInfo.getValidationStatus(device);
  }

  public void loadDevices() {
    rpcController.start().listen(MoreFutures.transformAsync(client.getDevices(), paths -> {
      List<ListenableFuture<DeviceCaptureInfo>> results = Lists.newArrayList();
      for (Path.Device path : paths) {
        ListenableFuture<Service.Value> dev = client.get(Paths.device(path), path);
        ListenableFuture<Service.Value> props = client.get(Paths.traceInfo(path), path);
        results.add(MoreFutures.transform(Futures.allAsList(dev, props), l -> {
          return new DeviceCaptureInfo(path, l.get(0).getDevice(), l.get(1).getTraceConfig(),
              new TraceTargets(shell, analytics, client, path));
        }));
      }
      return Futures.allAsList(results);
    }), new UiErrorCallback<List<DeviceCaptureInfo>, List<DeviceCaptureInfo>, Void>(shell, LOG) {
      @Override
      protected ResultOrError<List<DeviceCaptureInfo>, Void> onRpcThread(
          Rpc.Result<List<DeviceCaptureInfo>> result) throws RpcException, ExecutionException {
        try {
          return success(result.get());
        } catch (RpcException | ExecutionException e) {
          throttleLogRpcError(LOG, "LoadData error", e);
          return error(null);
        }
      }

      @Override
      protected void onUiThreadSuccess(List<DeviceCaptureInfo> result) {
        updateDevices(result);
      }

      @Override
      protected void onUiThreadError(Void error) {
        updateDevices(null);
      }
    });
  }

  protected void updateDevices(List<DeviceCaptureInfo> newDevices) {
    devices = newDevices;
    listeners.fire().onCaptureDevicesLoaded();
  }

  public boolean isLoaded() {
    return devices != null;
  }

  public boolean isReplayDevicesLoaded() {
    return replayDevices != null;
  }

  public List<Device.Instance> getAllDevices() {
    return (devices == null) ? null :
      devices.stream().map(info -> info.device).collect(toList());
  }

  public List<DeviceCaptureInfo> getCaptureDevices() {
    return (devices == null) ? null :
      devices.stream().filter(info -> !info.config.getApisList().isEmpty()).collect(toList());
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public static String getLabel(Device.Instance dev) {
    StringBuilder sb = new StringBuilder();
    if (!dev.getName().isEmpty()) {
      sb.append(dev.getName()).append(" - ");
    }

    appendOsLabel(sb, dev.getConfiguration().getOS());
    appendGpuLabel(sb, dev.getConfiguration().getHardware().getGPU());

    if (!dev.getSerial().isEmpty()) {
      sb.append(" - ").append(dev.getSerial());
    }

    return sb.toString();
  }

  private static StringBuilder appendOsLabel(StringBuilder sb, Device.OS os) {
    switch (os.getKind()) {
      case Android: sb.append("Android"); break;
      case Linux: sb.append("Linux"); break;
      case Windows: sb.append("Windows"); break;
      case OSX: sb.append("MacOS"); break;
      default: sb.append("Unknown OS"); break;
    }
    if (!os.getName().isEmpty()) {
      sb.append(" ").append(os.getName());
    }
    if (os.getAPIVersion() != 0) {
      sb.append(" API ").append(os.getAPIVersion());
    }
    return sb;
  }

  private static StringBuilder appendGpuLabel(StringBuilder sb, Device.GPU gpu) {
    if (!gpu.getVendor().isEmpty()) {
      sb.append(" - ").append(gpu.getVendor());
      if (!gpu.getName().isEmpty()) {
        sb.append(" ").append(gpu.getName());
      }
    } else if (!gpu.getName().isEmpty()) {
      sb.append(" - ").append(gpu.getName());
    }
    return sb;
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the replay devices have been loaded.
     */
    public default void onReplayDevicesLoaded() { /* empty */ }

    /**
     * Event indicating that the selected replay device has changed.
     */
    @SuppressWarnings("unused")
    public default void onReplayDeviceChanged(Device.Instance dev) { /* empty */ }

    /**
     * Event indicating that the capture devices have been loaded.
     */
    public default void onCaptureDevicesLoaded() { /* empty */ }
  }


  /**
   * Encapsulates information about a Device and what trace options
   *  are valid for that device.
   */
  public static class DeviceCaptureInfo {
    public final Path.Device path;
    public final Device.Instance device;
    public final Service.DeviceTraceConfiguration config;
    public final TraceTargets targets;

    public DeviceCaptureInfo(Path.Device path, Device.Instance device,
        Service.DeviceTraceConfiguration config, TraceTargets targets) {
      this.path = path;
      this.device = device;
      this.config = config;
      this.targets = targets;
    }

    public boolean isAndroid() {
      return device.getConfiguration().getOS().getKind() == Device.OSKind.Android;
    }

    public boolean isStadia() {
      return device.getConfiguration().getOS().getKind() == Device.OSKind.Stadia;
    }
  }

  /*
   *  Class that handles the gapis communication for device validation and the load/store
   *  of the validation cache.
   */
  private static class DeviceValidationInfo {
    private final Client client;
    private DeviceValidation.Builder deviceValidation;

    public DeviceValidationInfo(Client client, Settings settings) {
      this.client = client;
      deviceValidation = settings.writeDeviceValidation();
    }

    private static DeviceValidation.ValidationEntry buildValidationEntry(Device.Instance device) {
      return DeviceValidation.ValidationEntry.newBuilder()
          .setDevice(DeviceValidation.Device.newBuilder()
              .setSerial(device.getSerial())
              .setOs(device.getConfiguration().getOS())
              .setVersion(device.getConfiguration().getDrivers().getVulkan().getVersion()))
          .setResult(DeviceValidation.Result.newBuilder()
              .setPassed(true))
          .build();
    }

    // TODO(b/149406313): Move to UI thread to avoid synchronization
    public synchronized DeviceValidationResult getValidationStatus(Device.Instance device) {
      if (skipDeviceValidation.get()) {
        return DeviceValidationResult.getSkippedResult();
      }
      DeviceValidation.ValidationEntry entry  = buildValidationEntry(device);
      if (deviceValidation.getValidationEntriesList().contains(entry)) {
        return DeviceValidationResult.getPassedResult();
      }
      return DeviceValidationResult.getFailedResult();
    }

    // TODO(b/149406313): Move to UI thread to avoid synchronization
    public synchronized ListenableFuture<DeviceValidationResult> doValidation(Device.Instance device) {
      if (device == null) {
        return immediateFuture(DeviceValidationResult.getFailedResult());
      }
      DeviceValidation.ValidationEntry currentEntry = buildValidationEntry(device);
      if (deviceValidation.getValidationEntriesList().contains(currentEntry)) {
        return immediateFuture(DeviceValidationResult.getPassedResult());
      }

      return MoreFutures.transform(client.validateDevice(Paths.device(device.getID())), e -> {
        updateValidationStatus(currentEntry, e);
        return new DeviceValidationResult(e.getError(), !e.hasError(), false);
      });
    }

    // TODO(b/149406313): Move to UI thread to avoid synchronization
    protected synchronized void updateValidationStatus(DeviceValidation.ValidationEntry entry, Service.ValidateDeviceResponse response) {
      if (response == null || response.hasError()) {
        return;
      }
      deviceValidation.addValidationEntries(entry);
    }
  }

  public static class DeviceValidationResult {
    private static DeviceValidationResult passedResult = new DeviceValidationResult(null, true, false);
    private static DeviceValidationResult failedResult = new DeviceValidationResult(null, false, false);
    private static DeviceValidationResult skippedResult = new DeviceValidationResult(null, true, true);
    public Service.Error error;
    public boolean passed;
    public boolean skipped;

    public DeviceValidationResult(Service.Error error, boolean passed, boolean skipped) {
      this.error = error;
      this.passed = passed;
      this.skipped = skipped;
    }

    public static DeviceValidationResult getPassedResult() {
      return passedResult;
    }

    public static DeviceValidationResult getSkippedResult() {
      return skippedResult;
    }

    public static DeviceValidationResult getFailedResult() {
      return failedResult;
    }
  }
}
