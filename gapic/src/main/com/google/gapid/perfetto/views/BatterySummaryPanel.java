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
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Fonts.TextMeasurer;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.BatterySummaryTrack;
import com.google.gapid.perfetto.views.StyleConstants.Palette.BaseColor;

import org.eclipse.swt.graphics.RGBA;

import java.util.Arrays;

public class BatterySummaryPanel extends TrackPanel<BatterySummaryPanel> {
  private static final double HEIGHT = 50;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static final double LEGEND_SIZE = 8;

  private final BatterySummaryTrack track;
  protected HoverCard hovered = null;
  protected double mouseXpos, mouseYpos;

  public BatterySummaryPanel(State state, BatterySummaryTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public BatterySummaryPanel copy() {
    return new BatterySummaryPanel(state, track);
  }

  @Override
  public String getTitle() {
    return "Battery Usage";
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  protected void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("BatSummary", () -> {
      BatterySummaryTrack.Data data = track.getData(state, () -> {
        repainter.repaint(new Area(0, 0, width, height));
      });
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      long maxAbs = maxAbsCurrent(data.current);

      // Draw outgoing battery current above the x axis.
      ctx.setBackgroundColor(BaseColor.ORANGE.rgb);
      ctx.path(path -> {
        path.moveTo(0, h / 2);
        double lastX = 0, lastY = h / 2;
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = data.current[i] > 0 ? h / 2 - h / 2 * data.current[i] / maxAbs : h / 2;
          path.lineTo(nextX, lastY);
          path.lineTo(nextX, nextY);
          lastX = nextX;
          lastY = nextY;
        }
        path.lineTo(lastX, h / 2);
        path.close();
        ctx.fillPath(path);
      });

      // Draw ingoing battery current below the x axis.
      ctx.setBackgroundColor(BaseColor.GREEN.rgb);
      ctx.path(path -> {
        path.moveTo(0, h / 2);
        double lastX = 0, lastY = h / 2;
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = data.current[i] <= 0 ? h / 2 - h / 2 * data.current[i] / maxAbs : h / 2;
          path.lineTo(nextX, lastY);
          path.lineTo(nextX, nextY);
          lastX = nextX;
          lastY = nextY;
        }
        path.lineTo(lastX, h / 2);
        path.close();
        ctx.fillPath(path);
      });

      // Draw hovered card.
      if (hovered != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(mouseXpos + HOVER_MARGIN, mouseYpos,
            hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE, hovered.allSize.h);

        double x = mouseXpos + HOVER_MARGIN + HOVER_PADDING, y = mouseYpos;
        double dy = hovered.allSize.h / 2;

        RGBA color = hovered.current > 0 ? BaseColor.ORANGE.rgb : BaseColor.GREEN.rgb;
        ctx.setBackgroundColor(color);
        ctx.fillRect(x, y + dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);

        x += LEGEND_SIZE + HOVER_PADDING;
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Bold, HoverCard.REMAINING_POWER_LABEL, x, y, dy);
        String s = hovered.current > 0 ? HoverCard.CURRENT_OUT_LABEL : HoverCard.CURRENT_IN_LABEL;
        ctx.drawText(Fonts.Style.Bold, s, x, y + dy, dy);

        x += hovered.labelSize.w + HOVER_PADDING + hovered.valueSize.w;
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.remainingS, x, y, dy);
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.currentAbsS, x, y + dy, dy);

        ctx.drawCircle(mouseXpos, h / 2 - h / 2 * hovered.current / maxAbs,
            CURSOR_SIZE / 2);
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    BatterySummaryTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null || data.ts.length == 0) {
      return Hover.NONE;
    }

    long time = state.pxToTime(x);
    if (time < data.ts[0] || time > data.ts[data.ts.length - 1]) {
      return Hover.NONE;
    }
    int idx = 0;
    for (; idx < data.ts.length - 1; idx++) {
      if (data.ts[idx + 1] > time) {
        break;
      }
    }

    hovered = new HoverCard(m, data.capacity[idx], data.charge[idx], data.current[idx]);
    mouseXpos = x;
    mouseYpos = (height - 2 * TRACK_MARGIN - hovered.allSize.h) / 2;
    return new Hover() {
      @Override
      public Area getRedraw() {
        return new Area(mouseXpos - CURSOR_SIZE, -TRACK_MARGIN,
            CURSOR_SIZE + HOVER_MARGIN + hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE,
            HEIGHT + 2 * TRACK_MARGIN);
      }

      @Override
      public void stop() {
        hovered = null;
      }
    };
  }

  private static long maxAbsCurrent(long[] currents) {
    return Arrays.stream(currents).map(Math::abs).max().orElse(0);
  }

  private static class HoverCard {
    public static final String REMAINING_POWER_LABEL = "Capacity:";
    public static final String CURRENT_OUT_LABEL = "Current Out:";
    public static final String CURRENT_IN_LABEL = "Current In:";

    public final long current;

    public final String remainingS;
    public final String currentAbsS;

    public final Size valueSize;
    public final Size labelSize;
    public final Size allSize;

    public HoverCard(Fonts.TextMeasurer tm, long capacity, long charge, long current) {
      this.current = current;
      this.remainingS = formatCapacity(capacity) + " / " + formatCharge(charge);
      this.currentAbsS = formatCurrent(current);

      this.labelSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
          tm.measure(Fonts.Style.Bold, REMAINING_POWER_LABEL),
          tm.measure(Fonts.Style.Bold, CURRENT_OUT_LABEL));

      this.valueSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
          tm.measure(Fonts.Style.Normal, this.remainingS),
          tm.measure(Fonts.Style.Normal, this.currentAbsS));

      this.allSize =
          new Size(labelSize.w + HOVER_PADDING + valueSize.w, Math.max(labelSize.h, valueSize.h));
    }

    private static String formatCapacity(long capacityInPct) {
      return capacityInPct + "%";
    }

    private static String formatCharge(long chargeInUah) {
      if (chargeInUah / 1000 > 0) {
        return chargeInUah / 1000 + "mAH";
      } else {
        return chargeInUah + "uAH";
      }
    }

    private static String formatCurrent(long currentInUa) {
      // Show absolute value. Use color and in/out label to show electric current's direction.
      long absCurrentInUa = Math.abs(currentInUa);
      if (absCurrentInUa / 1000 > 0) {
        return absCurrentInUa / 1000 + "mA";
      } else {
        return absCurrentInUa + "uA";
      }
    }
  }
}
