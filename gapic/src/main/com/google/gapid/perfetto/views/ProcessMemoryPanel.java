/*
 * Copyright (C) 2020 Google Inc.
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
import static com.google.gapid.perfetto.views.StyleConstants.memoryRssAnonGradient;
import static com.google.gapid.perfetto.views.StyleConstants.memoryRssFileGradient;
import static com.google.gapid.perfetto.views.StyleConstants.memoryRssSharedGradient;
import static com.google.gapid.perfetto.views.StyleConstants.memorySwapGradient;

import com.google.common.collect.Lists;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.ProcessMemoryTrack;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.Selection.Kind;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

import java.util.List;

public class ProcessMemoryPanel extends TrackPanel<ProcessMemoryPanel> implements Selectable {
  private static final double HEIGHT = 80;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static final double LEGEND_SIZE = 8;

  protected final ProcessMemoryTrack track;
  protected HoverCard hovered = null;
  protected double mouseXpos, mouseYpos;

  public ProcessMemoryPanel(State state, ProcessMemoryTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public ProcessMemoryPanel copy() {
    return new ProcessMemoryPanel(state, track);
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
    ctx.trace("ProcessMemory", () -> {
      ProcessMemoryTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null || data.ts.length == 0) {
        return;
      }

      Selection<?> selected = state.getSelection(Selection.Kind.ProcessMemory);
      List<Integer> visibleSelected = Lists.newArrayList();
      long maxUsage = track.getMaxUsage(), maxSwap = Math.max(maxUsage, track.getMaxSwap());

      memoryRssSharedGradient().applyBase(ctx);
      ctx.path(path -> {
        path.moveTo(0, h);
        double lastX = 0, lastY = h;
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = h - (h * (data.file[i] + data.anon[i] + data.shared[i]) / maxUsage);
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

      memoryRssAnonGradient().applyBase(ctx);
      ctx.path(path -> {
        path.moveTo(0, h);
        double lastX = 0, lastY = h;
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = h - (h * (data.file[i] + data.anon[i]) / maxUsage);
          path.lineTo(nextX, lastY);
          path.lineTo(nextX, nextY);
          lastX = nextX;
          lastY = nextY;
        }
        path.lineTo(lastX, h);
        path.close();
        ctx.fillPath(path);
      });

      memoryRssFileGradient().applyBase(ctx);
      ctx.path(path -> {
        path.moveTo(0, h);
        double lastX = 0, lastY = h;
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = h - (h * (data.file[i]) / maxUsage);
          path.lineTo(nextX, lastY);
          path.lineTo(nextX, nextY);
          lastX = nextX;
          lastY = nextY;
        }
        path.lineTo(lastX, h);
        path.close();
        ctx.fillPath(path);
      });

      memorySwapGradient().applyBorder(ctx);
      ctx.path(path -> {
        double lastX = state.timeToPx(data.ts[0]), lastY = h - (h * data.swap[0] / maxSwap);
        path.moveTo(lastX, lastY);
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = h - (h * data.swap[i] / maxSwap);
          path.lineTo(nextX, lastY);
          path.lineTo(nextX, nextY);
          lastX = nextX;
          lastY = nextY;
        }
        path.lineTo(lastX, h);
        ctx.drawPath(path);
      });

      // Draw highlight line after the whole graph is rendered, so that the highlight is on the top.
      for (int index : visibleSelected) {
        double startX = state.timeToPx(data.ts[index]);
        double endX = (index >= data.ts.length - 1) ? startX : state.timeToPx(data.ts[index + 1]);
        ctx.setBackgroundColor(memoryRssSharedGradient().highlight);
        ctx.fillRect(startX, h - h * (data.file[index] + data.anon[index] + data.shared[index]) / maxUsage - 1, endX - startX, 3);
        ctx.setBackgroundColor(memoryRssAnonGradient().highlight);
        ctx.fillRect(startX, h - h * (data.file[index] + data.anon[index]) / maxUsage - 1, endX - startX, 3);
        ctx.setBackgroundColor(memoryRssFileGradient().highlight);
        ctx.fillRect(startX, h - h * data.file[index] / maxUsage - 1, endX - startX, 3);
        ctx.setBackgroundColor(memorySwapGradient().highlight);
        ctx.fillRect(startX, h - h * data.swap[index] / maxSwap - 1, endX - startX, 3);
      }


      if (hovered != null) {
        double cardW = hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE;
        double cardX = mouseXpos + CURSOR_SIZE / 2 + HOVER_MARGIN;
        if (cardX >= w - cardW) {
          cardX = mouseXpos - CURSOR_SIZE / 2 - HOVER_MARGIN - cardW;
        }
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(cardX, mouseYpos, cardW, hovered.allSize.h);

        double x = cardX + HOVER_PADDING, y = mouseYpos;
        double dy = hovered.allSize.h / 4;
        memoryRssSharedGradient().applyBase(ctx);
        ctx.fillRect(x, y + 0 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);
        memoryRssAnonGradient().applyBase(ctx);
        ctx.fillRect(x, y + 1 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);
        memoryRssFileGradient().applyBase(ctx);
        ctx.fillRect(x, y + 2 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);
        memorySwapGradient().applyBase(ctx);
        ctx.fillRect(x, y + 3 * dy + (dy - LEGEND_SIZE) / 2, LEGEND_SIZE, LEGEND_SIZE);

        x += LEGEND_SIZE + HOVER_PADDING;
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Bold, HoverCard.SHARED_LABEL, x, y + 0 * dy, dy);
        ctx.drawText(Fonts.Style.Bold, HoverCard.ANON_LABEL,   x, y + 1 * dy, dy);
        ctx.drawText(Fonts.Style.Bold, HoverCard.FILE_LABEL,   x, y + 2 * dy, dy);
        ctx.drawText(Fonts.Style.Bold, HoverCard.SWAP_LABEL,   x, y + 3 * dy, dy);

        x += hovered.labelSize.w + HOVER_PADDING + hovered.valueSize.w;
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.sharedS, x, y + 0 * dy, dy);
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.anonS,   x, y + 1 * dy, dy);
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.fileS,   x, y + 2 * dy, dy);
        ctx.drawTextRightJustified(Fonts.Style.Normal, hovered.swapS,   x, y + 3 * dy, dy);

        ctx.drawCircle(mouseXpos, h - h * (hovered.file + hovered.anon + hovered.shared) / maxUsage,
            CURSOR_SIZE / 2);
        ctx.drawCircle(
            mouseXpos, h - h * (hovered.file + hovered.anon) / maxUsage, CURSOR_SIZE / 2);
        ctx.drawCircle(mouseXpos, h - h * hovered.file / maxUsage, CURSOR_SIZE / 2);
        ctx.drawCircle(mouseXpos, h - h * hovered.swap / maxSwap, CURSOR_SIZE / 2);
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    ProcessMemoryTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
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
    hovered = new HoverCard(m, data.shared[idx], data.anon[idx], data.file[idx], data.swap[idx]);
    mouseXpos = x;
    mouseYpos = (height - 2 * TRACK_MARGIN - hovered.allSize.h) / 2;
    return new Hover() {
      @Override
      public Area getRedraw() {
        double redrawW = CURSOR_SIZE + HOVER_MARGIN + hovered.allSize.w + 3 * HOVER_PADDING + LEGEND_SIZE;
        double redrawX = mouseXpos - CURSOR_SIZE / 2;
        if (redrawX >= state.getWidth() - redrawW) {
          redrawX = mouseXpos + CURSOR_SIZE / 2 - redrawW;
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
          state.addSelection(Kind.ProcessMemory, track.getValue(id));
        } else {
          state.setSelection(Kind.ProcessMemory, track.getValue(id));
        }
        return true;
      }
    };
  }

  @Override
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts) {
    builder.add(Selection.Kind.ProcessMemory, track.getValues(ts));
  }

  private static class HoverCard {
    public static final String SHARED_LABEL = "Shared: ";
    public static final String ANON_LABEL = "Anon: ";
    public static final String FILE_LABEL = "File: ";
    public static final String SWAP_LABEL = "Swap: ";

    public final long shared;
    public final long anon;
    public final long file;
    public final long swap;

    public final String sharedS;
    public final String anonS;
    public final String fileS;
    public final String swapS;

    public final Size valueSize;
    public final Size labelSize;
    public final Size allSize;

    public HoverCard(Fonts.TextMeasurer tm, long shared, long anon, long file, long swap) {
      this.shared = shared;
      this.anon = anon;
      this.file = file;
      this.swap = swap;
      this.sharedS = bytesToString(shared);
      this.anonS = bytesToString(anon);
      this.fileS = bytesToString(file);
      this.swapS = bytesToString(swap);

      this.labelSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
          tm.measure(Fonts.Style.Bold, SHARED_LABEL),
          tm.measure(Fonts.Style.Bold, ANON_LABEL),
          tm.measure(Fonts.Style.Bold, FILE_LABEL),
          tm.measure(Fonts.Style.Bold, SWAP_LABEL));

      this.valueSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
          tm.measure(Fonts.Style.Normal, this.sharedS),
          tm.measure(Fonts.Style.Normal, this.anonS),
          tm.measure(Fonts.Style.Normal, this.fileS),
          tm.measure(Fonts.Style.Normal, this.swapS));

      this.allSize =
          new Size(labelSize.w + HOVER_PADDING + valueSize.w, Math.max(labelSize.h, valueSize.h));
    }
  }
}
