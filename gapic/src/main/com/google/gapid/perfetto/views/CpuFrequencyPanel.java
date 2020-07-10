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

import static com.google.gapid.perfetto.views.Loading.drawLoading;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.StyleConstants.gradient;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.CpuFrequencyTrack;

/**
 * Draws the CPU frequency and idle graph.
 */
public class CpuFrequencyPanel extends TrackPanel<CpuFrequencyPanel> {
  private static final double HEIGHT = 30;
  private static final double CURSOR_SIZE = 5;
  private static final double HOVER_PADDING = 8;
  private static final double HOVER_MARGIN = 10;

  private final CpuFrequencyTrack track;

  private double mouseXpos = 0;
  protected Integer hoveredValue = null;
  protected Long hoveredTs = null;
  protected Long hoveredTsEnd = null;
  protected Byte hoveredIdle = null;
  protected String hoverLabel = null;

  public CpuFrequencyPanel(State state, CpuFrequencyTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public CpuFrequencyPanel copy() {
    return new CpuFrequencyPanel(state, track);
  }

  @Override
  public String getTitle() {
    return "CPU " + track.getCpu().id + " Frequency";
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("CpuFrequencyPanel", () -> {
      CpuFrequencyTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null || data.tsStarts.length == 0) {
        return;
      }

      TimeSpan visible = state.getVisibleTime();
      double startPx = state.timeToPx(visible.start);
      double endPx = state.timeToPx(visible.end);

      final String[] kUnits = new String[] { "", "K", "M", "G", "T", "E" };
      double exp = Math.ceil(Math.log10(Math.max(track.getCpu().maxFreq, 1)));
      double pow10 = Math.pow(10, exp);
      double yMax = Math.ceil(track.getCpu().maxFreq / (pow10 / 4)) * (pow10 / 4);
      int unitGroup = (int)Math.floor(exp / 3);
      // The values we have for cpufreq are in kHz so +1 to unitGroup.
      String yLabel = (yMax / Math.pow(10, unitGroup * 3)) + " " + kUnits[unitGroup + 1] + "Hz";

      // Draw the CPU frequency graph.
      gradient(track.getCpu().id).applyBaseAndBorder(ctx);
      ctx.path(path -> {
        double lastX = startPx, lastY = h;
        path.moveTo(lastX, lastY);
        for (int i = 0; i < data.freqKHz.length; i++) {
          double x = state.timeToPx(data.tsStarts[i]);
          double y = (1 - data.freqKHz[i] / yMax) * h;
          if (y == lastY) {
            continue;
          }
          lastX = x;
          path.lineTo(lastX, lastY);
          path.lineTo(lastX, y);
          lastY = y;
        }
        // Find the end time for the last frequency event and then draw
        // down to zero to show that we do not have data after that point.
        long endTime = data.tsEnds[data.tsEnds.length - 1];
        double finalX = state.timeToPx(endTime);
        path.lineTo(finalX, lastY);
        path.lineTo(finalX, h);
        path.lineTo(endPx, h);
        path.close();
        ctx.fillPath(path);
        ctx.drawPath(path);
      });

      // Draw CPU idle rectangles that overlay the CPU freq graph.
      ctx.setBackgroundColor(colors().cpuFreqIdle);
      for (int i = 0; i < data.freqKHz.length; i++) {
        if (data.idles[i] >= 0) {
          double firstX = state.timeToPx(data.tsStarts[i]);
          double secondX = state.timeToPx(data.tsEnds[i]);
          double idleW = secondX - firstX;
          if (idleW < 0.5) {
            continue;
          }
          double lastY = (1 - data.freqKHz[i] / yMax) * h;
          ctx.fillRect(firstX, h, idleW, lastY - h);
        }
      }

      if (hoveredValue != null && hoveredTs != null) {
        gradient(track.getCpu().id).applyBaseAndBorder(ctx);

        Size textSize = ctx.measure(Fonts.Style.Normal, hoverLabel);
        double xStart = Math.floor(state.timeToPx(hoveredTs));
        double xEnd = hoveredTsEnd == null ? endPx : Math.floor(state.timeToPx(hoveredTsEnd));
        double y = (1 - hoveredValue / yMax) * h;

        // Highlight line.
        ctx.path(path -> {
          path.moveTo(xStart, y);
          path.lineTo(xEnd, y);
          //ctx.setLineWidth(3);
          ctx.drawPath(path);
          //ctx.setLineWidth(1);
        });

        // Draw change marker.
        ctx.path(path -> {
          path.circle(xStart, y, CURSOR_SIZE / 2);
          ctx.fillPath(path);
          ctx.drawPath(path);
        });

        // Draw the tooltip.
        // Technically the cursor is always drawn at the beginning of the sample, and hence when the
        // track is not in quantized view, there's no cursor drawn near to the hover point most of
        // the time. Hover, to be consistent with other cases, always assume the cursor is drawn at
        // the hover point.
        double cardX = mouseXpos + CURSOR_SIZE / 2 + HOVER_MARGIN;
        if (cardX >= w - (textSize.w + 2 * HOVER_PADDING)) {
          cardX = mouseXpos - CURSOR_SIZE / 2 - HOVER_MARGIN - textSize.w - 2 * HOVER_PADDING;
        }
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(cardX, 0, textSize.w + 2 * HOVER_PADDING, h);
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Normal, hoverLabel, cardX + HOVER_PADDING, (h - 2 * textSize.h) / 4);
        if (hoveredIdle != null && hoveredIdle != -1) {
          String idle = "Idle: " + (hoveredIdle + 1);
          ctx.drawText(Fonts.Style.Normal, idle, cardX + HOVER_PADDING, (3 * h - 2 * textSize.h) / 4);
        }
      }

      // Write the Y scale on the top left corner.
      Size labelSize = ctx.measure(Fonts.Style.Normal, yLabel);
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(0, 0, labelSize.w + HOVER_PADDING, labelSize.h + HOVER_PADDING);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, yLabel, 4, 4);
    });
  }

  @Override
  public Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y, int mods) {
    CpuFrequencyTrack.Data data = track.getData(state.toRequest(), onUiThread());
    if (data == null) {
      return Hover.NONE;
    }

    mouseXpos = x;
    long time = state.pxToTime(x);
    for (int i = 0; i < data.tsStarts.length; i++) {
      if (data.tsStarts[i] <= time && time <= data.tsEnds[i]) {
        hoveredTs = data.tsStarts[i];
        hoveredTsEnd = data.tsEnds[i];
        hoveredValue = data.freqKHz[i];
        hoveredIdle = data.idles[i];
        hoverLabel = String.format(
            "%s: %,dkHz", (data.quantized ? "Average Freq" : "Freq"), hoveredValue);

        double startX = Math.floor(state.timeToPx(hoveredTs)) - 4;
        double endX = hoveredTsEnd == null ?
                      state.timeToPx(state.getVisibleTime().end)
                      : Math.floor(state.timeToPx(hoveredTsEnd));

        return new Hover() {
          @Override
          public Area getRedraw() {
            // The area of the sample, especially when in steady performance or idle state, can be
            // larger than the hover card. Hence the redraw area needs to at least cover the whole
            // sample because when we move between samples, the highlight of the previous sample
            // needs to be redrawn without highlight. If the redraw area only covers the hover card
            // then the area that is not intersected with the hover card will not be redrawn, and
            // hence the highlight remains in that area.

            // First, calculate the default left boundary x-axis coordinate and width of the
            // redrawn area.
            double defaultX = x - CURSOR_SIZE / 2;
            double defaultW = CURSOR_SIZE + HOVER_MARGIN + HOVER_PADDING
                              + m.measure(Fonts.Style.Normal, hoverLabel).w + HOVER_PADDING;

            // Assuming the hover card is drawn on the right of the hover point.

            // Determine the x-axis coordinate of the left boundary of the redrawn area.
            double redrawLx = Math.min(defaultX, startX);

            // Determine the x-axis coordinate of the right boundary of the redrawn area by
            // comparing the right end of the sample with the right boundary of the default
            // redrawn area.
            double redrawRx = Math.max(defaultX + defaultW, endX);

            // Calculate the real redrawn width.
            double redrawW = redrawRx - redrawLx;

            if (defaultX >= state.getWidth() - defaultW) {

              // re-calculate the left boundary of the redrawn area by comparing the start end of
              // the sample with the left boundary of the default redrawn area when the hover card
              // is drawn on the left side of the hover point.
              redrawLx = Math.min(startX, x + CURSOR_SIZE / 2 - defaultW);

              // re-calculate the right boundary of the redrawn area by comparing the end of the
              // sample with the right boundary of the default redrawn area when the hover card is
              // drawn on the left side of the hover point.
              redrawRx = Math.max(x + CURSOR_SIZE / 2, endX);

              // Finally, re-calculate the redrawn width, plus radius of the cursor to avoid the
              // precision issue at the right edge of the cursor.
              redrawW = redrawRx - redrawLx + CURSOR_SIZE / 2;
            }

            return new Area(redrawLx, 0, redrawW, HEIGHT);
          }

          @Override
          public void stop() {
            hoveredTs = null;
            hoveredTsEnd = null;
            hoveredValue = null;
            hoveredIdle = null;
          }
        };
      }
    }

    return Hover.NONE;
  }
}
