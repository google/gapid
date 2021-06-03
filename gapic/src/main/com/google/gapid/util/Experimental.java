/*
 * Copyright (C) 2020 Google Inc.
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
package com.google.gapid.util;

import com.google.common.collect.Lists;
import com.google.gapid.models.Settings;
import com.google.gapid.util.Flags.Flag;

import java.util.List;

/**
 * Command line flag definition for experimental features.
 */
public class Experimental {
  public static final Flag<Boolean> enableAll = Flags.value("experimental-enable-all", false,
      "Enable all experimental features. " +
      "Features turned on by this flag are all unstable and under development.");

  public static final Flag<Boolean> enableVulkanTracing = Flags.value("experimental-enable-vulkan-tracing",
      false, "Enable the experimental feature Vulkan tracing.");

  public static final Flag<Boolean> enableAngleTracing = Flags.value("experimental-enable-angle-tracing",
      false, "Enable the experimental feature Angle tracing.");

  public static final Flag<Boolean> enablePerfTab = Flags.value("experimental-enable-perf-tab",
      false, "Enable the experimental feature Counter performance tab.");

  public static final Flag<Boolean> enableProfileExperiments = Flags.value("experimental-enable-profile-experiments",
      false, "Enable Profile Experiments.");

  public static final Flag<Boolean> enableUnstableFeatures = Flags.value("experimental-enable-unstable-features",
      false, "Enable various unstable features that are not ready for use yet.");

  public static List<String> getGapisFlags(boolean enableAllExperimentalFeatures) {
    List<String> args = Lists.newArrayList();
    // The --experimental-enable-all flag is a sugar flag from the UI. GAPIS knows nothing about it.
    if (enableAllExperimentalFeatures || Experimental.enableAll.get()) {
      // All --experimental-enable-<feature-name> flags must be added here.
      args.add("--experimental-enable-vulkan-tracing");
      args.add("--experimental-enable-angle-tracing");
      args.add("--experimental-enable-perf-tab");
    } else {
      if (Experimental.enableVulkanTracing.get()) {
        args.add("--experimental-enable-vulkan-tracing");
      }
      if (Experimental.enableAngleTracing.get()) {
        args.add("--experimental-enable-angle-tracing");
      }
      if (Experimental.enablePerfTab.get()) {
        args.add("--experimental-enable-perf-tab");
      }
    }
    return args;
  }

  public static boolean enableVulkanTracing(Settings settings) {
    return settings.preferences().getEnableAllExperimentalFeatures() ||
        enableAll.get() || enableVulkanTracing.get();
  }

  public static boolean enablePerfTab(Settings settings) {
    return settings.preferences().getEnableAllExperimentalFeatures() ||
        enableAll.get() || enablePerfTab.get();
  }

  public static boolean enableProfileExperiments(Settings settings) {
    return settings.preferences().getEnableAllExperimentalFeatures() ||
        enableAll.get() || enableProfileExperiments.get();
  }

  public static boolean enableUnstableFeatures(Settings settings) {
    return settings.preferences().getEnableAllExperimentalFeatures() ||
        enableAll.get() || enableUnstableFeatures.get();
  }
}
