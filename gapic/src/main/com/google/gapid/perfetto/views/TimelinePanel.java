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

import static com.google.gapid.perfetto.views.StyleConstants.LABEL_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;

/**
 * Draws a timeline for navigating the trace.
 */
public class TimelinePanel extends Panel.Base {
  private static final double DESIRED_PX_PER_STEP = 160;
  private static final double HEIGHT = 40;
  private static final double MARGIN = 3;
  private static final double TICK_SIZE = 8;
  private static final double LABEL_Y = 5;
  private static final double LEGEND_H = 4;

  private final State state;

  public TimelinePanel(State state) {
    this.state = state;
  }

  @Override
  public double getPreferredHeight() {
    return HEIGHT;
  }

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    ctx.trace("TimelinePanel", () -> {
      TimeSpan visible = state.getVisibleTime();
      TimeSpan trace = state.getTraceTime();

      // TODO: this should be part of the state - dedupe with below.
      long step = getGridStepSize(
          visible.getDuration(), (width - LABEL_WIDTH) / DESIRED_PX_PER_STEP);
      long offset = ((visible.start - trace.start) / step) * step;
      long start = offset + trace.start;

      ctx.setForegroundColor(colors().textMain);
      String label = TimeSpan.timeToString(step / 5);
      Size labelSize = ctx.measure(Fonts.Style.Normal, label);
      double labelX = Math.floor(LABEL_WIDTH - MARGIN - labelSize.w);
      double legendX = labelX - 2 * MARGIN;
      double legendY = Math.floor(LABEL_Y + labelSize.h - ctx.getDescent(Fonts.Style.Normal));
      double legendW = state.durationToDeltaPx(step / 5);
      ctx.drawText(Fonts.Style.Normal, label, labelX, LABEL_Y);

      ctx.withClip(0, 0, legendX - legendW - MARGIN, height, () -> {
        String duration = "Total Time: " + TimeSpan.timeToShortString(trace.getDuration());
        ctx.drawText(Fonts.Style.Normal, duration, 2 * MARGIN, LABEL_Y);
      });


      ctx.setForegroundColor(colors().timelineRuler);

      ctx.drawLine(legendX, legendY, legendX, legendY - LEGEND_H);
      ctx.drawLine(legendX - legendW, legendY, legendX, legendY);
      ctx.drawLine(legendX - legendW, legendY, legendX - legendW, legendY - LEGEND_H);

      ctx.drawLine(LABEL_WIDTH - 1, 0, LABEL_WIDTH - 1, height);
      ctx.drawLine(0, 0, width, 0);
      ctx.drawLine(0, height - 1, width, height - 1);

      ctx.withClip(LABEL_WIDTH + 1, 0, width, height, () -> {
        ctx.withTranslation(LABEL_WIDTH, 0, () -> {
          for (long s = start; step > 0 && s < visible.end; s += step) {
            double xPos = Math.floor(state.timeToPx(s));
            if (xPos > width - LABEL_WIDTH) {
              break;
            } else if (xPos > 0) {
              ctx.setForegroundColor(colors().textMain);
              ctx.drawText(
                  Fonts.Style.Normal, timeToString(s - trace.start, step), xPos + MARGIN, LABEL_Y);
            }

            ctx.setForegroundColor(colors().timelineRuler);
            ctx.drawLine(xPos, 0, xPos, height);

            for (int i = 1; i < 5; i++) {
              double x = Math.floor(state.timeToPx(s + (i * step) / 5));
              ctx.drawLine(x, height - TICK_SIZE, x, height);
            }
          }
        });
      });
    });
  }

  public static void drawGridLines(
      RenderContext ctx, State state, double x, double y, double width, double height) {
    TimeSpan visible = state.getVisibleTime();
    TimeSpan trace = state.getTraceTime();
    long step = getGridStepSize(visible.getDuration(), width / DESIRED_PX_PER_STEP);
    long offset = ((visible.start - trace.start + step) / step) * step;
    long start = offset + trace.start;

    ctx.setForegroundColor(colors().gridline);
    ctx.path(path -> {
      for (long sec = start; sec < visible.end; sec += step) {
        double xPos = Math.floor(state.timeToPx(sec));
        if (xPos < 0) {
          continue;
        } else if (xPos >= width) {
          break;
        }
        path.moveTo(x + xPos, y);
        path.lineTo(x + xPos, y + height);
      }
      ctx.drawPath(path);
    });

  }

  /**
   * Returns the step size of a grid line in ns.
   * The returned step size has two properties:
   * (1) It is 1, 2, or 5, multiplied by some integer power of 10.
   * (2) The number steps in |range| produced by |stepSize| is as close as
   *     possible to |desiredSteps|.
   */
  private static long getGridStepSize(long range, double desiredSteps) {
    double initial = Math.pow(10, Math.floor(Math.log10(range / Math.max(1, desiredSteps))));
    double result = initial;
    double min = range / initial - desiredSteps;
    for (double step : new double[] {2 * initial, 5 * initial, 10 * initial}) {
      double delta = Math.abs(range / step - desiredSteps);
      if (delta < min) {
        min = delta;
        result = step;
      }
    }
    return Math.max(1, Math.round(result));
  }

  private static String timeToString(long ns, long resolution) {
    final String[] units = { "s", "ms", "us", "ns" };
    int u = 3;
    while (u > 0 && resolution > 1000) {
      ns /= 1000;
      resolution /= 1000;
      u--;
    }
    return String.format("%,d%s", ns, units[u]);
  }
}
