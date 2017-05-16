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

import com.google.gapid.proto.service.Service;

/**
 * Version specifier.
 *
 * @see <a href="https://github.com/google/gapid/blob/master/core/app/run.go">
 *   $GPUPATH/core/app/run.go VersionSpec</a>
 */
public class Version {
  public static final Version GAPIC_VERSION = new Version(0, 2, 0);

  public final int major;
  public final int minor;
  public final int point;

  public Version(int major, int minor, int point) {
    this.major = major;
    this.minor = minor;
    this.point = point;
  }

  public static Version fromProto(Service.ServerInfo info) {
    return new Version(info.getVersionMajor(), info.getVersionMinor(), info.getVersionPoint());
  }

  public boolean isCompatible(Version version) {
    return major == version.major && minor == version.minor;
  }

  @Override
  public int hashCode() {
    return major << 22 | minor << 12 | point;
  }

  @Override
  public boolean equals(Object obj) {
    if (obj == this) {
      return true;
    } else if (!(obj instanceof Version)) {
      return false;
    }
    Version v = (Version)obj;
    return major == v.major && minor == v.minor && point == v.point;
  }

  @Override
  public String toString() {
    return major + "." + minor + "." + point;
  }

  public String toPatternString() {
    return major + "." + minor + ".*";
  }
}
