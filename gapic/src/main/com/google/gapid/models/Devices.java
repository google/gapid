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
import static com.google.gapid.util.MoreFutures.transformAsync;
import static com.google.gapid.widgets.Widgets.submitIfNotDisposed;
import static java.util.concurrent.TimeUnit.DAYS;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.device.Device.Instance;
import com.google.gapid.proto.device.Device.VulkanDriver;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.stringtable.Stringtable;
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

import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Map;
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
  private final DeviceValidationCache validationCache;
  private List<Device.Instance> replayDevices;
  private List<ReplayDeviceInfo> incompatibleReplayDevices;
  private Device.Instance selectedReplayDevice;
  private List<DeviceCaptureInfo> devices;

  public static final Flag<Boolean> skipDeviceValidation = Flags.value("skip-device-validation", false,
      "Skips the device support validation process. " +
      "Device support validation verifies that the GPU events emitted are within the acceptable threshold.", true);

  public Devices(Shell shell, Analytics analytics, Client client, Capture capture, Settings settings) {
    this.shell = shell;
    this.analytics = analytics;
    this.client = client;
    this.validationCache = new DeviceValidationCache(settings);

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
    incompatibleReplayDevices = null;
  }

  public static String GetDriverVersion(Device.Instance device) {
    Device.VulkanDriver vkDriver = device.getConfiguration().getDrivers().getVulkan();
    if (vkDriver.getPhysicalDevicesCount() <= 0) {
      return "no physical device found";
    }
    return Integer.toUnsignedString(vkDriver.getPhysicalDevices(0).getDriverVersion());
  }

  public void loadReplayDevices(Path.Capture capturePath) {
    rpcController.start().listen(MoreFutures.transformAsync(client.getDevicesForReplay(capturePath),
          devs -> {
            ListenableFuture<List<Device.Instance>> allDevices = MoreFutures.transform(
                Futures.allAsList(devs.getListList().stream()
                .map(d -> client.get(Paths.device(d), d))
                .collect(toList())),
            l -> l.stream().map(v -> v.getDevice()).collect(toList()));

            List<Boolean> compatibilities = devs.getCompatibilitiesList();
            List<Stringtable.Msg> reasons = devs.getReasonsList();

            return MoreFutures.transform(allDevices, instances -> {
              List<ReplayDeviceInfo> replayDevs = Lists.newArrayList();
              for (int i = 0; i < instances.size(); ++i) {
                replayDevs.add(new ReplayDeviceInfo(instances.get(i), compatibilities.get(i), reasons.get(i)));
              }
              return replayDevs;
            });
          }),

        new UiErrorCallback<List<ReplayDeviceInfo>, List<ReplayDeviceInfo>, Void>(shell, LOG) {
      @Override
      protected ResultOrError<List<ReplayDeviceInfo>, Void> onRpcThread(Result<List<ReplayDeviceInfo>> result) {
        try {
          return success(result.get());
        } catch (RpcException | ExecutionException e) {
          analytics.reportException(e);
          throttleLogRpcError(LOG, "LoadData error", e);
          return error(null);
        }
      }

      @Override
      protected void onUiThreadSuccess(List<ReplayDeviceInfo> devs) {
        updateReplayDevices(devs);
      }

      @Override
      protected void onUiThreadError(Void error) {
        updateReplayDevices(null);
      }
    });
  }

  protected void updateReplayDevices(List<ReplayDeviceInfo> devs) {
    if (devs == null) {
      replayDevices = null;
      incompatibleReplayDevices = null;
    } else {
      replayDevices = Lists.newArrayList();
      incompatibleReplayDevices = Lists.newArrayList();
      for (ReplayDeviceInfo d: devs) {
        if (d.compatible) {
          replayDevices.add(d.instance);
        } else {
          incompatibleReplayDevices.add(d);
        }
      }
    }
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

  public List<ReplayDeviceInfo> getIncompatibleReplayDevices() {
    return incompatibleReplayDevices;
  }

  public Path.Device getReplayDevicePath() {
    return (selectedReplayDevice == null) ? null : Paths.device(selectedReplayDevice.getID());
  }

  public void selectReplayDevice(Device.Instance dev) {
    selectedReplayDevice = dev;
    listeners.fire().onReplayDeviceChanged(dev);
  }

  public ListenableFuture<DeviceValidationResult> validateDevice(Device.Instance device) {
    DeviceValidationResult fromCache = getValidationStatus(device);
    if (fromCache.passed) {
      return immediateFuture(fromCache);
    }

    return transformAsync(client.validateDevice(Paths.device(device.getID())), r ->
      submitIfNotDisposed(shell, () -> validationCache.add(device, new DeviceValidationResult(r))));
  }

  public DeviceValidationResult getValidationStatus(Device.Instance device) {
    DeviceValidationResult fromCache = validationCache.getFromCache(device);
    if (fromCache != null) {
      return fromCache;
    } else if (skipDeviceValidation.get()) {
      return DeviceValidationResult.SKIPPED;
    } else {
      return DeviceValidationResult.FAILED;
    }
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
    if (devices != null) {
      return devices.stream().map(info -> info.device).collect(toList());
    }
    return Collections.emptyList();
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

  public static String getVulkanDriverVersions(Device.Instance dev) {
    StringBuilder version = new StringBuilder("N/A");
    VulkanDriver vkDriver = dev.getConfiguration().getDrivers().getVulkan();
    boolean first = true;
    for (int i = 0; i < vkDriver.getPhysicalDevicesCount(); i++) {
      if (first) {
        version.setLength(0);
        first = false;
      } else {
        version.append(", ");
      }
      version.append(Integer.toUnsignedString(vkDriver.getPhysicalDevices(0).getDriverVersion()));
    }
    return version.toString();
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
   * Encapsulates information about a replay device.
   */
  public static class ReplayDeviceInfo {
    public final Device.Instance instance;
    public final Boolean compatible;
    public final Stringtable.Msg reason;

    public ReplayDeviceInfo(Instance instance, Boolean compatible, Stringtable.Msg reason) {
      this.instance = instance;
      this.compatible = compatible;
      this.reason = reason;
    }
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

  public static class DeviceValidationResult {
    public static final DeviceValidationResult PASSED = new DeviceValidationResult(null, true, false);
    public static final DeviceValidationResult FAILED = new DeviceValidationResult(null, false, false);
    public static final DeviceValidationResult SKIPPED = new DeviceValidationResult(null, true, true);

    public final Service.Error error;
    public final boolean passed;
    public final boolean skipped;

    public DeviceValidationResult(Service.Error error, boolean passed, boolean skipped) {
      this.error = error;
      this.passed = passed;
      this.skipped = skipped;
    }

    public DeviceValidationResult(Service.ValidateDeviceResponse r) {
      this(r.getError(), !r.hasError(), false);
    }

    @Override
    public String toString() {
      if (this.skipped) {
        return "Skipped";
      } else if (this.passed) {
        return "Passed";
      } else {
        return "Failed";
      }
    }

  }

  private static class DeviceValidationCache {
    private static final long MAX_VALIDATION_AGE = DAYS.toMillis(30);

    private final Map<Key, SettingsProto.DeviceValidation.ValidationEntry.Builder> cache =
        Maps.newHashMap(); // We only remember passed validations.
    private final SettingsProto.DeviceValidation.Builder stored;

    public DeviceValidationCache(Settings settings) {
      this.stored = settings.writeDeviceValidation();
      for (int i = 0; i < stored.getValidationEntriesCount(); i++) {
        SettingsProto.DeviceValidation.ValidationEntry.Builder entry =
            stored.getValidationEntriesBuilder(i);
        if ((System.currentTimeMillis() - entry.getLastSeen()) > MAX_VALIDATION_AGE) {
          stored.removeValidationEntries(i);
          i--;
        } else if (entry.getResult().getPassed()) {
          cache.put(new Key(entry.getDevice()), entry);
        }
      }
    }

    public DeviceValidationResult getFromCache(Device.Instance device) {
      SettingsProto.DeviceValidation.ValidationEntry.Builder entry = cache.get(new Key(device));
      if (entry == null) {
        return null;
      } else {
        entry.setLastSeen(System.currentTimeMillis());
        return DeviceValidationResult.PASSED;
      }
    }

    public DeviceValidationResult add(Device.Instance device, DeviceValidationResult result) {
      if (result.passed) {
        Key key = new Key(device);
        cache.put(key, stored.addValidationEntriesBuilder()
            .setDevice(key.device)
            .setLastSeen(System.currentTimeMillis())
            .setResult(SettingsProto.DeviceValidation.Result.newBuilder()
                .setPassed(true)));
      }
      return result;
    }

    private static class Key {
      public final SettingsProto.DeviceValidation.Device device;

      public Key(Device.Instance device) {
        this.device = SettingsProto.DeviceValidation.Device.newBuilder()
            .setSerial(device.getSerial())
            .setOs(device.getConfiguration().getOS())
            .setVersion(device.getConfiguration().getDrivers().getVulkan().getVersion())
            .build();
      }

      public Key(SettingsProto.DeviceValidation.Device device) {
        this.device = device;
      }

      @Override
      public int hashCode() {
        return device.hashCode();
      }

      @Override
      public boolean equals(Object obj) {
        if (obj == this) {
          return true;
        } else if (!(obj instanceof Key)) {
          return false;
        }
        return device.equals(((Key)obj).device);
      }
    }
  }
}
