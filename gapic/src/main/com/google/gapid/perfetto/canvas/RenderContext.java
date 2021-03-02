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

import com.google.common.base.Preconditions;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.gapid.perfetto.canvas.Fonts.Style;
import com.google.gapid.widgets.Theme;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.Transform;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.widgets.Control;

import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * Handles all the drawing operations.
 */
public class RenderContext implements Fonts.TextMeasurer, AutoCloseable {
  protected static final Logger LOG = Logger.getLogger(RenderContext.class.getName());

  private static final int textSizeGreediness = 3; // must be >= 1.
  protected static final double scale = DPIUtil.autoScaleDown(8f);

  public final Theme theme;

  private final GC gc;
  private final ColorCache colors;
  private final Fonts.Context fontContext;
  private final LinkedList<TransformAndClip> transformStack = Lists.newLinkedList();
  private final List<Overlay> overlays = Lists.newArrayList();
  private final Map<String, Long> traces = Maps.newHashMap();
  private Fonts.Style lastFontStyle = Fonts.Style.Normal;

  public RenderContext(Theme theme, GC gc, ColorCache colors, Fonts.Context fontContext) {
    this.theme = theme;
    this.gc = gc;
    this.colors = colors;
    this.fontContext = fontContext;

    Area clip = Area.of(gc.getClipping());
    Transform transform = new Transform(gc.getDevice());
    gc.getTransform(transform);
    transform.scale((float)(1 / scale), (float)(1 / scale));
    gc.setTransform(transform);
    transformStack.push(new TransformAndClip(transform, clip));
    gc.setLineWidth((int)scale);
    fontContext.setFont(gc, Fonts.Style.Normal);
  }

  public void addOverlay(Runnable renderer) {
    float[] transform = new float[6];
    transformStack.getLast().transform.getElements(transform);
    overlays.add(new Overlay(new Transform(gc.getDevice(), transform), renderer));
  }

  public void renderOverlays() {
    if (!overlays.isEmpty()) {
      trace("Overlays", () -> {
        for (Overlay overlay : overlays) {
          gc.setTransform(overlay.transform);
          overlay.renderer.run();
          overlay.dispose();
        }
        overlays.clear();
        gc.setTransform(transformStack.getLast().transform);
      });
    }
  }

  @Override
  public Size measure(Fonts.Style style, String text) {
    return fontContext.measure(style, text);
  }

  @Override
  public double getAscent(Style style) {
    return fontContext.getAscent(style);
  }

  @Override
  public double getDescent(Style style) {
    return fontContext.getDescent(style);
  }

  @Override
  public void close() {
    Preconditions.checkState(
        transformStack.size() == 1, "transform stack size != 1: %s", transformStack.size());
    for (TransformAndClip t : transformStack) {
      t.dispose();
    }
    for (Overlay overlay : overlays) {
      overlay.dispose();
    }
    gc.setTransform(null);
    gc.setLineWidth(0);
    transformStack.clear();
  }

  public void setForegroundColor(RGBA color) {
    gc.setForeground(colors.get(color));
    gc.setAlpha(color.alpha);
  }

  public void setForegroundColor(int sysColor) {
    gc.setForeground(gc.getDevice().getSystemColor(sysColor));
    gc.setAlpha(255);
  }

  public void setBackgroundColor(RGBA color) {
    gc.setBackground(colors.get(color));
    gc.setAlpha(color.alpha);
  }

  public void setBackgroundColor(int sysColor) {
    gc.setBackground(gc.getDevice().getSystemColor(sysColor));
    gc.setAlpha(255);
  }

  public void drawLine(double x1, double y1, double x2, double y2) {
    gc.drawLine(scale(x1), scale(y1), scale(x2), scale(y2));
  }

  public void drawLine(double x1, double y1, double x2, double y2, int lineWidthScale) {
    int lineWidth = gc.getLineWidth();
    try {
      gc.setLineWidth(lineWidth * lineWidthScale);
      drawLine(x1, y1, x2, y2);
    } finally {
      gc.setLineWidth(lineWidth);
    }
  }

  public void drawRect(double x, double y, double w, double h) {
    gc.drawRectangle(rect(x, y, w, h));
  }

  public void drawRect(double x, double y, double w, double h, int lineWidthScale) {
    int lineWidth = gc.getLineWidth();
    try {
      gc.setLineWidth(lineWidth * lineWidthScale);
      drawRect(x, y, w, h);
    } finally {
      gc.setLineWidth(lineWidth);
    }
  }

  public void fillRect(double x, double y, double w, double h) {
    gc.fillRectangle(rect(x, y, w, h));
  }

  public void drawCircle(double cx, double cy, double r) {
    int d = 2 * scale(r);
    gc.drawOval(scale(cx - r), scale(cy - r), d, d);
  }

  public void fillPolygon(double[] xPoints, double[] yPoints, int n) {
    int[] points = new int[2 * n];
    for (int i = 0, j = 0; i < n; i++, j += 2) {
      points[j + 0] = scale(xPoints[i]);
      points[j + 1] = scale(yPoints[i]);
    }
    gc.fillPolygon(points);
  }

  public void drawPolygon(double[] xPoints, double[] yPoints, int n) {
    int[] points = new int[2 * n];
    for (int i = 0, j = 0; i < n; i++, j += 2) {
      points[j + 0] = scale(xPoints[i]);
      points[j + 1] = scale(yPoints[i]);
    }
    gc.drawPolygon(points);
  }

  // x, y is top left corner of text.
  public void drawText(Fonts.Style style, String text, double x, double y) {
    if (style != lastFontStyle) {
      lastFontStyle = style;
      fontContext.setFont(gc, style);
    }
    gc.drawText(text, scale(x), scale(y), SWT.DRAW_TRANSPARENT);
  }

  // draws text centered vertically
  public void drawText(Fonts.Style style, String text, double x, double y, double h) {
    drawText(style, text, x, y + (h - fontContext.measure(style, text).h) / 2);
  }

  // draws the text centered horizontally and vertically, truncated to fit into the given width.
  public void drawText(Fonts.Style style, String text, double x, double y, double w, double h)  {
    drawText(style, text, x, y, w, h, true);
  }

  // draws the text centered horizontally and vertically, only if it fits.
  public void drawTextIfFits(
      Fonts.Style style, String text, double x, double y, double w, double h) {
    drawText(style, text, x, y, w, h, false);
  }

  private void drawText(
      Fonts.Style style, String text, double x, double y, double w, double h, boolean truncate) {
    String toDisplay = text;
    for (int l = text.length(); ; ) {
      Size size = fontContext.measure(style, toDisplay);
      if (size.w < w) {
        drawText(style, toDisplay, x + (w - size.w) / 2 , y + (h - size.h) / 2);
        break;
      } else if (!truncate) {
        break;
      }

      l = Math.min(l - textSizeGreediness, (int)(w / (size.w / toDisplay.length())));
      if (l <= 0) {
        break;
      }
      toDisplay = text.substring(0, l) + "...";
    }
  }

  // draws text centered vertically, left truncated to fit into the given width.
  public void drawTextLeftTruncate(
      Fonts.Style style, String text, double x, double y, double w, double h) {
    drawTextLeftTruncate(style, text, x, y, w, h, false);
  }

  // draws text centered horizontally and vertically, left truncated to fit into the given width.
  public void drawTextCenteredLeftTruncate(
      Fonts.Style style, String text, double x, double y, double w, double h) {
    drawTextLeftTruncate(style, text, x, y, w, h, true);
  }

  private void drawTextLeftTruncate(
      Fonts.Style style, String text, double x, double y, double w, double h, boolean centered) {
    String toDisplay = text;
    for (int l = text.length(); ; ) {
      Size size = fontContext.measure(style, toDisplay);
      if (size.w < w) {
        drawText(style, toDisplay, x + (centered ? (w - size.w) / 2 : 0), y + (h - size.h) / 2);
        break;
      }

      l = Math.min(l - textSizeGreediness, (int)(w / (size.w / toDisplay.length())));
      if (l <= 0) {
        break;
      }
      toDisplay = "..." + text.substring(text.length() - l);
    }
  }

  // draws text centered vertically and horizontally, left truncated to fit into the given width.
  // If the primary text is too long, it falls back to using the alternative.
  public void drawTextLeftTruncate(
      Fonts.Style style, String text, String alternative, double x, double y, double w, double h) {
    Size size = fontContext.measure(style, text);
    if (size.w < w) {
      drawText(style, text, x + (w - size.w)/ 2 , y + (h - size.h) / 2);
    } else {
      drawTextCenteredLeftTruncate(style, alternative, x, y, w, h);
    }
  }

  // draws text centered vertically, right truncated to fit into the given width.
  public void drawTextRightTruncate(
      Fonts.Style style, String text, double x, double y, double w, double h) {
    drawTextRightTruncate(style, text, x, y, w, h, false);
  }

  // draws text centered horizontally and vertically, right truncated to fit into the given width.
  public void drawTextCenteredRightTruncate(
      Fonts.Style style, String text, double x, double y, double w, double h) {
    drawTextRightTruncate(style, text, x, y, w, h, true);
  }

  private void drawTextRightTruncate(
      Fonts.Style style, String text, double x, double y, double w, double h, boolean centered) {
    String toDisplay = text;
    for (int l = text.length(); ; ) {
      Size size = fontContext.measure(style, toDisplay);
      if (size.w < w) {
        drawText(style, toDisplay, x + (centered ? (w - size.w) / 2 : 0), y + (h - size.h) / 2);
        break;
      }

      l = Math.min(l - textSizeGreediness, (int)(w / (size.w / toDisplay.length())));
      if (l <= 0) {
        break;
      }
      toDisplay = text.substring(0, l) + "...";
    }
  }

  public void drawTextTruncate(
      Fonts.Style style, String text, double x, double y, double w, double h, boolean rightTruncate) {
    if (rightTruncate) {
      drawTextRightTruncate(style, text, x, y, w, h);
    } else {
      drawTextLeftTruncate(style, text, x, y, w, h);
    }
  }

  // draws the text on the left of x, and beneath the bottom of y.
  public void drawTextRightJustified(Fonts.Style style, String text, double x, double y) {
    Size size = fontContext.measure(style, text);
    drawText(style, text, x - size.w, y);
  }

  // draws the text centered vertically and on the left of x.
  public void drawTextRightJustified(Fonts.Style style, String text, double x, double y, double h) {
    Size size = fontContext.measure(style, text);
    drawText(style, text, x - size.w, y + (h - size.h) / 2);
  }

  public void drawPath(Path path) {
    gc.drawPath(path.path);
  }

  public void fillPath(Path path) {
    gc.fillPath(path.path);
  }

  public void drawImage(Image image, double x, double y) {
    Rectangle bounds = image.getBounds();
    int sx = scale(x), sy = scale(y), sw = scale(bounds.width), sh = scale(bounds.height);
    gc.drawImage(image, 0, 0, bounds.width, bounds.height, sx, sy, sw, sh);
  }

  // draws the icon centered vertically
  public void drawIcon(Image image, double x, double y, double h) {
    Rectangle size = image.getBounds();
    gc.drawImage(image, 0, 0, size.width,size.height,
        scale(x), scale(y + (h - size.height) / 2), scale(size.width), scale(size.height));
  }

  public void path(Consumer<Path> fun) {
    org.eclipse.swt.graphics.Path path = new org.eclipse.swt.graphics.Path(gc.getDevice());
    try {
      fun.accept(new Path(path));
    } finally {
      path.dispose();
    }
  }

  public Area getClip() {
    return transformStack.getLast().clip;
  }

  public void withTranslation(double x, double y, Runnable run) {
    Area clip = getClip().translate(-x, -y);
    Transform transform = new Transform(gc.getDevice());
    try {
      gc.getTransform(transform);
      transform.translate((float)(x * scale), (float)(y * scale));
      gc.setTransform(transform);
      transformStack.add(new TransformAndClip(transform, clip));
      run.run();
      transformStack.removeLast(); // == transform
      gc.setTransform(transformStack.getLast().transform);
    } finally {
      transform.dispose();
    }
  }

  // This is cumulative, meaning that the new clip will be the intersection of the current clip and
  // the passed in rectangle. If the final clip has zero area, the runnable is not invoked.
  public void withClip(double x, double y, double w, double h, Runnable run) {
    Area old = getClip();
    Area clip = old.intersect(x, y, w, h);
    if (clip.isEmpty()) {
      return;
    }

    gc.setClipping(rect(clip.x, clip.y, clip.w, clip.h));
    transformStack.add(new TransformAndClip(transformStack.getLast().transform, clip));
    run.run();
    transformStack.removeLast();
    gc.setClipping(rect(old.x, old.y, old.w, old.h));
  }

  public void trace(String label, Runnable run) {
    long start = System.nanoTime();
    try {
      run.run();
    } finally {
      long delta = System.nanoTime() - start;
      traces.merge(label, delta, (a, b) -> a + delta);
    }
  }

  public Map<String, Long> getTraces() {
    return traces;
  }

  protected static int scale(double x) {
    return (int)Math.round(x * scale);
  }

  private static Rectangle rect(double x, double y, double w, double h) {
    return new Rectangle(scale(x), scale(y), scale(w), scale(h));
  }

  public class Path {
    protected final org.eclipse.swt.graphics.Path path;

    protected Path(org.eclipse.swt.graphics.Path path) {
      this.path = path;
    }

    public void moveTo(double x, double y) {
      path.moveTo((float)(x * scale), (float)(y * scale));
    }

    public void lineTo(double x, double y) {
      path.lineTo((float)(x * scale), (float)(y * scale));
    }

    public void circle(double cx, double cy, double r) {
      float d = (float)(2 * r * scale);
      path.addArc((float)((cx - r) * scale), (float)((cy - r) * scale), d, d, 0, 360);
    }

    public void close() {
      path.close();
    }
  }

  public static class Global implements Fonts.TextMeasurer {
    private final Theme theme;
    private final ColorCache colors;
    private final Fonts.Context fontContext;

    public Global(Theme theme, Control owner) {
      this.theme = theme;
      this.colors = new ColorCache(owner.getDisplay());
      this.fontContext = new Fonts.Context(owner);
    }

    public RenderContext newContext(GC gc) {
      return new RenderContext(theme, gc, colors, fontContext);
    }

    public Color getColor(RGBA rgba) {
      return colors.get(rgba);
    }

    @Override
    public Size measure(Style style, String text) {
      return fontContext.measure(style, text);
    }

    @Override
    public double getAscent(Style style) {
      return fontContext.getAscent(style);
    }

    @Override
    public double getDescent(Style style) {
      return fontContext.getDescent(style);
    }

    public void dispose() {
      colors.dispose();
      fontContext.dispose();
    }
  }

  // We maintain our own clip rectangle due MacOS performance and GTK bug:
  // https://bugs.eclipse.org/bugs/show_bug.cgi?id=545226
  private static class TransformAndClip {
    public final Transform transform;
    public final Area clip;

    public TransformAndClip(Transform transform, Area clip) {
      this.transform = transform;
      this.clip = clip;
    }

    public void dispose() {
      transform.dispose();
    }
  }

  private static class Overlay {
    public final Transform transform;
    public final Runnable renderer;

    public Overlay(Transform transform, Runnable renderer) {
      this.transform = transform;
      this.renderer = renderer;
    }

    public void dispose() {
      transform.dispose();
    }
  }
}
