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
import static com.google.gapid.perfetto.views.StyleConstants.batteryInGradient;
import static com.google.gapid.perfetto.views.StyleConstants.batteryOutGradient;
import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.common.collect.Lists;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.BatterySummaryTrack;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.Selection.Kind;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

import java.util.List;

public class BatterySummaryPanel extends TrackPanel<BatterySummaryPanel> implements Selectable {
  private static final double HEIGHT = 50;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static final double LEGEND_SIZE = 8;

  protected final BatterySummaryTrack track;
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
      BatterySummaryTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      double maxAbs = track.getMaxAbsCurrent();
      Selection<?> selected = state.getSelection(Selection.Kind.Battery);
      List<Integer> visibleSelected = Lists.newArrayList();

      // Draw outgoing battery current above the x axis.
      batteryOutGradient().applyBase(ctx);
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
          if (selected.contains(data.id[i])) {
            visibleSelected.add(i);
          }
        }
        path.lineTo(lastX, h / 2);
        path.close();
        ctx.fillPath(path);
      });

      // Draw ingoing battery current below the x axis.
      batteryInGradient().applyBase(ctx);
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

      // Draw highlight line after the whole graph is rendered, so that the highlight is on the top.
      for (int index : visibleSelected) {
        double startX = state.timeToPx(data.ts[index]);
        double endX = (index >= data.ts.length - 1) ? startX : state.timeToPx(data.ts[index + 1]);
        ctx.setBackgroundColor(data.current[index] > 0 ?
            batteryOutGradient().highlight : batteryInGradient().highlight);
        ctx.fillRect(startX, h / 2 - h / 2 * data.current[index] / maxAbs - 1, endX - startX, 3);
      }

      // Draw hovered card.
      if (hovered != null) {
        double cardW = hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE;
        double cardX = mouseXpos + CURSOR_SIZE / 2 + HOVER_MARGIN;
        if (cardX >= w - cardW) {
          cardX = mouseXpos - CURSOR_SIZE / 2 - HOVER_MARGIN - cardW;
        }
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(cardX, mouseYpos, cardW, hovered.allSize.h);

        double x = cardX + HOVER_PADDING, y = mouseYpos;
        double dy = hovered.allSize.h / 2;

        (hovered.current > 0 ? batteryOutGradient() : batteryInGradient()).applyBase(ctx);
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
  protected Hover onTrackMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    BatterySummaryTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
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

    long id = data.id[idx];
    hovered = new HoverCard(m, data.capacity[idx], data.charge[idx], data.current[idx]);
    mouseXpos = x;
    mouseYpos = (height - 2 * TRACK_MARGIN - hovered.allSize.h) / 2;
    return new Hover() {
      @Override
      public Area getRedraw() {
        double redrawW = CURSOR_SIZE + HOVER_MARGIN + hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE;
        double redrawX = mouseXpos - CURSOR_SIZE / 2;
        if (redrawX >= state.getWidth() - redrawW) {
          redrawX = mouseXpos + CURSOR_SIZE / 2.0 - redrawW;

          // If the hover card is drawn on the left side of the hover point, when moving the mouse
          // from left to right, the right edge of the cursor doesn't seem to get redrawn all the
          // time, this looks like a precision issue. This also happens when cursor is now on the
          // right side of the hover card, and the mouse moving from right to left there seems to
          // be a precision issue on the right edge of the cursor, hence extend the redraw with by
          // plusing the radius of the cursor.
          redrawW += CURSOR_SIZE / 2;
        }
        return new Area(redrawX, -TRACK_MARGIN, redrawW, HEIGHT + 2 * TRACK_MARGIN);
      }

      @Override
      public void stop() {
        hovered = null;
      }

      @Override
      public Cursor getCursor(Display display) {
        return display.getSystemCursor(SWT.CURSOR_HAND);
      }

      @Override
      public boolean click() {
        if ((mods & SWT.MOD1) == SWT.MOD1) {
          state.addSelection(Kind.Battery, track.getValue(id));
        } else {
          state.setSelection(Kind.Battery, track.getValue(id));
        }
        return true;
      }
    };
  }

  @Override
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts) {
    builder.add(Selection.Kind.Battery, track.getValues(ts));
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
