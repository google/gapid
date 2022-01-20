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

import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_INSTANCE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_PHYSICAL_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_QUEUE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.MemoryTracking.MEMORY_TRACKING_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.MemoryTracking.MEMORY_TRACKING_DRIVER;
import static java.util.logging.Level.FINE;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.server.Client;
import com.google.gapid.server.GapiPaths;
import com.google.gapid.util.OS;
import com.google.protobuf.TextFormat;

import org.eclipse.swt.graphics.Point;

import java.io.File;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;
import java.io.Reader;
import java.io.Writer;
import java.nio.file.FileSystems;
import java.nio.file.Files;
import java.nio.file.InvalidPathException;
import java.nio.file.Path;
import java.util.Arrays;
import java.util.List;
import java.util.UUID;
import java.util.logging.Logger;

/**
 * Stores various settings based on user interactions with the UI to maintain customized looks and
 * other shortcuts between runs. E.g. size and location of the window, directory of the last opened
 * file, options for the last trace, etc. The settings are persisted in a ".agic" file in the
 * user's home directory.
 */
public class Settings {
  private static final Logger LOG = Logger.getLogger(Settings.class.getName());

  private static final String SETTINGS_FILE = ".agic";
  private static final int MAX_RECENT_FILES = 16;
  private static final int CURRENT_VERSION = 4;

  // Only set values for fields where the proto zero/false/empty default doesn't make sense.
  private static SettingsProto.Settings DEFAULT_SETTINGS = SettingsProto.Settings.newBuilder()
      .setVersion(CURRENT_VERSION)
      .setTabs(SettingsProto.Tabs.newBuilder()
          .setStructure("g2;f1;f1;") // See GraphicsTraceView.MainTab.getFolders().
          .addAllWeights(Arrays.asList(-1, 1, 4))
          .addAllTabs(Arrays.asList("Profile", "ApiCalls"))
          .addAllHidden(Arrays.asList("Log")))
      .setUi(SettingsProto.UI.newBuilder()
          .setPerfetto(SettingsProto.UI.Perfetto.newBuilder()
              .setDrawerHeight(250))
          .setFramebufferPicker(SettingsProto.UI.FramebufferPicker.newBuilder()
              .setEnabled(true)))
      .setTrace(SettingsProto.Trace.newBuilder()
          .setType("Graphics")
          .setGfxDuration(SettingsProto.Trace.Duration.newBuilder()
              .setType(SettingsProto.Trace.Duration.Type.MANUAL)
              .setStartFrame(100)
              .setStartTime(10)
              .setDuration(1))
          .setProfileDuration(SettingsProto.Trace.Duration.newBuilder()
              .setType(SettingsProto.Trace.Duration.Type.MANUAL)
              .setStartTime(10)
              .setDuration(5)))
      .setPerfetto(SettingsProto.Perfetto.newBuilder()
          .setCpu(SettingsProto.Perfetto.CPU.newBuilder()
              .setEnabled(true)
              .setChain(true)
              .setSlices(true))
          .setGpu(SettingsProto.Perfetto.GPU.newBuilder()
              .setEnabled(true)
              .setSlices(true)
              .setCounters(true)
              .setCounterRate(1)
              .setSurfaceFlinger(false))
          .setMemory(SettingsProto.Perfetto.Memory.newBuilder()
              .setEnabled(true)
              .setRate(10))
          .setBattery(SettingsProto.Perfetto.Battery.newBuilder()
              .setEnabled(true)
              .setRate(250))
          .setVulkan(SettingsProto.Perfetto.Vulkan.newBuilder()
              .setEnabled(true)
              .addCpuTimingCategories(CPU_TIMING_DEVICE)
              .addCpuTimingCategories(CPU_TIMING_INSTANCE)
              .addCpuTimingCategories(CPU_TIMING_PHYSICAL_DEVICE)
              .addCpuTimingCategories(CPU_TIMING_QUEUE)
              .addMemoryTrackingCategories(MEMORY_TRACKING_DEVICE)
              .addMemoryTrackingCategories(MEMORY_TRACKING_DRIVER)))
      .build();

  private final SettingsProto.Settings.Builder proto;

  private Settings(SettingsProto.Settings.Builder proto) {
    this.proto = fixup(proto);
  }

  @SuppressWarnings("deprecation")
  private static SettingsProto.Settings.Builder fixup(SettingsProto.Settings.Builder proto) {
    tryFindAdb(proto.getPreferencesBuilder());
    switch (proto.getVersion()) {
      case 0:
        // Version 1 introduced the Trace.Duration message. Set the defaults for it.
        proto.getTraceBuilder()
            .setGfxDuration(SettingsProto.Trace.Duration.newBuilder()
                .setType(SettingsProto.Trace.Duration.Type.MANUAL)
                .setStartFrame(100)
                .setDuration(7))
            .setProfileDuration(SettingsProto.Trace.Duration.newBuilder()
                .setType(SettingsProto.Trace.Duration.Type.MANUAL)
                .setDuration(5));
        //$FALL-THROUGH$
      case 1:
        // Version 2 added the DeviceValidation.ValidationEntry.last_seen field.
        for (SettingsProto.DeviceValidation.ValidationEntry.Builder entry :
            proto.getDeviceValidationBuilder().getValidationEntriesBuilderList()) {
          entry.setLastSeen(System.currentTimeMillis());
        }
        //$FALL-THROUGH$
      case 2:
        // Version 3 removed the Trace.api field and expanded the Trace.Type field.
        if ("Graphics".equals(proto.getTrace().getType()) &&
            "OpenGL on ANGLE".equals(proto.getTrace().getApi())) {
          proto.getTraceBuilder().setType("ANGLE");
        }
        proto.getTraceBuilder().clearApi();
        //$FALL-THROUGH$
      case 3:
        // Version 4 resets the default layout after the mergin of the Performance and Command tabs.
        proto.setTabs(DEFAULT_SETTINGS.getTabs());
    }
    return proto.setVersion(CURRENT_VERSION);
  }

  public static Settings load() {
    File file = new File(OS.userHomeDir, SETTINGS_FILE);
    if (file.exists() && file.canRead()) {
      try (Reader reader = new FileReader(file)) {
        SettingsProto.Settings.Builder read = SettingsProto.Settings.newBuilder();
        TextFormat.Parser.newBuilder()
            .setAllowUnknownFields(true)
            .build()
            .merge(reader, read);
        return new Settings(read);
      } catch (TextFormat.ParseException e) {
        LOG.log(FINE, "Proto parse error reading properties from " + file, e);
      } catch (IOException e) {
        LOG.log(FINE, "IO error reading properties from " + file, e);
      }
    }
    return new Settings(DEFAULT_SETTINGS.toBuilder());
  }

  public void save() {
    File file = new File(OS.userHomeDir, SETTINGS_FILE);
    try (Writer writer = new FileWriter(file)) {
      TextFormat.printer().print(proto, writer);
    } catch (IOException e) {
      LOG.log(FINE, "IO error writing properties to " + file, e);
    }
  }

  public SettingsProto.WindowOrBuilder window() {
    return proto.getWindowOrBuilder();
  }

  public SettingsProto.TabsOrBuilder tabs() {
    return proto.getTabsOrBuilder();
  }

  public SettingsProto.FilesOrBuilder files() {
    return proto.getFilesOrBuilder();
  }

  public SettingsProto.PreferencesOrBuilder preferences() {
    return proto.getPreferencesOrBuilder();
  }

  public SettingsProto.UIOrBuilder ui() {
    return proto.getUiOrBuilder();
  }

  public SettingsProto.TraceOrBuilder trace() {
    return proto.getTraceOrBuilder();
  }

  public SettingsProto.PerfettoOrBuilder perfetto() {
    return proto.getPerfettoOrBuilder();
  }

  public SettingsProto.FuchsiaTracingOrBuilder fuchsiaTracing() {
    return proto.getFuchsiaTracingOrBuilder();
  }

  public SettingsProto.Window.Builder writeWindow() {
    return proto.getWindowBuilder();
  }

  public SettingsProto.Tabs.Builder writeTabs() {
    return proto.getTabsBuilder();
  }

  public SettingsProto.Files.Builder writeFiles() {
    return proto.getFilesBuilder();
  }

  public SettingsProto.Preferences.Builder writePreferences() {
    return proto.getPreferencesBuilder();
  }

  public SettingsProto.UI.Builder writeUi() {
    return proto.getUiBuilder();
  }

  public SettingsProto.Trace.Builder writeTrace() {
    return proto.getTraceBuilder();
  }

  public SettingsProto.Perfetto.Builder writePerfetto() {
    return proto.getPerfettoBuilder();
  }

  public SettingsProto.FuchsiaTracing.Builder writeFuchsiaTracing() {
    return proto.getFuchsiaTracingBuilder();
  }

  public SettingsProto.DeviceValidation.Builder writeDeviceValidation() {
    return proto.getDeviceValidationBuilder();
  }

  public int[] getSplitterWeights(SplitterWeights type) {
    return type.get(ui());
  }

  public void setSplitterWeights(SplitterWeights type, int[] weights) {
    type.set(writeUi(), weights);
  }

  // -1 to gracefully handle defaults.
  public static Point getPoint(SettingsProto.Point point) {
    if (point.getX() <= 0 && point.getY() <= 0) {
      return null;
    }
    return new Point(point.getX() - 1, point.getY() - 1);
  }

  // +1 to gracefully handle defaults.
  public static SettingsProto.Point setPoint(Point point) {
    if (point == null) {
      return SettingsProto.Point.getDefaultInstance();
    }
    return SettingsProto.Point.newBuilder()
        .setX(point.x + 1)
        .setY(point.y + 1)
        .build();
  }

  public ListenableFuture<Void> updateOnServer(Client client) {
    SettingsProto.Preferences.Builder prefs = writePreferences();
    return client.updateSettings(prefs.getReportCrashes(), !prefs.getAnalyticsClientId().isEmpty(),
        prefs.getAnalyticsClientId(), prefs.getAdb());
  }

  public void addToRecent(String file) {
    List<String> recentFiles = Lists.newArrayList(files().getRecentList());
    if (!recentFiles.isEmpty() && file.equals(recentFiles.get(0))) {
      // Nothing to do.
      return;
    }

    for (int i = 1; i < recentFiles.size(); i++) {
      String recent = recentFiles.get(i);
      if (file.equals(recent)) {
        // Move to front.
        recentFiles.remove(i);
        break;
      }
    }

    while (recentFiles.size() >= MAX_RECENT_FILES) {
      recentFiles.remove(recentFiles.size() - 1);
    }

    writeFiles()
      .clearRecent()
      .addRecent(file)
      .addAllRecent(recentFiles);
  }

  public String[] getRecent() {
    return files().getRecentList().stream()
        .map(file -> new File(file))
        .filter(File::exists)
        .filter(File::canRead)
        .map(File::getAbsolutePath)
        .toArray(String[]::new);
  }

  public boolean analyticsEnabled() {
    return !preferences().getAnalyticsClientId().isEmpty();
  }

  public void setAnalyticsEnabled(boolean enabled) {
    if (enabled && !analyticsEnabled()) {
      writePreferences().setAnalyticsClientId(UUID.randomUUID().toString());
    } else if (!enabled && analyticsEnabled()) {
      writePreferences().clearAnalyticsClientId();
    }
  }

  public boolean isAdbValid() {
    return checkAdbIsValid(preferences().getAdb()) == null;
  }

  /**
   * Returns an error message if adb is invalid or {@code null} if it is valid.
   */
  public static String checkAdbIsValid(String adb) {
    if (adb.isEmpty()) {
      return "path is missing";
    }

    try {
      Path path = FileSystems.getDefault().getPath(adb);
      if (!Files.exists(path)) {
        return "path does not exist";
      } else if (!Files.isRegularFile(path)) {
        return "path is not a file";
      } else if (!Files.isExecutable(path)) {
        return "path is not an executable";
      }
    } catch (InvalidPathException e) {
      return "path is invalid: " + e.getReason();
    }

    return null;
  }

  private static void tryFindAdb(SettingsProto.Preferences.Builder current) {
    if (!current.getAdb().isEmpty()) {
      return;
    }

    String[] sdkVars = { "ANDROID_HOME", "ANDROID_ROOT", "ANDROID_SDK_ROOT" };
    for (String sdkVar : sdkVars) {
      File adb = findAdbInSdk(System.getenv(sdkVar));
      if (adb != null) {
        current.setAdb(adb.getAbsolutePath());
        return;
      }
    }

    // If not found, but the flag is specified, use that.
    String adb = GapiPaths.adbPath.get();
    if (!adb.isEmpty()) {
      current.setAdb(adb);
      return;
    }
  }

  private static File findAdbInSdk(String sdk) {
    if (sdk == null || sdk.isEmpty()) {
      return null;
    }
    File adb = new File(sdk, "platform-tools" + File.separator + "adb" + OS.exeExtension);
    return (adb.exists() && adb.canExecute()) ? adb : null;
  }

  public static enum SplitterWeights {
    Report(new int[] { 75, 25 }),
    Shaders(new int[] { 20, 80 }),
    Textures(new int[] { 30, 70 }),
    Commands(new int[] { 20, 80 });

    private final int[] dflt;

    private SplitterWeights(int[] dflt) {
      this.dflt = dflt;
    }

    public int[] get(SettingsProto.UIOrBuilder ui) {
      return (ui.containsSplitterWeights(name())) ?
          toArray(ui.getSplitterWeightsOrThrow(name())) : dflt;
    }

    public void set(SettingsProto.UI.Builder ui, int[] weights) {
      SettingsProto.UI.SplitterWeights.Builder sw = SettingsProto.UI.SplitterWeights.newBuilder();
      for (int weight : weights) {
        sw.addWeight(weight);
      }
      ui.putSplitterWeights(name(), sw.build());
    }

    private int[] toArray(SettingsProto.UI.SplitterWeights w) {
      if (w.getWeightCount() == 0) {
        return dflt;
      }
      int[] r = new int[w.getWeightCount()];
      for (int i = 0; i < r.length; i++) {
        r[i] = w.getWeight(i);
      }
      return r;
    }
  }
}
