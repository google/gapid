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
    return "CPU " + (track.getCpu() + 1) + " Frequency";
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("CpuFrequencyPanel", () -> {
      CpuFrequencyTrack.Data data = track.getData(state, () -> {
        repainter.repaint(new Area(0, 0, width, height));
      });
      drawLoading(ctx, data, state, h);

      if (data == null || data.tsStarts.length == 0) {
        return;
      }

      TimeSpan visible = state.getVisibleTime();
      double startPx = state.timeToPx(visible.start);
      double endPx = state.timeToPx(visible.end);

      final String[] kUnits = new String[] { "", "K", "M", "G", "T", "E" };
      double exp = Math.ceil(Math.log10(Math.max(data.maximumValue, 1)));
      double pow10 = Math.pow(10, exp);
      double yMax = Math.ceil(data.maximumValue / (pow10 / 4)) * (pow10 / 4);
      int unitGroup = (int)Math.floor(exp / 3);
      // The values we have for cpufreq are in kHz so +1 to unitGroup.
      String yLabel = (yMax / Math.pow(10, unitGroup * 3)) + " " + kUnits[unitGroup + 1] + "Hz";

      // Draw the CPU frequency graph.
      ctx.setBackgroundColor(StyleConstants.Palette.getColor(track.getCpu(), 3));
      ctx.setForegroundColor(StyleConstants.Palette.getColor(track.getCpu()));
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
        ctx.setBackgroundColor(StyleConstants.Palette.getColor(track.getCpu(), 3));
        ctx.setForegroundColor(StyleConstants.Palette.getColor(track.getCpu()));

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
          path.circle(xStart, y, 3);
          ctx.fillPath(path);
          ctx.drawPath(path);
        });

        // Draw the tooltip.
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(mouseXpos + 5, 0, textSize.w + 16, h);
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Normal, hoverLabel, mouseXpos + 5 + 8, (h - 2 * textSize.h) / 4);
        if (hoveredIdle != null && hoveredIdle != -1) {
          String idle = "Idle: " + (hoveredIdle + 1);
          ctx.drawText(Fonts.Style.Normal, idle, mouseXpos + 5 + 8, (3 * h - 2 * textSize.h) / 4);
        }
      }

      // Write the Y scale on the top left corner.
      Size labelSize = ctx.measure(Fonts.Style.Normal, yLabel);
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(0, 0, labelSize.w + 8, labelSize.h + 8);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, yLabel, 4, 4);
    });
  }

  @Override
  public Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y) {
    CpuFrequencyTrack.Data data = track.getData(state, () -> { /* nothing */ });
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

        double xStart = Math.floor(state.timeToPx(hoveredTs)) - 4;
        double xEnd = Math.max(hoveredTsEnd == null ?
            state.timeToPx(state.getVisibleTime().end) : Math.floor(state.timeToPx(hoveredTsEnd)),
            x + m.measure(Fonts.Style.Normal, hoverLabel).w + 21);

        return new Hover() {
          @Override
          public Area getRedraw() {
            return new Area(xStart, 0, xEnd - xStart, HEIGHT);
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
