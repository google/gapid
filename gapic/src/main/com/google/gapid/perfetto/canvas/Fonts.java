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
import org.eclipse.swt.graphics.Device;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.FontData;
import org.eclipse.swt.graphics.GC;
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
  }

  public static class Context implements TextMeasurer {
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

      private FontAndGC(Font font, GC gc) {
        this.font = font;
        this.gc = gc;
      }

      public static FontAndGC get(Control owner, int style) {
        Font font = scaleFont(owner.getDisplay(), owner.getFont(), style);
        GC gc = new GC(owner.getDisplay());
        gc.setFont(font);
        return new FontAndGC(font, gc);
      }

      public Size measure(String text) {
        return Size.of(gc.stringExtent(text), 1 / scale);
      }

      public void apply(GC target) {
        target.setFont(font);
      }

      public void dispose() {
        font.dispose();
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
