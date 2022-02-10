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
package com.google.gapid.perfetto.canvas;

import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;

/**
 * Represents a width, height tuple.
 */
public class Size {
  public static final Size ZERO = new Size(0, 0);

  public final double w;
  public final double h;

  public Size(double w, double h) {
    this.w = w;
    this.h = h;
  }

  public static Size of(double w, double h) {
    return new Size(w, h);
  }

  public static Size of(Point p) {
    return new Size(p.x, p.y);
  }

  public static Size of(Point p, double scale) {
    return new Size(p.x * scale, p.y * scale);
  }

  public static Size of(Rectangle rect) {
    return new Size(rect.width, rect.height);
  }

  public static Size of(Rectangle rect, double scale) {
    return new Size(rect.width * scale, rect.height * scale);
  }

  public static Size vertCombine(double margin, double padding, Size... sizes) {
    double w = 0, h = 0;
    for (Size size : sizes) {
      if (!size.isEmpty()) {
        w = Math.max(w, size.w);
        h += ((h == 0) ? margin + 2 * padding : 2 * padding) + size.h;
      }
    }
    return (h == 0) ? ZERO : new Size(w, h);
  }

  public boolean isEmpty() {
    return w == 0 || h == 0;
  }

  public Point toPoint() {
    return new Point((int)Math.ceil(w), (int)Math.ceil(h));
  }

  @Override
  public String toString() {
    return "size { " + w + ", " + h + " }";
  }
}
