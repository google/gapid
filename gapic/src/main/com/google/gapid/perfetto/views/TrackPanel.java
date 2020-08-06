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
package com.google.gapid.perfetto.views;

import static com.google.common.base.CharMatcher.whitespace;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.TimelinePanel.drawGridLines;

import com.google.common.base.CharMatcher;
import com.google.common.base.Splitter;
import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.Track;

import java.util.List;
import java.util.function.Consumer;

/**
 * {@link Panel} displaying a {@link Track}.
 */
public abstract class TrackPanel<T extends TrackPanel<T>> extends Panel.Base
    implements TitledPanel, CopyablePanel<T> {
  private static final double HOVER_X_OFF = 10;
  private static final double HOVER_Y_OFF = 7;
  private static final double HOVER_PADDING = 4;

  protected final State state;
  protected Tooltip tooltip;

  public TrackPanel(State state) {
    this.state = state;
  }

  @Override
  public final double getPreferredHeight() {
    return getHeight() + 2 * TRACK_MARGIN;
  }

  public abstract double getHeight();

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    double w = width - LABEL_WIDTH, h = height - 2 * TRACK_MARGIN;
    drawGridLines(ctx, state, LABEL_WIDTH, 0, w, height);
    ctx.withTranslation(LABEL_WIDTH, TRACK_MARGIN, () ->
      ctx.withClip(0, -TRACK_MARGIN, w, h + 2 * TRACK_MARGIN, () ->
        renderTrack(ctx, repainter, w, h)));

    if (tooltip != null) {
      ctx.addOverlay(() -> {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(tooltip.x, tooltip.y,
            tooltip.width + 2 * HOVER_PADDING, tooltip.height + 2 * HOVER_PADDING);
        ctx.setForegroundColor(colors().panelBorder);
        ctx.drawRect(tooltip.x, tooltip.y,
            tooltip.width + 2 * HOVER_PADDING - 1, tooltip.height + 2 * HOVER_PADDING - 1);
        ctx.setForegroundColor(colors().textMain);

        double tx = tooltip.x + HOVER_PADDING, ty = tooltip.y + HOVER_PADDING;
        for (Tooltip.Line line : tooltip.lines) {
          line.render(ctx, tx, ty);
        }
      });
    }
  }

  protected abstract void renderTrack(RenderContext ctx, Repainter repainter, double w, double h);

  @Override
  public void visit(Visitor v, Area area) {
    area.intersect(LABEL_WIDTH, TRACK_MARGIN, width - LABEL_WIDTH, height - 2 * TRACK_MARGIN)
      .ifNotEmpty(a -> v.visit(this, a));
  }

  @Override
  public Hover onMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    if (x < LABEL_WIDTH) {
      String text = getTooltip();
      if (text.isEmpty()) {
        return Hover.NONE;
      }

      tooltip = Tooltip.compute(m, text, x + HOVER_X_OFF, y + HOVER_Y_OFF);
      return new Hover() {
        @Override
        public Area getRedraw() {
          return new Area(tooltip.x, tooltip.y,
              2 * HOVER_PADDING + tooltip.width, 2 * HOVER_PADDING + tooltip.height);
        }

        @Override
        public boolean isOverlay() {
          return true;
        }

        @Override
        public void stop() {
          tooltip = null;
        }
      };
    } else if (y < TRACK_MARGIN || y > height - TRACK_MARGIN) {
      return Hover.NONE;
    }
    return onTrackMouseMove(m, x - LABEL_WIDTH, y - TRACK_MARGIN, mods)
        .translated(LABEL_WIDTH, TRACK_MARGIN);
  }

  protected abstract Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y, int mods);

  // Helper functions for the track.getData(..) calls.
  protected <D> Track.OnUiThread<D> onUiThread() {
    return onUiThread(state, () -> { /* do nothing */ });
  }

  // Helper functions for the track.getData(..) calls.
  protected <D> Track.OnUiThread<D> onUiThread(Repainter repainter) {
    return onUiThread(state, () -> repainter.repaint(new Area(0, 0, width, height)));
  }

  public static <D> Track.OnUiThread<D> onUiThread(State state, Runnable repaint) {
    return new Track.OnUiThread<D>() {
      @Override
      public void onUiThread(ListenableFuture<D> future, Consumer<D> callback) {
        state.thenOnUiThread(future, callback);
      }

      @Override
      public void repaint() {
        repaint.run();
      }
    };
  }

  // Helper function to determine the color of a slice.
  protected StyleConstants.Gradient getSliceColor(String title, int depth) {
    int commaIndex = title.indexOf(',');
    int colorCode = (commaIndex == -1) ? title.hashCode() :
        title.substring(0, commaIndex).hashCode();
    return StyleConstants.gradient(colorCode ^ depth);
  }

  protected StyleConstants.Gradient getSliceColor(String title) {
    return getSliceColor(title, 0);
  }

  private static class Tooltip {
    private static final Splitter LINE_SPLITTER =
        Splitter.on(CharMatcher.anyOf("\r\n")).omitEmptyStrings().trimResults();
    private static final int MAX_WIDTH = 400;

    public final double x, y;
    public final Line[] lines;
    public final double width;
    public final double height;

    public Tooltip(double x, double y, Line[] lines, double width, double height) {
      this.x = x;
      this.y = y;
      this.lines = lines;
      this.width = width;
      this.height = height;
    }

    public static Tooltip compute(Fonts.TextMeasurer m, String text, double x, double y) {
      Builder builder = new Builder(m.measure(Fonts.Style.Normal, " "));
      para: for (String paragraph : LINE_SPLITTER.split(text)) {
        boolean first = true;
        Fonts.Style style = Fonts.Style.Normal;
        if (paragraph.startsWith("\\b")) {
          style = Fonts.Style.Bold;
          paragraph = paragraph.substring(2);
        }

        do {
          Size size = m.measure(style, paragraph);
          if (size.w <= MAX_WIDTH) {
            builder.addLine(paragraph, style, size, first);
            continue para;
          }

          int guess = (int)(MAX_WIDTH * paragraph.length() / size.w);
          while (guess < paragraph.length() && !whitespace().matches(paragraph.charAt(guess))) {
            guess++;
          }
          size = m.measure(style, paragraph.substring(0, guess));

          if (size.w <= MAX_WIDTH) {
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
            builder.addLine(paragraph.substring(0, guess), style, size, first);
            paragraph = paragraph.substring(guess).trim();
            first = false;
          } else {
            do {
              int next = guess - 1;
              while (next > 0 && !whitespace().matches(paragraph.charAt(next))) {
                next--;
              }

              if (next == 0) {
                // We have a single word longer than our max width. Blow our limit.
                builder.addLine(paragraph.substring(0, guess), style, size, first);
                paragraph = paragraph.substring(guess).trim();
                first = false;
                break;
              }

              guess = next;
              size = m.measure(style, paragraph.substring(0, next));
              if (size.w <= MAX_WIDTH) {
                builder.addLine(paragraph.substring(0, guess), style, size, first);
                paragraph = paragraph.substring(guess).trim();
                first = false;
                break;
              }
            } while (true);
          }
        } while (!paragraph.isEmpty());
      }
      return builder.build(x, y);
    }

    public static class Line {
      private final String line;
      private final Fonts.Style style;
      private final double y;

      public Line(String line, Fonts.Style style, double y) {
        this.line = line;
        this.y = y;
        this.style = style;
      }

      public void render(RenderContext ctx, double ox, double oy) {
        if (!line.isEmpty()) {
          ctx.drawText(style, line, ox, oy + y);
        }
      }
    }

    private static class Builder {
      private final Size empty;
      private double width = 0;
      private double height = 0;
      private List<Line> lines = Lists.newArrayList();

      public Builder(Size empty) {
        this.empty = empty;
      }

      public Tooltip build(double x, double y) {
        return new Tooltip(x, y, lines.toArray(new Line[lines.size()]), width, height);
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
    }
  }
}
