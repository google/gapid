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
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.MemorySummaryTrack;

/**
 * Displays information about the system memory usage.
 */
public class MemorySummaryPanel extends TrackPanel {
  private static final double HEIGHT = 80;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static final double LEGEND_SIZE = 8;

  private final MemorySummaryTrack track;
  protected HoverCard hovered = null;
  protected double mouseXpos, mouseYpos;

  public MemorySummaryPanel(State state, MemorySummaryTrack track) {
    super(state);
    this.track = track;
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
      MemorySummaryTrack.Data data = track.getData(state, () -> {
        repainter.repaint(new Area(0, 0, width, height));
      });
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      ctx.setBackgroundColor(colors().memoryBufferedCached);
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
        }
        path.lineTo(lastX, h);
        path.close();
        ctx.fillPath(path);
      });

      ctx.setBackgroundColor(colors().memoryUsed);
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

      if (hovered != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(mouseXpos + HOVER_MARGIN, mouseYpos,
            hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE, hovered.allSize.h);

        double x = mouseXpos + HOVER_MARGIN + HOVER_PADDING, y = mouseYpos;
        double dy = hovered.allSize.h / 4;
        ctx.setBackgroundColor(colors().background);
        ctx.fillRect(x, y + 1 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);
        ctx.setBackgroundColor(colors().memoryBufferedCached);
        ctx.fillRect(x, y + 2 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);
        ctx.setBackgroundColor(colors().memoryUsed);
        ctx.fillRect(x, y + 3 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);

        x += LEGEND_SIZE + HOVER_PADDING;
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(HoverCard.TOTAL_LABEL,     x, y + 0 * dy, dy);
        ctx.drawText(HoverCard.FREE_LABEL,      x, y + 1 * dy, dy);
        ctx.drawText(HoverCard.BUFFCACHE_LABEL, x, y + 2 * dy, dy);
        ctx.drawText(HoverCard.USED_LABEL,      x, y + 3 * dy, dy);

        x += hovered.labelSize.w + HOVER_PADDING + hovered.valueSize.w;
        ctx.drawTextRightJustified(hovered.totalS,     x, y + 0 * dy, dy);
        ctx.drawTextRightJustified(hovered.freeS,      x, y + 1 * dy, dy);
        ctx.drawTextRightJustified(hovered.buffCacheS, x, y + 2 * dy, dy);
        ctx.drawTextRightJustified(hovered.usedS,      x, y + 3 * dy, dy);

        ctx.drawCircle(mouseXpos, h * hovered.free / hovered.total, CURSOR_SIZE / 2);
        ctx.drawCircle(mouseXpos, h * (hovered.free + hovered.buffCache) / hovered.total, CURSOR_SIZE / 2);
      }

      String label = bytesToString(track.getMaxTotal());
      Size labelSize = ctx.measure(label);
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(0, 0, labelSize.w + 8, labelSize.h + 8);
      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(label, 4, 4);
    });
  }

  @Override
  protected Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    MemorySummaryTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null || data.ts.length == 0) {
      return Hover.NONE;
    }

    long time = state.pxToTime(x);
    int idx = 0;
    for (; idx < data.ts.length - 1; idx++) {
      if (data.ts[idx + 1] > time) {
        break;
      }
    }

    hovered = new HoverCard(m, data.total[idx], data.unused[idx], data.buffCache[idx]);
    mouseXpos = state.timeToPx(data.ts[idx]);
    mouseYpos = (height - 2 * TRACK_MARGIN - hovered.allSize.h) / 2;
    return new Hover() {
      @Override
      public Area getRedraw() {
        return new Area(mouseXpos - CURSOR_SIZE, mouseYpos,
            CURSOR_SIZE + HOVER_MARGIN + hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE,
            hovered.allSize.h);
      }

      @Override
      public void stop() {
        hovered = null;
      }
    };
  }

  protected static String bytesToString(long val) {
    if (val < 1024 + 512) {
      return val + "b";
    }
    double v = val / 1024.0;
    if (v < 1024 + 512) {
      return String.format("%.1fKb", v);
    }
    v /= 1024;
    if (v < 1024 + 512) {
      return String.format("%.1fMb", v);
    }
    v /= 1024;
    if (v < 1024 + 512) {
      return String.format("%.1fGb", v);
    }
    v /= 1024;
    return String.format("%.1fTb", v);
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

    public HoverCard(TextMeasurer tm, long total, long free, long buffCache) {
      this.total = total;
      this.free = free;
      this.buffCache = buffCache;
      this.totalS = bytesToString(total);
      this.freeS = bytesToString(free);
      this.buffCacheS = bytesToString(buffCache);
      this.usedS = bytesToString(total - free - buffCache);

      this.labelSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
          tm.measure(TOTAL_LABEL),
          tm.measure(FREE_LABEL),
          tm.measure(BUFFCACHE_LABEL),
          tm.measure(USED_LABEL));

      this.valueSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
          tm.measure(this.totalS),
          tm.measure(this.freeS),
          tm.measure(this.buffCacheS),
          tm.measure(this.usedS));
      this.allSize =
          new Size(labelSize.w + HOVER_PADDING + valueSize.w, Math.max(labelSize.h, valueSize.h));
    }
  }
}
