package com.google.gapid.perfetto.canvas;

import com.google.common.base.Preconditions;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.gapid.widgets.Theme;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Device;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.FontData;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.Transform;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.widgets.Control;

import java.util.LinkedList;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Handles all the drawing operations.
 */
public class RenderContext implements Panel.TextMeasurer, AutoCloseable {
  protected static final Logger LOG = Logger.getLogger(RenderContext.class.getName());

  private static final int textSizeGreediness = 3; // must be >= 1.
  protected static final double scale = DPIUtil.autoScaleDown(8f);

  public final Theme theme;

  private final GC gc;
  private final ColorCache colors;
  private final Panel.TextMeasurer textMeasurer;
  private final LinkedList<TransformAndClip> transformStack = Lists.newLinkedList();
  private final Map<String, Long> traces = Maps.newHashMap();

  public RenderContext(
      Theme theme, GC gc, ColorCache colors, Panel.TextMeasurer textMeasurer) {
    this.theme = theme;
    this.gc = gc;
    this.colors = colors;
    this.textMeasurer = textMeasurer;

    Area clip = Area.of(gc.getClipping());
    Transform transform = new Transform(gc.getDevice());
    gc.getTransform(transform);
    transform.scale((float)(1 / scale), (float)(1 / scale));
    gc.setTransform(transform);
    transformStack.push(new TransformAndClip(transform, clip));
    gc.setLineWidth((int)scale);
  }

  @Override
  public Size measure(String text) {
    return textMeasurer.measure(text);
  }

  @Override
  public void close() {
    Preconditions.checkState(
        transformStack.size() == 1, "transform stack size != 1: %s", transformStack.size());
    for (TransformAndClip t : transformStack) {
      t.dispose();
    }
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

  public void drawRect(double x, double y, double w, double h) {
    gc.drawRectangle(rect(x, y, w, h));
  }

  public void fillRect(double x, double y, double w, double h) {
    gc.fillRectangle(rect(x, y, w, h));
  }

  public void drawCircle(double cx, double cy, double r) {
    int d = 2 * scale(r);
    gc.drawOval(scale(cx - r), scale(cy - r), d, d);
  }

  // x, y is top left corner of text.
  public void drawText(String text, double x, double y) {
    gc.drawText(text, scale(x), scale(y), SWT.DRAW_TRANSPARENT);
  }

  // draws text centered vertically
  public void drawText(String text, double x, double y, double h) {
    drawText(text, x, y + (h - textMeasurer.measure(text).h) / 2);
  }

  // draws the text centered horizontally and vertically, truncated to fit into the given width.
  public void drawText(String text, double x, double y, double w, double h)  {
    drawText(text, x, y, w, h, true);
  }

  // draws the text centered horizontally and vertically, only if it fits.
  public void drawTextIfFits(String text, double x, double y, double w, double h) {
    drawText(text, x, y, w, h, false);
  }

  private void drawText(String text, double x, double y, double w, double h, boolean truncate) {
    String toDisplay = text;
    for (int l = text.length(); ; ) {
      Size size = textMeasurer.measure(toDisplay);
      if (size.w < w) {
        drawText(toDisplay, x + (w - size.w) / 2 , y + (h - size.h) / 2);
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

  // draws the text centered vertically and left truncated to fit into the given width.
  public void drawTextLeftTruncate(String text, double x, double y, double w, double h) {
    String toDisplay = text;
    for (int l = text.length(); ; ) {
      Size size = textMeasurer.measure(toDisplay);
      if (size.w < w) {
        drawText(toDisplay, x, y + (h - size.h) / 2);
        break;
      }

      l = Math.min(l - textSizeGreediness, (int)(w / (size.w / toDisplay.length())));
      if (l <= 0) {
        break;
      }
      toDisplay = "..." + text.substring(text.length() - l);
    }
  }

  // draws the text centered vertically and on the left of x.
  public void drawTextRightJustified(String text, double x, double y, double h)  {
    Size size = textMeasurer.measure(text);
    drawText(text, x - size.w, y + (h - size.h) / 2);
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

  public static class Global implements Panel.TextMeasurer {
    private final Theme theme;
    private final ColorCache colors;
    private final Font font;
    private final GC textGC;
    private final Cache<String, Size> textExtentCache = CacheBuilder.newBuilder()
        .softValues()
        .recordStats()
        .build();

    public Global(Theme theme, Control owner) {
      this.theme = theme;
      this.colors = new ColorCache(owner.getDisplay());
      this.font = scaleFont(owner.getDisplay(), owner.getFont());
      this.textGC = new GC(owner.getDisplay());
      textGC.setFont(font);
    }

    @Override
    public Size measure(String text) {
      try {
        return textExtentCache.get(text, () -> Size.of(textGC.stringExtent(text), 1 / scale));
      } catch (ExecutionException e) {
        throw new RuntimeException(e.getCause());
      }
    }

    public RenderContext newContext(GC gc) {
      gc.setFont(font);
      return new RenderContext(theme, gc, colors, this);
    }

    public void dispose() {
      colors.dispose();
      font.dispose();
      textGC.dispose();

      LOG.log(Level.FINE, "Text extent cache stats: {0}", textExtentCache.stats());
      textExtentCache.invalidateAll();
    }

    private static Font scaleFont(Device display, Font font) {
      FontData[] fds = font.getFontData();
      for (FontData fd : fds) {
        fd.setHeight(scale(fd.getHeight()));
      }
      return new Font(display, fds);
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
}
