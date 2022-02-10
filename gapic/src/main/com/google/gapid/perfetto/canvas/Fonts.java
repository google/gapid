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

import static com.google.gapid.perfetto.canvas.RenderContext.scale;

import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StyleRange;
import org.eclipse.swt.graphics.Device;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.FontData;
import org.eclipse.swt.graphics.FontMetrics;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.TextLayout;
import org.eclipse.swt.widgets.Control;

import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

public class Fonts {
  protected static final Logger LOG = Logger.getLogger(Fonts.class.getName());

  private Fonts() {
  }

  public static enum Style {
    Normal, Bold;
  }

  public static interface TextMeasurer {
    public Size measure(Style style, String text);
    public Size measure(TextLayout layout);
    public double getAscent(Style style);
    public double getDescent(Style style);
    public int getOffset(TextLayout layout, double x, double y);
  }

  public static interface FontContext {
    public TextLayout newTextLayout();
    public void applyStyle(TextLayout layout, StyleRange range);
  }

  public static class Context implements TextMeasurer, FontContext {
    private final FontAndGC[] fonts = new FontAndGC[Style.values().length];
    private final Cache<SizeCacheKey, Size> textExtentCache = CacheBuilder.newBuilder()
        .softValues()
        .recordStats()
        .build();

    public Context(Control owner) {
      this.fonts[Style.Normal.ordinal()] = FontAndGC.get(owner, SWT.NORMAL);
      this.fonts[Style.Bold.ordinal()] = FontAndGC.get(owner, SWT.BOLD);
    }

    @Override
    public Size measure(Style style, String text) {
      try {
        return textExtentCache.get(new SizeCacheKey(style, text),
            () -> fonts[style.ordinal()].measure(text));
      } catch (ExecutionException e) {
        throw new RuntimeException(e.getCause());
      }
    }

    @Override
    public Size measure(TextLayout layout) {
      return Size.of(layout.getBounds(), 1 / scale);
    }

    @Override
    public double getAscent(Style style) {
      return fonts[style.ordinal()].getAscent();
    }

    @Override
    public double getDescent(Style style) {
      return fonts[style.ordinal()].getDescent();
    }

    @Override
    public int getOffset(TextLayout layout, double x, double y) {
      return layout.getOffset(scale(x), scale(y), null);
    }

    @Override
    public TextLayout newTextLayout() {
      return fonts[Style.Normal.ordinal()].newTextLayout();
    }

    @Override
    public void applyStyle(TextLayout layout, StyleRange style) {
      if (style.font == null && style.fontStyle == SWT.BOLD) {
        fonts[Style.Bold.ordinal()].updateStyle(style);
      }
      layout.setStyle(style, style.start, style.start + style.length - 1);
    }

    public void setFont(GC gc, Style style) {
      fonts[style.ordinal()].apply(gc);
    }

    public void dispose() {
      for (FontAndGC fgc : fonts) {
        fgc.dispose();
      }

      LOG.log(Level.FINE, "Text extent cache stats: {0}", textExtentCache.stats());
      textExtentCache.invalidateAll();
    }

    private static class FontAndGC {
      private final Font font;
      private final GC gc;
      private final double ascent, descent;

      private FontAndGC(Font font, GC gc, double ascent, double descent) {
        this.font = font;
        this.gc = gc;
        this.ascent = ascent;
        this.descent = descent;
      }

      public static FontAndGC get(Control owner, int style) {
        Font font = scaleFont(owner.getDisplay(), owner.getFont(), style);
        GC gc = new GC(owner.getDisplay());
        gc.setFont(font);
        FontMetrics fm = gc.getFontMetrics();
        return new FontAndGC(font, gc, fm.getAscent() / scale, fm.getDescent() / scale);
      }

      public Size measure(String text) {
        return Size.of(gc.textExtent(text, SWT.DRAW_TRANSPARENT | SWT.DRAW_DELIMITER), 1 / scale);
      }

      public double getAscent() {
        return ascent;
      }

      public double getDescent() {
        return descent;
      }

      public void apply(GC target) {
        target.setFont(font);
      }

      public TextLayout newTextLayout() {
        TextLayout layout = new TextLayout(font.getDevice());
        layout.setFont(font);
        return layout;
      }

      public void updateStyle(StyleRange style) {
        style.font = font;
      }

      public void dispose() {
        font.dispose();
        gc.dispose();
      }

      private static Font scaleFont(Device display, Font font, int style) {
        FontData[] fds = font.getFontData();
        for (FontData fd : fds) {
          fd.setHeight(scale(fd.getHeight()));
          fd.setStyle(style);
        }
        return new Font(display, fds);
      }
    }

    private static class SizeCacheKey {
      public final Style style;
      public final String text;

      public SizeCacheKey(Style style, String text) {
        this.style = style;
        this.text = text;
      }

      @Override
      public int hashCode() {
        return style.hashCode() ^ text.hashCode();
      }

      @Override
      public boolean equals(Object obj) {
        if (obj == this) {
          return true;
        } else if (!(obj instanceof SizeCacheKey)) {
          return false;
        }
        SizeCacheKey o = (SizeCacheKey)obj;
        return style == o.style && text.equals(o.text);
      }
    }
  }
}
