package com.google.gapid.perfetto.canvas;

import org.eclipse.swt.graphics.Point;

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
