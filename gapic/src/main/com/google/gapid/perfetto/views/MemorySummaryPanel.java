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

import static com.google.gapid.perfetto.Unit.bytesToString;
import static com.google.gapid.perfetto.views.Loading.drawLoading;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.StyleConstants.memoryBuffersGradient;
import static com.google.gapid.perfetto.views.StyleConstants.memoryUsedGradient;

import com.google.common.collect.Lists;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.MemorySummaryTrack;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;
import com.google.gapid.perfetto.models.Selection.Kind;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

import java.util.List;

/**
 * Displays information about the system memory usage.
 */
public class MemorySummaryPanel extends TrackPanel<MemorySummaryPanel> implements Selectable {
  private static final double HEIGHT = 80;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static final double LEGEND_SIZE = 8;

  protected final MemorySummaryTrack track;

  protected HoverCard hovered = null;
  protected double mouseXpos, mouseYpos;

  public MemorySummaryPanel(State state, MemorySummaryTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public MemorySummaryPanel copy() {
    return new MemorySummaryPanel(state, track);
  }

  @Override
  public String getTitle() {
    return "Memory Usage";
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  protected void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("MemSummary", () -> {
      MemorySummaryTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      Selection<?> selected = state.getSelection(Selection.Kind.Memory);
      List<Integer> visibleSelected = Lists.newArrayList();

      memoryBuffersGradient().applyBase(ctx);
      ctx.path(path -> {
        path.moveTo(0, h);
        double lastX = 0, lastY = h;
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = h * data.unused[i] / data.total[i];
          path.lineTo(nextX, lastY);
          path.lineTo(nextX, nextY);
          lastX = nextX;
          lastY = nextY;
          if (selected.contains(data.id[i])) {
            visibleSelected.add(i);
          }
        }
        path.lineTo(lastX, h);
        path.close();
        ctx.fillPath(path);
      });

      memoryUsedGradient().applyBase(ctx);
      ctx.path(path -> {
        path.moveTo(0, h);
        double lastX = 0, lastY = h;
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = h * (data.unused[i] + data.buffCache[i]) / data.total[i];
          path.lineTo(nextX, lastY);
          path.lineTo(nextX, nextY);
          lastX = nextX;
          lastY = nextY;
        }
        path.lineTo(lastX, h);
        path.close();
        ctx.fillPath(path);
      });

      String label = bytesToString(track.getMaxTotal());
      Size labelSize = ctx.measure(Fonts.Style.Normal, label);
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(0, 0, labelSize.w + 8, labelSize.h + 8);
      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, label, 4, 4);

      // Draw highlight line after the whole graph is rendered, so that the highlight is on the top.
      for (int index : visibleSelected) {
        double startX = state.timeToPx(data.ts[index]);
        double endX = (index >= data.ts.length - 1) ? startX : state.timeToPx(data.ts[index + 1]);
        ctx.setBackgroundColor(memoryBuffersGradient().highlight);
        ctx.fillRect(startX, h * data.unused[index] / data.total[index] - 1, endX - startX, 3);
        ctx.setBackgroundColor(memoryUsedGradient().highlight);
        ctx.fillRect(startX, h * (data.unused[index] + data.buffCache[index]) / data.total[index] - 1, endX - startX, 3);
      }

      if (hovered != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(mouseXpos + HOVER_MARGIN, mouseYpos,
            hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE, hovered.allSize.h);

        double x = mouseXpos + HOVER_MARGIN + HOVER_PADDING, y = mouseYpos;
        double dy = hovered.allSize.h / 4;
        ctx.setBackgroundColor(colors().background);
        ctx.fillRect(x, y + 1 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);
        memoryBuffersGradient().applyBase(ctx);
        ctx.fillRect(x, y + 2 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);
        memoryUsedGradient().applyBase(ctx);
        ctx.fillRect(x, y + 3 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);

        x += LEGEND_SIZE + HOVER_PADDING;
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Bold, HoverCard.TOTAL_LABEL,     x, y + 0 * dy, dy);
        ctx.drawText(Fonts.Style.Bold, HoverCard.FREE_LABEL,      x, y + 1 * dy, dy);
        ctx.drawText(Fonts.Style.Bold, HoverCard.BUFFCACHE_LABEL, x, y + 2 * dy, dy);
        ctx.drawText(Fonts.Style.Bold, HoverCard.USED_LABEL,      x, y + 3 * dy, dy);

        x += hovered.labelSize.w + HOVER_PADDING + hovered.valueSize.w;
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.totalS,     x, y + 0 * dy, dy);
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.freeS,      x, y + 1 * dy, dy);
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.buffCacheS, x, y + 2 * dy, dy);
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.usedS,      x, y + 3 * dy, dy);

        ctx.drawCircle(mouseXpos, h * hovered.free / hovered.total, CURSOR_SIZE / 2);
        ctx.drawCircle(mouseXpos, h * (hovered.free + hovered.buffCache) / hovered.total, CURSOR_SIZE / 2);
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y, int mods) {
    MemorySummaryTrack.Data data = track.getData(state.toRequest(), onUiThread());
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
    hovered = new HoverCard(m, data.total[idx], data.unused[idx], data.buffCache[idx]);
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

      @Override
      public Cursor getCursor(Display display) {
        return display.getSystemCursor(SWT.CURSOR_HAND);
      }

      @Override
      public boolean click() {
        if ((mods & SWT.MOD1) == SWT.MOD1) {
          state.addSelection(Kind.Memory, track.getValue(id));
        } else {
          state.setSelection(Kind.Memory, track.getValue(id));
        }
        return true;
      }
    };
  }

  @Override
  public void computeSelection(CombiningBuilder builder, Area area, TimeSpan ts) {
    builder.add(Selection.Kind.Memory, track.getValues(ts));
  }

  private static class HoverCard {
    public static final String TOTAL_LABEL = "Total:";
    public static final String FREE_LABEL = "Unused:";
    public static final String BUFFCACHE_LABEL = "Buffers/Cache:";
    public static final String USED_LABEL = "Used:";

    public final long total;
    public final long free;
    public final long buffCache;

    public final String totalS;
    public final String freeS;
    public final String buffCacheS;
    public final String usedS;

    public final Size valueSize;
    public final Size labelSize;
    public final Size allSize;

    public HoverCard(Fonts.TextMeasurer tm, long total, long free, long buffCache) {
      this.total = total;
      this.free = free;
      this.buffCache = buffCache;
      this.totalS = bytesToString(total);
      this.freeS = bytesToString(free);
      this.buffCacheS = bytesToString(buffCache);
      this.usedS = bytesToString(total - free - buffCache);

      this.labelSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
          tm.measure(Fonts.Style.Bold, TOTAL_LABEL),
          tm.measure(Fonts.Style.Bold, FREE_LABEL),
          tm.measure(Fonts.Style.Bold, BUFFCACHE_LABEL),
          tm.measure(Fonts.Style.Bold, USED_LABEL));

      this.valueSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
          tm.measure(Fonts.Style.Normal, this.totalS),
          tm.measure(Fonts.Style.Normal, this.freeS),
          tm.measure(Fonts.Style.Normal, this.buffCacheS),
          tm.measure(Fonts.Style.Normal, this.usedS));
      this.allSize =
          new Size(labelSize.w + HOVER_PADDING + valueSize.w, Math.max(labelSize.h, valueSize.h));
    }
  }
}
