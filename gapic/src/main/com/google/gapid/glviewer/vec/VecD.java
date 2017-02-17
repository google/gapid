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
package com.google.gapid.glviewer.vec;

import static java.lang.Double.doubleToLongBits;

public class VecD {
  public final double x, y, z;
  private final int h;

  public VecD() {
    this(0, 0, 0);
  }

  public VecD(double x, double y, double z) {
    this.x = x;
    this.y = y;
    this.z = z;

    long hl = doubleToLongBits(x) + 31 * (doubleToLongBits(y) + 31 * doubleToLongBits(z));
    this.h = (int)(hl ^ (hl >>> 32));
  }

  @Override
  public int hashCode() {
    return h;
  }

  @Override
  public boolean equals(Object obj) {
    if (!(obj instanceof VecD)) {
      return false;
    }
    VecD o = (VecD) obj;
    return x == o.x && y == o.y && z == o.z;
  }

  public double get(int idx) {
    switch (idx) {
      case 0: return x;
      case 1: return y;
      case 2: return z;
      default: throw new IndexOutOfBoundsException();
    }
  }

  public VecD add(VecD v) {
    return new VecD(x + v.x, y + v.y, z + v.z);
  }

  public VecD add(double v) {
    return new VecD(x + v, y + v, z + v);
  }

  public VecD addScaled(VecD v, double s) {
    return new VecD(x + s * v.x, y + s * v.y, z + s * v.z);
  }

  public VecD subtract(VecD v) {
    return new VecD(x - v.x, y - v.y, z - v.z);
  }

  public VecD subtract(double v) {
    return new VecD(x - v, y - v, z - v);
  }

  public VecD multiply(VecD v) {
    return new VecD(x * v.x, y * v.y, z * v.z);
  }

  public VecD scale(double v) {
    return new VecD(x * v, y * v, z * v);
  }

  public VecD divide(double v) {
    return new VecD(x / v, y / v, z / v);
  }

  public double magnitudeSquared() {
    return x * x + y * y + z * z;
  }

  public double magnitude() {
    return Math.sqrt(magnitudeSquared());
  }

  public double distanceSquared(VecD v) {
    double dx = x - v.x;
    double dy = y - v.y;
    double dz = z - v.z;
    return dx * dx + dy * dy + dz * dz;
  }

  public double distance(VecD v) {
    return Math.sqrt(distanceSquared(v));
  }

  public VecD normalize() {
    double s = magnitude();
    return (s == 0) ? this : divide(s);
  }

  public VecD abs() {
    return new VecD(Math.abs(x), Math.abs(y), Math.abs(z));
  }

  public VecD min(VecD v) {
    return new VecD(Math.min(x, v.x), Math.min(y, v.y), Math.min(z, v.z));
  }

  public VecD min(double v) {
    return new VecD(Math.min(x, v), Math.min(y, v), Math.min(z, v));
  }

  public VecD max(VecD v) {
    return new VecD(Math.max(x, v.x), Math.max(y, v.y), Math.max(z, v.z));
  }

  public VecD max(double v) {
    return new VecD(Math.max(x, v), Math.max(y, v), Math.max(z, v));
  }

  public double dot(VecD v) {
    return x * v.x + y * v.y + z * v.z;
  }

  public VecD cross(VecD v) {
    return new VecD(y * v.z - z * v.y, z * v.x - x * v.z, x * v.y - y * v.x);
  }

  public VecD lerp(VecD v, double a) {
    return new VecD(v.x + (x - v.x) * a, v.y + (y - v.y) * a, v.z + (z - v.z) * a);
  }

  public static VecD fromArray(double[] v) {
    return new VecD(v[0], v[1], v[2]);
  }

  public static void min(double[] target, double[] buffer, int offset) {
    target[0] = Math.min(target[0], buffer[offset + 0]);
    target[1] = Math.min(target[1], buffer[offset + 1]);
    target[2] = Math.min(target[2], buffer[offset + 2]);
  }

  public static void min(double[] target, double x, double y, double z) {
    target[0] = Math.min(target[0], x);
    target[1] = Math.min(target[1], y);
    target[2] = Math.min(target[2], z);
  }

  public static void max(double[] target, double[] buffer, int offset) {
    target[0] = Math.max(target[0], buffer[offset + 0]);
    target[1] = Math.max(target[1], buffer[offset + 1]);
    target[2] = Math.max(target[2], buffer[offset + 2]);
  }

  public static void max(double[] target, double x, double y, double z) {
    target[0] = Math.max(target[0], x);
    target[1] = Math.max(target[1], y);
    target[2] = Math.max(target[2], z);
  }
}
