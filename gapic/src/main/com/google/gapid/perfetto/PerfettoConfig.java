/*
 * Copyright (C) 2019 Google Inc.
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
package com.google.gapid.perfetto;

import static java.util.logging.Level.WARNING;

import com.google.common.collect.ImmutableList;
import com.google.gapid.server.GapiPaths;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.protobuf.TextFormat;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.Reader;
import java.util.function.Supplier;
import java.util.logging.Logger;

public class PerfettoConfig {
  public static final Flag<String> perfettoConfig = Flags.value("perfetto", "",
      "Path to a file containing a System Trace config proto in text format. " +
      "Specifying this flag will enable the System Trace UI features");

  private static final Logger LOG = Logger.getLogger(PerfettoConfig.class.getName());
  private static final PerfettoConfig MISSING = new PerfettoConfig(null);

  private final perfetto.protos.PerfettoConfig.TraceConfig config;

  public PerfettoConfig(perfetto.protos.PerfettoConfig.TraceConfig config) {
    this.config = config;
  }

  // This call is expensive as it re-reads the perfetto config file, in order to enable perfetto
  // configuration changes without having to restart GAPID.
  public static synchronized PerfettoConfig get() {
    return ImmutableList.<Supplier<File>>of(
        () -> {
          String path = perfettoConfig.get();
          return "".equals(path) ? null : new File(path);
        },
        GapiPaths.get()::perfettoConfig
    ).stream()
        .map(dir -> checkForConfig(dir.get()))
        .filter(PerfettoConfig::shouldUse)
        .findFirst()
        .orElse(MISSING);
  }

  public boolean hasConfig() {
    return config != null;
  }

  public perfetto.protos.PerfettoConfig.TraceConfig getConfig() {
    return config;
  }

  private static boolean shouldUse(PerfettoConfig config) {
    return config != null && config.hasConfig();
  }

  private static PerfettoConfig checkForConfig(File file) {
    if (file == null || !file.isFile()) {
      return null;
    }
    try (Reader in = new InputStreamReader(new FileInputStream(file))) {
      perfetto.protos.PerfettoConfig.TraceConfig.Builder config =
          perfetto.protos.PerfettoConfig.TraceConfig.newBuilder();
      TextFormat.merge(in, config);
      return new PerfettoConfig(config.build());
    } catch (IOException e) {
      LOG.log(WARNING, "Failed to read System Trace config from " + file, e);
    }
    return null;
  }
}
