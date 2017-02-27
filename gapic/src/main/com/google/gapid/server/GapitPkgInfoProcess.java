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
import com.google.gapid.proto.pkginfo.PkgInfo;
import com.google.gapid.proto.pkginfo.PkgInfo.PackageList;

import java.io.File;
import java.util.List;
import java.util.logging.Logger;

/**
 * {@link ChildProcess} running the "gapit packages" command to read the package info from a
 * connected Android device for tracing.
 */
public class GapitPkgInfoProcess extends ChildProcess<PkgInfo.PackageList> {
  private static final Logger LOG = Logger.getLogger(GapitPkgInfoProcess.class.getName());

  private final String deviceSerial;
  private final float iconDensityScale;

  public GapitPkgInfoProcess(String deviceSerial, float iconDensityScale) {
    super("gapit");
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

    if (deviceSerial != null) {
      args.add("--device");
      args.add(deviceSerial);
    }

    pb.command(args);
    return null;
  }

  @Override
  protected OutputHandler<PkgInfo.PackageList> createStdoutHandler() {
    return new BinaryHandler<PkgInfo.PackageList>(in -> {
      byte[] data = ByteStreams.toByteArray(in);
      return (data.length == 0) ? null : PkgInfo.PackageList.parseFrom(data);
    }) {
      @Override
      public void finish(SettableFuture<PackageList> result) throws InterruptedException {
        if (!result.isDone()) {
          result.setException(new Exception("The gapit command didn't produce any output!"));
        }
      }
    };
  }

  @Override
  protected OutputHandler<PackageList> createStderrHandler() {
    StringBuilder output = new StringBuilder();
    return new LoggingStringHandler<PackageList>(LOG, name, true, (line) -> {
      output.append(line).append('\n');
      return null;
    }) {
      @Override
      public void finish(SettableFuture<PackageList> result) throws InterruptedException {
        if (!result.isDone() && output.length() > 0) {
          result.setException(new Exception("The gapit command failed:\n" + output.toString()));
        }
      }
    };
  }
}
