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

import static java.util.Arrays.stream;
import static java.util.logging.Level.FINE;
import static java.util.stream.Collectors.joining;
import static java.util.stream.StreamSupport.stream;

import com.google.common.base.Splitter;
import com.google.gapid.server.GapiPaths;
import com.google.gapid.util.OS;

import org.eclipse.swt.graphics.Point;

import java.io.File;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;
import java.io.Reader;
import java.io.Writer;
import java.util.Properties;
import java.util.logging.Logger;

/**
 * Stores various settings based on user interactions with the UI to maintain customized looks and
 * other shortcuts between runs. E.g. size and location of the window, directory of the last opened
 * file, options for the last trace, etc. The settings are persisted in a ".gapic" file in the
 * user's home directory.
 */
public class Settings {
  private static final Logger LOG = Logger.getLogger(Settings.class.getName());
  private static final String SETTINGS_FILE = ".gapic";
  private static final int MAX_RECENT_FILES = 16;

  public Point windowLocation = null;
  public Point windowSize = null;
  public boolean hideScrubber = false;
  public boolean hideLeft = false;
  public boolean hideRight = false;
  public int splitterTopHeight = 200;
  public String[] leftTabs = new String[0], centerTabs = new String[0], rightTabs = new String[0];
  public String[] hiddenTabs = new String[] { "Log" };
  public double[] tabWeights = new double[] { 0.2, 0.6, 0.2 };
  public String lastOpenDir = "";
  public int[] reportSplitterWeights = new int[] { 75, 25 };
  public int[] shaderSplitterWeights = new int[] { 70, 30 };
  public int[] texturesSplitterWeights = new int[] { 20, 80 };
  public String traceApi = "";
  public String traceDevice = "";
  public String tracePackage = "";
  public String traceOutDir = "";
  public boolean traceClearCache = false;
  public boolean traceDisablePcs = true;
  public String traceExecutable = "";
  public String traceArgs = "";
  public String traceCwd = "";
  public boolean traceMidExecution = false;
  public int traceFrameCount = 0;
  public boolean skipWelcomeScreen = false;
  public String[] recentFiles = new String[0];
  public String adb = "";
  public boolean autoCheckForUpdates = true;
  public boolean updateAvailable = false;
  public long lastCheckForUpdates = 0; // milliseconds since midnight, January 1, 1970 UTC.

  public static Settings load() {
    Settings result = new Settings();

    File file = new File(OS.userHomeDir, SETTINGS_FILE);
    if (file.exists() && file.canRead()) {
      try (Reader reader = new FileReader(file)) {
        Properties properties = new Properties();
        properties.load(reader);
        result.updateFrom(properties);
      } catch (IOException e) {
        LOG.log(FINE, "IO error reading properties from " + file, e);
      }
    }

    return result;
  }

  public void save() {
    File file = new File(OS.userHomeDir, SETTINGS_FILE);
    try (Writer writer = new FileWriter(file)) {
      Properties properties = new Properties();
      updateTo(properties);
      properties.store(writer, " GAPIC Properties");
    } catch (IOException e) {
      LOG.log(FINE, "IO error writing properties to " + file, e);
    }
  }

  public void addToRecent(String file) {
    for (int i = 0; i < recentFiles.length; i++) {
      if (file.equals(recentFiles[i])) {
        if (i != 0) {
          // Move to front.
          System.arraycopy(recentFiles, 0, recentFiles, 1, i);
          recentFiles[0] = file;
        }
        return;
      }
    }

    // Not found.
    if (recentFiles.length >= MAX_RECENT_FILES) {
      String[] tmp = new String[MAX_RECENT_FILES];
      System.arraycopy(recentFiles, 0, tmp, 1, MAX_RECENT_FILES - 1);
      recentFiles = tmp;
    } else {
      String[] tmp = new String[recentFiles.length + 1];
      System.arraycopy(recentFiles, 0, tmp, 1, recentFiles.length);
      recentFiles = tmp;
    }
    recentFiles[0] = file;
  }

  public String[] getRecent() {
    return stream(recentFiles)
        .map(file -> new File(file))
        .filter(File::exists)
        .filter(File::canRead)
        .map(File::getAbsolutePath)
        .toArray(l -> new String[l]);
  }

  private void updateFrom(Properties properties) {
    windowLocation = getPoint(properties, "window.pos");
    windowSize = getPoint(properties, "window.size");
    hideScrubber = getBoolean(properties, "hide.scrubber", hideScrubber);
    hideLeft = getBoolean(properties, "hide.left", hideLeft);
    hideRight = getBoolean(properties, "hide.right", hideRight);
    splitterTopHeight = getInt(properties, "splitter.topHeight", splitterTopHeight);
    leftTabs = getStringList(properties, "tabs.left", leftTabs);
    centerTabs = getStringList(properties, "tabs.center", centerTabs);
    rightTabs = getStringList(properties, "tabs.right", rightTabs);
    hiddenTabs = getStringList(properties, "tabs.hidden", hiddenTabs);
    tabWeights = getDoubleList(properties, "tabs.weights", tabWeights);
    lastOpenDir = properties.getProperty("lastOpenDir", lastOpenDir);
    reportSplitterWeights =
        getIntList(properties, "report.splitter.weights", reportSplitterWeights);
    shaderSplitterWeights =
        getIntList(properties, "shader.splitter.weights", shaderSplitterWeights);
    texturesSplitterWeights =
        getIntList(properties, "texture.splitter.weights", texturesSplitterWeights);
    traceApi = properties.getProperty("trace.api", traceApi);
    traceDevice = properties.getProperty("trace.device", traceDevice);
    tracePackage = properties.getProperty("trace.package", tracePackage);
    traceOutDir = properties.getProperty("trace.dir", traceOutDir);
    traceClearCache = getBoolean(properties, "trace.clearCache", traceClearCache);
    traceDisablePcs = getBoolean(properties, "trace.disablePCS", traceDisablePcs);
    traceExecutable = properties.getProperty("trace.executable", traceExecutable);
    traceArgs = properties.getProperty("trace.args", traceArgs);
    traceCwd  = properties.getProperty("trace.cwd", traceCwd);
    traceMidExecution = getBoolean(properties, "trace.midExecution", traceMidExecution);
    traceFrameCount = getInt(properties, "trace.frameCount", traceFrameCount);
    skipWelcomeScreen = getBoolean(properties, "skip.welcome", skipWelcomeScreen);
    recentFiles = getStringList(properties, "open.recent", recentFiles);
    adb = tryFindAdb(properties.getProperty("adb.path", ""));
    autoCheckForUpdates = getBoolean(properties, "updates.autoCheck", autoCheckForUpdates);
    lastCheckForUpdates = getLong(properties, "updates.lastCheck", 0);
    updateAvailable = getBoolean(properties, "updates.available", updateAvailable);
  }

  private void updateTo(Properties properties) {
    setPoint(properties, "window.pos", windowLocation);
    setPoint(properties, "window.size", windowSize);
    properties.setProperty("hide.scrubber", Boolean.toString(hideScrubber));
    properties.setProperty("hide.left", Boolean.toString(hideLeft));
    properties.setProperty("hide.right", Boolean.toString(hideRight));
    properties.setProperty("splitter.topHeight", Integer.toString(splitterTopHeight));
    setStringList(properties, "tabs.left", leftTabs);
    setStringList(properties, "tabs.center", centerTabs);
    setStringList(properties, "tabs.right", rightTabs);
    setStringList(properties, "tabs.hidden", hiddenTabs);
    setDoubleList(properties, "tabs.weights", tabWeights);
    properties.setProperty("lastOpenDir", lastOpenDir);
    setIntList(properties, "report.splitter.weights", reportSplitterWeights);
    setIntList(properties, "shader.splitter.weights", shaderSplitterWeights);
    setIntList(properties, "texture.splitter.weights", texturesSplitterWeights);
    properties.setProperty("trace.api", traceApi);
    properties.setProperty("trace.device", traceDevice);
    properties.setProperty("trace.package", tracePackage);
    properties.setProperty("trace.dir", traceOutDir);
    properties.setProperty("trace.clearCache", Boolean.toString(traceClearCache));
    properties.setProperty("trace.disablePCS", Boolean.toString(traceDisablePcs));
    properties.setProperty("trace.executable", traceExecutable);
    properties.setProperty("trace.args", traceArgs);
    properties.setProperty("trace.cwd", traceCwd);
    properties.setProperty("trace.midExecution", Boolean.toString(traceMidExecution));
    properties.setProperty("trace.frameCount", Integer.toString(traceFrameCount));
    properties.setProperty("skip.welcome", Boolean.toString(skipWelcomeScreen));
    setStringList(properties, "open.recent", recentFiles);
    properties.setProperty("adb.path", adb);
    properties.setProperty("updates.autoCheck", Boolean.toString(autoCheckForUpdates));
    properties.setProperty("updates.lastCheck", Long.toString(lastCheckForUpdates));
    properties.setProperty("updates.available", Boolean.toString(updateAvailable));
  }

  private static Point getPoint(Properties properties, String name) {
    int x = getInt(properties, name + ".x", -1), y = getInt(properties, name + ".y", -1);
    return (x >= 0 && y >= 0) ? new Point(x, y) : null;
  }

  private static int getInt(Properties properties, String name, int dflt) {
    String value = properties.getProperty(name);
    if (value == null) {
      return dflt;
    }

    try {
      return Integer.parseInt(value);
    } catch (NumberFormatException e) {
      return dflt;
    }
  }

  private static long getLong(Properties properties, String name, long dflt) {
    String value = properties.getProperty(name);
    if (value == null) {
      return dflt;
    }

    try {
      return Long.parseLong(value);
    } catch (NumberFormatException e) {
      return dflt;
    }
  }

  private static boolean getBoolean(Properties properties, String name, boolean dflt) {
    if (dflt) {
      return !"false".equalsIgnoreCase(properties.getProperty(name));
    } else {
      return "true".equalsIgnoreCase(properties.getProperty(name));
    }
  }

  private static int[] getIntList(Properties properties, String name, int[] dflt) {
    String value = properties.getProperty(name);
    if (value == null) {
      return dflt;
    }

    try {
      return stream(Splitter.on(',').split(value).spliterator(), false)
          .mapToInt(Integer::parseInt).toArray();
    } catch (NumberFormatException e) {
      return dflt;
    }
  }

  private static double[] getDoubleList(Properties properties, String name, double[] dflt) {
    String value = properties.getProperty(name);
    if (value == null) {
      return dflt;
    }

    try {
      return stream(Splitter.on(',').split(value).spliterator(), false)
          .mapToDouble(Double::parseDouble).toArray();
    } catch (NumberFormatException e) {
      return dflt;
    }
  }


  private static String[] getStringList(Properties properties, String name, String[] dflt) {
    String value = properties.getProperty(name);
    if (value == null) {
      return dflt;
    }
    return stream(
        Splitter.on(',').trimResults().omitEmptyStrings().split(value).spliterator(), false)
        .toArray(l -> new String[l]);
  }

  private static void setPoint(Properties properties, String name, Point point) {
    if (point != null) {
      properties.setProperty(name + ".x", Integer.toString(point.x));
      properties.setProperty(name + ".y", Integer.toString(point.y));
    }
  }

  private static void setIntList(Properties properties, String name, int[] value) {
    properties.setProperty(name, stream(value).mapToObj(String::valueOf).collect(joining(",")));
  }

  private static void setDoubleList(Properties properties, String name, double[] value) {
    properties.setProperty(name, stream(value).mapToObj(String::valueOf).collect(joining(",")));
  }

  private static void setStringList(Properties properties, String name, String[] value) {
    properties.setProperty(name, stream(value).collect(joining(",")));
  }

  private static String tryFindAdb(String current) {
    if (!current.isEmpty()) {
      return current;
    }

    String[] sdkVars = { "ANDROID_HOME", "ANDROID_SDK_HOME", "ANDROID_ROOT", "ANDROID_SDK_ROOT" };
    for (String sdkVar : sdkVars) {
      File adb = findAdbInSdk(System.getenv(sdkVar));
      if (adb != null) {
        return adb.getAbsolutePath();
      }
    }

    // If not found, but the flag is specified, use that.
    return GapiPaths.adbPath.get();
  }

  private static File findAdbInSdk(String sdk) {
    if (sdk == null || sdk.isEmpty()) {
      return null;
    }
    File adb = new File(sdk, "platform-tools" + File.separator + "adb" + OS.exeExtension);
    return (adb.exists() && adb.canExecute()) ? adb : null;
  }
}
