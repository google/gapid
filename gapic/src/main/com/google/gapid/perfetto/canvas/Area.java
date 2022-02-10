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

import org.eclipse.swt.graphics.Rectangle;

import java.util.function.Consumer;

/**
 * Represents a rectangular area.
 */
public class Area {
  public static final Area NONE = new Area(0, 0, 0, 0) {
    @Override
    public Area translate(double dx, double dy) {
      return NONE;
    }

    @Override
    public Area shrink(double dx, double dy) {
      return NONE;
    }

    @Override
    public Area combine(Area o) {
      return o;
    }

    @Override
    public boolean isEmpty() {
      return true;
    }

    @Override
    public void ifNotEmpty(Consumer<Area> c) {
      // Do nothing.
    }

    @Override
    public Area intersect(Area o) {
      return NONE;
    }

    @Override
    public Area intersect(double ox, double oy, double ow, double oh) {
      return NONE;
    }

    @Override
    public boolean intersects(Area o) {
      return false;
    }

    @Override
    public int hashCode() {
      return 0;
    }

    @Override
    public boolean equals(Object obj) {
      return (obj instanceof Area) && ((Area)obj).isEmpty();
    }
  };

  public static final Area FULL = new Area(-1, -1, -1, -1) {
    @Override
    public Area translate(double dx, double dy) {
      return FULL;
    }

    @Override
    public Area shrink(double dx, double dy) {
      return FULL;
    }

    @Override
    public Area combine(Area o) {
      return FULL;
    }

    @Override
    public boolean isEmpty() {
      return false;
    }

    @Override
    public Area intersect(Area o) {
      return o;
    }

    @Override
    public Area intersect(double ox, double oy, double ow, double oh) {
      return (ow <= 0 || oh <= 0) ? NONE : new Area(ox, oy, ow, oh);
    }

    @Override
    public boolean intersects(Area o) {
      return !o.isEmpty();
    }

    @Override
    public int hashCode() {
      return -1;
    }

    @Override
    public boolean equals(Object obj) {
      return obj == this;
    }
  };

  public final double x, y;
  public final double w, h;

  public Area(double x, double y, double w, double h) {
    this.x = x;
    this.y = y;
    this.w = w;
    this.h = h;
  }

  public static Area of(Rectangle r) {
    return new Area(r.x, r.y, r.width, r.height);
  }

  public static Area of(Rectangle r, double scale) {
    return new Area(r.x * scale, r.y * scale, r.width * scale, r.height * scale);
  }

  public Area translate(double dx, double dy) {
    return new Area(x + dx, y + dy, w, h);
  }

  // shrinks the area by dx on the left and right side, and by dy from top and bottom.
  public Area shrink(double dx, double dy) {
    if (2 * dx >= w || 2 * dy >= h) {
      return Area.NONE;
    }
    return new Area(x + dx, y + dy, w - 2 * dx, h - 2 * dy);
  }

  public boolean contains(double px, double py) {
    return px >= x && px < x + w && py >= y && py < y + h;
  }

  public Area combine(Area o) {
    if (o.isEmpty()) {
      return this;
    } else if (o == FULL) {
      return FULL;
    }
    double nx = Math.min(x, o.x);
    double ny = Math.min(y, o.y);
    return new Area(nx, ny,
        Math.max(x + w - nx, o.x + o.w - nx),
        Math.max(y + h - ny, o.y + o.h - ny));
  }

  public Area intersect(Area o) {
    if (o == FULL) {
      return this;
    }
    return intersect(o.x, o.y, o.w, o.h);
  }

  public Area intersect(double ox, double oy, double ow, double oh) {
    double nx = Math.max(x, ox), nw = Math.min(x + w, ox + ow) - nx;
    double ny = Math.max(y, oy), nh = Math.min(y + h, oy + oh) - ny;
    return (nw <= 0 || nh <= 0) ? Area.NONE : new Area(nx, ny, nw, nh);
  }

  public boolean intersects(Area o) {
    if (o == FULL) {
      return true;
    } else if (o.isEmpty()) {
      return false;
    }
    return (x + w > o.x) && (x < o.x + o.w) && (y + h > o.y) && (y < o.y + o.h);
  }

  public boolean isEmpty() {
    return w <= 0 || h <= 0;
  }

  public void ifNotEmpty(Consumer<Area> c) {
    if (!isEmpty()) {
      c.accept(this);
    }
  }

  @Override
  public String toString() {
    return "area { " + x + ", " + y + ", " + w + ", " + h + " }";
  }

  @Override
  public int hashCode() {
    return isEmpty() ? 0 :
      Double.hashCode(x) ^ Double.hashCode(y) ^ Double.hashCode(w) ^ Double.hashCode(h);
  }

  @Override
  public boolean equals(Object obj) {
    if (obj == this) {
      return true;
    } else if (!(obj instanceof Area)) {
      return false;
    }

    Area o = (Area)obj;
    return x == o.x && y == o.y && w == o.w && h == o.h;
  }
}
