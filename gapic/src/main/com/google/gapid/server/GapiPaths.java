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
package com.google.gapid.server;

import static java.nio.charset.StandardCharsets.UTF_8;
import static java.util.logging.Level.INFO;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.ImmutableList;
import com.google.common.io.Files;
import com.google.common.io.LineProcessor;
import com.google.gapid.models.Settings;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.util.OS;

import java.io.File;
import java.io.IOException;
import java.util.List;
import java.util.function.Supplier;
import java.util.logging.Logger;

/**
 * Utility for locating the various GAPID binaries and associated files.
 */
public final class GapiPaths {
  public static final Flag<String> gapidPath =
      Flags.value("gapid", "", "Path to the gapid binaries.");
  public static final Flag<String> adbPath = Flags.value("adb", "", "Path to the adb binary.");

  public static final GapiPaths MISSING = new GapiPaths(null, null, null, null, null, null);

  private static final Logger LOG = Logger.getLogger(GapiPaths.class.getName());

  private static final String GAPIS_EXECUTABLE_NAME = "gapis" + OS.exeExtension;
  private static final String GAPIT_EXECUTABLE_NAME = "gapit" + OS.exeExtension;
  private static final String STRINGS_DIR_NAME = "strings";
  private static final String PERFETTO_CONFIG_NAME = "perfetto.cfg";
  private static final String USER_HOME_GAPID_ROOT = "gapid";
  private static final String GAPID_PKG_SUBDIR = "pkg";
  private static final String GAPID_ROOT_ENV_VAR = "GAPID";
  private static final String RUNFILES_MANIFEST = "agi.runfiles_manifest";

  private static GapiPaths tools;

  private final File baseDir;
  private final File gapisPath;
  private final File gapitPath;
  private final File stringsPath;
  private final File perfettoConfigPath;
  private final File runfiles;

  public GapiPaths(File baseDir) {
    this.baseDir = baseDir;
    this.gapisPath = new File(baseDir, GAPIS_EXECUTABLE_NAME);
    this.gapitPath = new File(baseDir, GAPIT_EXECUTABLE_NAME);
    this.stringsPath = new File(baseDir, STRINGS_DIR_NAME);
    this.perfettoConfigPath = new File(baseDir, PERFETTO_CONFIG_NAME);
    this.runfiles = null;
  }

  protected GapiPaths(File baseDir, File gapisPath, File gapitPath, File stringsPath,
      File perfettoConfigPath, File runfiles) {
    this.baseDir = baseDir;
    this.gapisPath = gapisPath;
    this.gapitPath = gapitPath;
    this.stringsPath = stringsPath;
    this.perfettoConfigPath = perfettoConfigPath;
    this.runfiles = runfiles;
  }

  public static synchronized GapiPaths get() {
    if (tools == null) {
      tools = findTools();
    }
    return tools;
  }

  public File base() {
    return baseDir;
  }

  public File gapis() throws MissingToolsException {
    if (gapisPath == null || !gapisPath.exists()) {
      throw new MissingToolsException("gapis");
    }
    return gapisPath;
  }

  public File gapit() throws MissingToolsException {
    if (gapisPath == null || !gapisPath.exists()) {
      throw new MissingToolsException("gapit");
    }
    return gapitPath;
  }

  public File strings() {
    return stringsPath;
  }

  public File perfettoConfig() {
    return perfettoConfigPath;
  }

  public void addRunfilesFlag(List<String> args) {
    if (runfiles != null && runfiles.exists()) {
      args.add("--runfiles");
      args.add(runfiles.getAbsolutePath());
    }
  }

  public static String adb(Settings settings) {
    String adb = adbPath.get();
    return adb.isEmpty() ? settings.preferences().getAdb() : adb;
  }

  private static GapiPaths checkForTools(File dir) {
    if (dir == null) {
      return null;
    }

    GapiPaths paths;
    File runfiles = new File(dir, RUNFILES_MANIFEST);
    if (runfiles.exists() && runfiles.canRead()) {
      paths = fromRunfiles(dir, runfiles);
    } else {
      paths = new GapiPaths(dir);
    }

    LOG.log(INFO, "Looking for GAPID in " + dir + " -> " + paths.gapisPath);
    return paths;
  }

  private static GapiPaths fromRunfiles(File dir, File runfiles) {
    try {
      return Files.asCharSource(runfiles, UTF_8).readLines(new LineProcessor<GapiPaths>() {
        private File gapis, gapit, strings;

        @Override
        public boolean processLine(String line) throws IOException {
          String[] tokens = line.split("\\s+", 2);
          if (tokens.length == 2) {
            switch (tokens[0]) {
              case "gapid/cmd/gapis/gapis":
              case "gapid/cmd/gapis/gapis.exe":
                gapis = new File(tokens[1]);
                break;
              case "gapid/cmd/gapit/gapit":
              case "gapid/cmd/gapit/gapit.exe":
                gapit = new File(tokens[1]);
                break;
              case "gapid/gapis/messages/en-us.stb":
                strings = new File(tokens[1]).getParentFile();
                break;
            }
          }
          return true;
        }

        @Override
        public GapiPaths getResult() {
          // Fall back to ignoring the runfiles if no gapis was found.
          return (gapis == null) ? new GapiPaths(dir) :
              new GapiPaths(dir, gapis, gapit, strings, null, runfiles);
        }
      });
    } catch (IOException e) {
      LOG.log(WARNING, "Failed to process runfiles manifest " + runfiles, e);
      // Fall back to ignoring the runfiles.
      return new GapiPaths(dir);
    }
  }

  private static File join(File root, String... paths) {
    if (paths.length == 0) {
      return root;
    }
    StringBuilder sb = new StringBuilder().append(paths[0]);
    for (int i = 1; i < paths.length; i++) {
      sb.append(File.separatorChar).append(paths[i]);
    }
    return new File(root, sb.toString());
  }

  private static boolean shouldUse(GapiPaths paths) {
    // We handle a missing strings and gapit explicitly, so ignore if they're missing.
    return paths != null && paths.gapisPath != null && paths.gapisPath.exists();
  }

  private static GapiPaths findTools() {
    return ImmutableList.<Supplier<File>>of(
      () -> {
        String gapidRoot = gapidPath.get();
        return "".equals(gapidRoot) ? null : new File(gapidRoot);
      }, () -> {
        String gapidRoot = System.getenv(GAPID_ROOT_ENV_VAR);
        return gapidRoot != null && gapidRoot.length() > 0 ? new File(gapidRoot) : null;
      },
      () -> join(new File(OS.userHomeDir), USER_HOME_GAPID_ROOT),
      () -> join(new File(OS.userHomeDir), USER_HOME_GAPID_ROOT, GAPID_PKG_SUBDIR)
    ).stream()
        .map(dir -> checkForTools(dir.get()))
        .filter(GapiPaths::shouldUse)
        .findFirst()
        .orElse(MISSING);
  }

  public static class MissingToolsException extends Exception {
    public MissingToolsException(String tool) {
      super("Could not find the " + tool + " executable.");
    }
  }
}
