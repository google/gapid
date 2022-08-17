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
package com.google.gapid.perfetto.views;

import static com.google.gapid.perfetto.views.Loading.drawLoading;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.StyleConstants.mainGradient;

import com.google.gapid.perfetto.Unit;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.PowerSummaryTrack;
import com.google.gapid.perfetto.models.Selection;

public class PowerSummaryPanel extends TrackPanel<PowerSummaryPanel> {
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static Unit unit;

  protected final PowerSummaryTrack track;
  protected static final double TRACK_HEIGHT = 80;
  protected HoverCard hovered = null;

  public PowerSummaryPanel(State state, PowerSummaryTrack track) {
    super(state);
    this.track = track;
    this.unit = track.unit;
  }

  private double calculateYCoordinate(double value) {
    double range =
        (track.minValue == track.maxValue && track.minValue == 0)
            ? 1
            : (track.maxValue - track.minValue);
    return (TRACK_HEIGHT - 1) * (1 - (value - track.minValue) / range);
  }

  @Override
  public PowerSummaryPanel copy() {
    return new PowerSummaryPanel(state, track);
  }

  @Override
  public String getTitle() {
    return "Power Usage";
  }

  @Override
  public String getSubTitle() {
    return track.getNumPowerRailTracks() + " power rail tracks";
  }

  @Override
  public String getTooltip() {
    return "Total Power Usage";
  }

  @Override
  public double getHeight() {
    return TRACK_HEIGHT;
  }

  @Override
  protected void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace(
        "PowerSummary",
        () -> {
          PowerSummaryTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
          drawLoading(ctx, data, state, h);

          if (data == null || data.ts.length == 0) {
            return;
          }

          Selection<?> selected = state.getSelection(Selection.Kind.Counter);
          mainGradient().applyBaseAndBorder(ctx);
          ctx.path(
              path -> {
                double lastX = state.timeToPx(data.ts[0]);
                double lastY = h;
                path.moveTo(lastX, lastY);
                for (int i = 0; i < data.ts.length; i++) {
                  double nextX = state.timeToPx(data.ts[i]);
                  double nextY = calculateYCoordinate(data.values[i]);
                  path.lineTo(nextX, lastY);
                  path.lineTo(nextX, nextY);
                  lastX = nextX;
                  lastY = nextY;
                }

                path.lineTo(lastX, h);
                ctx.fillPath(path);
                ctx.drawPath(path);
              });

          if (hovered != null) {
            double y = calculateYCoordinate(hovered.value);
            ctx.setBackgroundColor(mainGradient().highlight);
            ctx.fillRect(hovered.startX, y - 1, hovered.endX - hovered.startX, TRACK_HEIGHT - y + 1);
            ctx.setForegroundColor(colors().textMain);
            ctx.drawCircle(hovered.mouseX, y, CURSOR_SIZE / 2);

            ctx.setBackgroundColor(colors().hoverBackground);
            double bgH = Math.max(hovered.size.h, TRACK_HEIGHT);
            // The left x-axis coordinate of HoverCard.
            double hx = hovered.mouseX + CURSOR_SIZE / 2 + HOVER_MARGIN;
            if (hx >= w - (2 * HOVER_PADDING + hovered.size.w)) {
              hx =
                  hovered.mouseX
                      - CURSOR_SIZE / 2
                      - HOVER_MARGIN
                      - 2 * HOVER_PADDING
                      - hovered.size.w;
            }
            ctx.fillRect(
                hx, Math.min((TRACK_HEIGHT - bgH) / 2, 0), 2 * HOVER_PADDING + hovered.size.w, bgH);
            ctx.setForegroundColor(colors().textMain);
            // The left x-axis coordinate of the left labels.
            double x = hx + HOVER_PADDING;
            y = (TRACK_HEIGHT - hovered.size.h) / 2;
            // The difference between the x-axis coordinate of the left labels and the right labels.
            double dx = hovered.leftWidth + HOVER_PADDING, dy = hovered.size.h / 2;
            ctx.drawText(Fonts.Style.Normal, "Value:", x, y);
            ctx.drawText(Fonts.Style.Normal, hovered.minLabel, x + dx, y);
            ctx.drawText(Fonts.Style.Normal, hovered.maxLabel, x + dx, y + dy);

            x = hx + HOVER_PADDING + hovered.leftWidth;
            ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.valueLabel, x, y, dy);

            x = hx + HOVER_PADDING + hovered.size.w;
            ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.min, x, y, dy);
            ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.max, x, y + dy, dy);
          }
        });
  }

  @Override
  protected Hover onTrackMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    PowerSummaryTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
    if (data == null || data.ts.length == 0) {
      return Hover.NONE;
    }

    long time = state.pxToTime(x);
    if (time < data.ts[0]) {
      return Hover.NONE;
    }

    int idx = 0;
    for (; idx < data.ts.length - 1; idx++) {
      if (data.ts[idx + 1] > time) {
        break;
      }
    }

    if (idx >= data.ts.length) {
      return Hover.NONE;
    }

    double startX = state.timeToPx(data.ts[idx]);
    double endX =
        (idx >= data.ts.length - 1)
            ? startX
            : state.timeToPx(data.ts[idx + 1]); // Moving endX to startX when
    hovered = new HoverCard(m, track.minValue, track.maxValue, data.values[idx], startX, endX, x);

    return new Hover() {
      @Override
      public Area getRedraw() {
        // The area of the sample can be larger than the hover card. Hence the redraw area needs to
        // at least cover the whole sample because when we move between samples, the highlight of
        // the previous sample needs to be redrawn without highlight. If the redraw area only
        // covers the hover card then the area that is not intersected with the hover card will
        // not be redrawn, and hence the highlight remains in that area.

        // First, calculate the default left boundary x-axis coordinate and width of the
        // redrawn area.
        final double defaultX = hovered.mouseX - CURSOR_SIZE / 2;
        final double defaultW =
            CURSOR_SIZE + HOVER_MARGIN + HOVER_PADDING + hovered.size.w + HOVER_PADDING;

        // Assuming the hover card is drawn on the right of the hover point.

        // Determine the x-axis coordinate of the left boundary of the redrawn area.
        double redrawLx = Math.min(defaultX, startX);

        // Determine the x-axis coordinate of the right boundary of the redrawn area by comparing
        // the right end of the sample with the right boundary of the default redrawn area.
        double redrawRx = Math.max(defaultX + defaultW, endX);

        // Calculate the real redrawn width.
        double redrawW = redrawRx - redrawLx;

        if (defaultX >= state.getWidth() - defaultW) {

          // Re-calculate the left boundary of the redrawn area by comparing the start of the
          // sample with the left boundary of the default redrawn area when the hover card is drawn
          // on the left side of the hover point.
          redrawLx = Math.min(startX, hovered.mouseX + CURSOR_SIZE / 2 - defaultW);

          // Re-calculate the right boundary of the redrawn area by comparing the end of the sample
          // with the right boundary of the default redrawn area when the hover card is drawn on
          // the left side of the hover point.
          redrawRx = Math.max(hovered.mouseX + CURSOR_SIZE / 2, endX);

          // Finally, re-calculate the redrawn width, plus radius of the cursor to avoid the
          // precision issue at the right edge of the cursor.
          redrawW = redrawRx - redrawLx + CURSOR_SIZE / 2;
        }

        return new Area(redrawLx, -TRACK_MARGIN, redrawW, TRACK_HEIGHT + 2 * TRACK_MARGIN);
      }

      @Override
      public void stop() {
        hovered = null;
      }
    };
  }

  private static class HoverCard {
    public final double value;
    public final double startX, endX;
    public final double mouseX;
    public final String valueLabel;
    public final String min, max;
    public final String minLabel, maxLabel;
    public final double leftWidth;
    public final Size size;

    public HoverCard(
        Fonts.TextMeasurer tm,
        Double minValue,
        Double maxValue,
        double value,
        double startX,
        double endX,
        double mouseX) {
      this.value = value;
      this.startX = startX;
      this.endX = endX;
      this.mouseX = mouseX;
      this.valueLabel = unit.format(value);
      this.min = unit.format(minValue);
      this.max = unit.format(maxValue);

      this.minLabel = "Trace Min:";
      this.maxLabel = "Trace Max:";

      Size valueSize = tm.measure(Fonts.Style.Normal, valueLabel);
      Size minSize = tm.measure(Fonts.Style.Normal, min);
      Size maxSize = tm.measure(Fonts.Style.Normal, max);

      double leftLabel = tm.measure(Fonts.Style.Normal, "Value:").w;
      double rightLabel =
          Math.max(
                  tm.measure(Fonts.Style.Normal, minLabel).w,
                  tm.measure(Fonts.Style.Normal, maxLabel).w)
              + HOVER_PADDING;
      this.leftWidth = leftLabel + valueSize.w;
      this.size =
          new Size(
              leftWidth + HOVER_PADDING + rightLabel + Math.max(minSize.w, maxSize.w),
              Math.max(valueSize.h, minSize.h + maxSize.h) + HOVER_PADDING);
    }
  }
}
