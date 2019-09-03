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
    Normal;
  }

  public static interface TextMeasurer {
    public Size measure(Style style, String text);
  }

  public static class Context implements TextMeasurer {
    private final Font font;
    private final GC textGC;
    private final Cache<String, Size> textExtentCache = CacheBuilder.newBuilder()
        .softValues()
        .recordStats()
        .build();

    public Context(Control owner) {
      this.font = scaleFont(owner.getDisplay(), owner.getFont());
      this.textGC = new GC(owner.getDisplay());
      textGC.setFont(font);
    }

    @Override
    public Size measure(Style style, String text) {
      assert style == Style.Normal;
      try {
        return textExtentCache.get(text, () -> Size.of(textGC.stringExtent(text), 1 / scale));
      } catch (ExecutionException e) {
        throw new RuntimeException(e.getCause());
      }
    }

    public void setFont(GC gc, Style style) {
      assert style == Style.Normal;
      gc.setFont(font);
    }

    public void dispose() {
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
}
