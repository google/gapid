/*
 * Copyright (C) 2022 Google Inc.
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

import static com.google.common.base.CharMatcher.whitespace;
import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.common.base.CharMatcher;
import com.google.common.base.Splitter;
import com.google.common.collect.Lists;

import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Rectangle;

import java.util.List;

public abstract class Tooltip {
  private static final Splitter LINE_SPLITTER =
      Splitter.on(CharMatcher.anyOf("\r\n")).omitEmptyStrings().trimResults();
  private static final int MAX_WIDTH = 400;
  private static final double PADDING = 4;

  private final Area area;

  public Tooltip(Area area) {
    this.area = area;
  }

  public Area getArea() {
    return area;
  }

  public void render(RenderContext ctx) {
    ctx.setBackgroundColor(colors().hoverBackground);
    ctx.fillRect(area.x, area.y, area.w, area.h);
    ctx.setForegroundColor(colors().panelBorder);
    ctx.drawRect(area.x, area.y, area.w - 1, area.h - 1);

    ctx.setForegroundColor(colors().textMain);
    ctx.withTranslation(area.x + PADDING, area.y + PADDING, () -> renderContents(ctx));
  }

  protected abstract void renderContents(RenderContext ctx);

  public static Tooltip forText(Fonts.TextMeasurer m, String text, LocationComputer lc) {
    TextBuilder builder = new TextBuilder(m.measure(Fonts.Style.Normal, " "));
    for (String paragraph : LINE_SPLITTER.split(text)) {
      Fonts.Style style = Fonts.Style.Normal;
      if (paragraph.startsWith("\\b")) {
        style = Fonts.Style.Bold;
        paragraph = paragraph.substring(2);
      }
      builder.addParagraph(m, paragraph, style);
    }
    return builder.build(lc);
  }

  public static Tooltip forFormattedText(Fonts.TextMeasurer m, String text, LocationComputer lc) {
    TextBuilder builder = new TextBuilder(Size.ZERO);
    for (String line : LINE_SPLITTER.split(text)) {
      Fonts.Style style = Fonts.Style.Normal;
      if (line.startsWith("\\b")) {
        style = Fonts.Style.Bold;
        line = line.substring(2);
      }
      builder.addLine(line, style, m.measure(style, line), false);
    }
    return builder.build(lc);
  }

  public static Tooltip forImage(Image image, LocationComputer lc) {
    Rectangle bounds = image.getBounds();
    return new Tooltip(lc.getLocation(bounds.width + 2 * PADDING, bounds.height + 2 * PADDING)) {
      @Override
      protected void renderContents(RenderContext ctx) {
        ctx.drawImage(image, 0, 0);
      }
    };
  }

  public interface LocationComputer {
    public Area getLocation(double w, double h);

    public static LocationComputer fixedLocation(double x, double y) {
      return (w, h) -> new Area(x, y, w, h);
    }

    public static LocationComputer standardTooltip(double x, double y, Area area) {
      return (w, h) -> {
        double endX = Math.min(area.x + area.w, x + w);
        double startX = Math.max(area.x, endX - w);
        double endY = Math.min(area.y + area.h, y + h);
        double startY = Math.max(area.y, endY - h);
        return new Area(startX, startY, w, h);
      };
    }

    public static LocationComputer horizontallyCenteredAndConstrained(
        double x, double y, double areaX, double areaWidth) {
      return (w, h) -> {
        double endX = Math.min(areaX + areaWidth, x + w / 2);
        double startX = Math.max(areaX, endX - w);
        return new Area(startX, y, w, h);
      };
    }

    public static LocationComputer verticallyCenteredAndConstrained(
        double x, double y, double areaY, double areaHeight) {
      return (w, h) -> {
        double endY = Math.min(areaY + areaHeight, y + h / 2);
        double startY = Math.max(areaY, endY - h);
        return new Area(x, startY, w, h);
      };
    }
  }

  private static class TextBuilder {
    private final Size empty;
    private double width = 0;
    private double height = 0;
    private List<Line> lines = Lists.newArrayList();

    public TextBuilder(Size empty) {
      this.empty = empty;
    }

    public void addParagraph(Fonts.TextMeasurer m, String paragraph, Fonts.Style style) {
      boolean first = true;
      do {
        Size size = m.measure(style, paragraph);
        if (size.w <= MAX_WIDTH) {
          addLine(paragraph, style, size, first);
          return;
        }

        int guess = (int)(MAX_WIDTH * paragraph.length() / size.w);
        while (guess < paragraph.length() && !whitespace().matches(paragraph.charAt(guess))) {
          guess++;
        }
        size = m.measure(style, paragraph.substring(0, guess));

        if (size.w <= MAX_WIDTH) {
          paragraph = addLineToNextSpace(m, paragraph, style, guess, size, first);
        } else {
          paragraph = addLineToPreviousSpace(m, paragraph, style, guess, size, first);
        }
        first = false;
      } while (!paragraph.isEmpty());
    }

    private String addLineToNextSpace(Fonts.TextMeasurer m, String paragraph, Fonts.Style style,
        int guess, Size size, boolean first) {
      do {
        int next = guess + 1;
        while (next < paragraph.length() && !whitespace().matches(paragraph.charAt(next))) {
          next++;
        }
        Size now = m.measure(style, paragraph.substring(0, next));
        if (now.w <= MAX_WIDTH) {
          guess = next;
          size = now;
        } else {
          break;
        }
      } while (guess < paragraph.length());
      addLine(paragraph.substring(0, guess), style, size, first);
      return paragraph.substring(guess).trim();
    }

    private String addLineToPreviousSpace(Fonts.TextMeasurer m, String paragraph, Fonts.Style style,
        int guess, Size size, boolean first) {
      do {
        int next = guess - 1;
        while (next > 0 && !whitespace().matches(paragraph.charAt(next))) {
          next--;
        }

        if (next == 0) {
          // We have a single word longer than our max width. Blow our limit.
          addLine(paragraph.substring(0, guess), style, size, first);
          return paragraph.substring(guess).trim();
        }

        guess = next;
        size = m.measure(style, paragraph.substring(0, next));
        if (size.w <= MAX_WIDTH) {
          addLine(paragraph.substring(0, guess), style, size, first);
          return paragraph.substring(guess).trim();
        }
      } while (true);
    }

    public void addLine(String line, Fonts.Style style, Size size, boolean addSep) {
      if (!lines.isEmpty() && addSep) {
        lines.add(new Line("", style, height));
        height += empty.h;
      }
      lines.add(new Line(line, style, height));
      width = Math.max(width, size.w);
      height += size.h;
    }

    public Tooltip build(LocationComputer lc) {
      double w = width + 2 * PADDING, h = height + 2 * PADDING;
      Line[] myLines = lines.toArray(Line[]::new);
      return new Tooltip(lc.getLocation(w, h)) {
        @Override
        protected void renderContents(RenderContext ctx) {
          for (Line line : myLines) {
            line.render(ctx);
          }
        }
      };
    }

    private static class Line {
      private final String line;
      private final Fonts.Style style;
      private final double y;

      public Line(String line, Fonts.Style style, double y) {
        this.line = line;
        this.y = y;
        this.style = style;
      }

      public void render(RenderContext ctx) {
        if (!line.isEmpty()) {
          ctx.drawText(style, line, 0, y);
        }
      }
    }
  }
}
