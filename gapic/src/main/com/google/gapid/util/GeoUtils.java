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
package com.google.gapid.util;

import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;

/**
 * Utilities for {@link Point points}, {@link Rectangle rectangles}, etc.
 */
public class GeoUtils {
  private GeoUtils() {
  }

  public static int left(Rectangle r) {
    return r.x;
  }

  public static int right(Rectangle r) {
    return r.x + r.width;
  }

  public static int top(Rectangle r) {
    return r.y;
  }

  public static int bottom(Rectangle r) {
    return r.y + r.height;
  }

  public static Point center(Rectangle r) {
    return new Point(r.x + r.width / 2, r.y + r.height / 2);
  }

  public static int horCenter(Rectangle r) {
    return r.x + r.width / 2;
  }

  public static int vertCenter(Rectangle r) {
    return r.y + r.height / 2;
  }

  public static Point topLeft(Rectangle r) {
    return new Point(r.x, r.y);
  }

  public static Point topRight(Rectangle r) {
    return new Point(r.x + r.width, r.y);
  }

  public static Point bottomLeft(Rectangle r) {
    return new Point(r.x, r.y + r.height);
  }

  public static Point bottomRight(Rectangle r) {
    return new Point(r.x + r.width, r.y + r.height);
  }

  public static Rectangle withX(Rectangle r, int x) {
    r.x = x;
    return r;
  }

  public static Rectangle withXW(Rectangle r, int x, int width) {
    r.x = x;
    r.width = width;
    return r;
  }

  public static Rectangle withXH(Rectangle r, int x, int height) {
    r.x = x;
    r.height = height;
    return r;
  }

  public static Rectangle withY(Rectangle r, int y) {
    r.y = y;
    return r;
  }

  public static Rectangle withW(Rectangle r, int width) {
    r.width = width;
    return r;
  }

  public static Rectangle withH(Rectangle r, int height) {
    r.height = height;
    return r;
  }
}
