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

import static java.util.logging.Level.WARNING;

import com.google.common.collect.Lists;
import com.google.common.io.ByteStreams;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.pkginfo.PkgInfo;

import java.io.BufferedInputStream;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.util.Arrays;
import java.util.List;
import java.util.logging.Logger;

/**
 * {@link ChildProcess} running the "gapit packages" command to read the package info from a
 * connected Android device for tracing.
 */
public class GapitPkgInfoProcess extends ChildProcess<PkgInfo.PackageList> {
  private static final Logger LOG = Logger.getLogger(GapitPkgInfoProcess.class.getName());
  private static final String PACKAGE_DATA_MARKER = "~-~-~-~-";

  private final String deviceSerial;
  private final float iconDensityScale;

  public GapitPkgInfoProcess(Settings settings, String deviceSerial, float iconDensityScale) {
    super("gapit", settings);
    this.deviceSerial = deviceSerial;
    this.iconDensityScale = iconDensityScale;
  }

  @Override
  protected Exception prepare(ProcessBuilder pb) {
    File gapit = GapiPaths.gapit();
    if (gapit == null || !gapit.exists()) {
      LOG.log(WARNING, "Could not find gapit for package info.");
      return new Exception("Could not find the gapit executable.");
    }

    List<String> args = Lists.newArrayList();
    args.add(gapit.getAbsolutePath());
    args.add("packages");

    args.add("--format");
    args.add("proto");

    args.add("--icons");

    args.add("--icondensity");
    args.add(String.valueOf(iconDensityScale));

    args.add("--dataheader");
    args.add(PACKAGE_DATA_MARKER);

    if (deviceSerial != null) {
      args.add("--device");
      args.add(deviceSerial);
    }

    String adb = GapiPaths.adb(settings);
    if (!adb.isEmpty()) {
      args.add("--adb");
      args.add(adb);
    }

    pb.command(args);
    return null;
  }

  private class SearchingInputStream extends BufferedInputStream {
    public SearchingInputStream(InputStream in) {
      super(in);
    }

    public boolean search(byte[] pattern) throws IOException {
      byte[] test = new byte[pattern.length];
      while (true) {
        mark(pattern.length);
        if (read(test, 0, pattern.length) < 0) {
          return false;
        }
        if (Arrays.equals(test, pattern)) {
          return true;
        }
        reset();
        skip(1);
      }
    }
  }

  @Override
  protected OutputHandler<PkgInfo.PackageList> createStdoutHandler() {
    return new BinaryHandler<PkgInfo.PackageList>(in -> {
      try (SearchingInputStream is = new SearchingInputStream(in)) {
        if (!is.search(PACKAGE_DATA_MARKER.getBytes())) {
          throw new RuntimeException("The gapit command didn't produce the data marker.");
        }
        byte[] data = ByteStreams.toByteArray(is);
        return (data.length == 0) ? null : PkgInfo.PackageList.parseFrom(data);
      }
    }) {
      @Override
      public void finish(SettableFuture<PkgInfo.PackageList> result) throws InterruptedException {
        if (!result.isDone()) {
          result.setException(new Exception("The gapit command didn't produce any output!"));
        }
      }
    };
  }

  @Override
  protected OutputHandler<PkgInfo.PackageList> createStderrHandler() {
    StringBuilder output = new StringBuilder();
    return new LoggingStringHandler<PkgInfo.PackageList>(LOG, name, true, (line) -> {
      output.append(line).append('\n');
      return null;
    }) {
      @Override
      public void finish(SettableFuture<PkgInfo.PackageList> result) throws InterruptedException {
        if (!result.isDone() && output.length() > 0) {
          result.setException(new Exception("The gapit command failed:\n" + output.toString()));
        }
      }
    };
  }
}
