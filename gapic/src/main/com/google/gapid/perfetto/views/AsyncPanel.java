/*
 * Copyright (C) 2021 Google Inc.
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
import static com.google.gapid.perfetto.views.StyleConstants.SELECTION_THRESHOLD;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.common.collect.Lists;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.AsyncInfo;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.SliceTrack;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.widgets.Display;

import java.util.List;

public class AsyncPanel extends TrackPanel<AsyncPanel> implements Selectable {
  private static final double SLICE_HEIGHT = 25 - 2 * TRACK_MARGIN;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final int BOUNDING_BOX_LINE_WIDTH = 3;

  private final AsyncInfo async;
  protected final SliceTrack track;

  protected double mouseXpos, mouseYpos;
  protected String hoveredTitle;
  protected String hoveredCategory;
  protected Size hoveredSize = Size.ZERO;

  public AsyncPanel(State state, AsyncInfo async, SliceTrack track) {
    super(state);
    this.async = async;
    this.track = track;
  }

  @Override
  public AsyncPanel copy() {
    return new AsyncPanel(state, async, track);
  }

  @Override
  public String getTitle() {
    return async.name;
  }

  @Override
  public String getTooltip() {
    return "\\b" + async.name;
  }

  @Override
  public double getHeight() {
    return async.maxDepth * SLICE_HEIGHT;
  }

  @Override
  protected void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("Async", () -> {
      SliceTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      TimeSpan visible = state.getVisibleTime();
      Selection<?> selected = state.getSelection(Selection.Kind.Async);
      List<Highlight> visibleSelected = Lists.newArrayList();
      String[] concatedIds = data.getExtraStrings("concatedIds");

      for (int i = 0; i < data.starts.length; i++) {
        long tStart = data.starts[i];
        long tEnd = data.ends[i];
        int depth = data.depths[i];
        long id = data.ids[i];
        String title = data.titles[i];

        if (tEnd <= visible.start || tStart >= visible.end) {
          continue;
        }
        double rectStart = state.timeToPx(tStart);
        double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);
        double y = depth * SLICE_HEIGHT;

        StyleConstants.Gradient color = getSliceColor(data.titles[i]);
        color.applyBase(ctx);
        ctx.fillRect(rectStart, y, rectWidth, SLICE_HEIGHT);

        // Highlight slice if it's selected.
        if (selected.contains(id)) {
          visibleSelected.add(new Highlight(color.border, rectStart, y, rectWidth));
        } else if (i < concatedIds.length) {
          for (String cId : concatedIds[i].split(",")) {
            if (selected.contains(Long.parseLong(cId))) {
              visibleSelected.add(new Highlight(color.border, rectStart, y, rectWidth));
              break;
            }
          }
        }

        // Don't render text when we have less than 7px to play with.
        if (rectWidth < 7) {
          continue;
        }

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(
            Fonts.Style.Normal, title, rectStart + 2, y + 2, rectWidth - 4, SLICE_HEIGHT - 4);
      }

      // Draw bounding rectangles after all the slices are rendered, so that the border is on the top.
      for (Highlight highlight : visibleSelected) {
        ctx.setForegroundColor(highlight.color);
        ctx.drawRect(highlight.x, highlight.y, highlight.w, SLICE_HEIGHT, BOUNDING_BOX_LINE_WIDTH);
      }

      if (hoveredTitle != null) {
        double cardW = hoveredSize.w + 2 * HOVER_PADDING;
        double cardX = mouseXpos + HOVER_MARGIN;
        if (cardX >= w - cardW) {
          cardX = mouseXpos - HOVER_MARGIN - cardW;
        }
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(cardX, mouseYpos, cardW, hoveredSize.h);

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Normal, hoveredTitle, cardX + HOVER_PADDING,
            mouseYpos + HOVER_PADDING / 2);
        if (!hoveredCategory.isEmpty()) {
          ctx.setForegroundColor(colors().textAlt);
          ctx.drawText(Fonts.Style.Normal, hoveredCategory, cardX + HOVER_PADDING,
              mouseYpos + hoveredSize.h / 2, hoveredSize.h / 2);
        }
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    SliceTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
    if (data == null) {
      return Hover.NONE;
    }

    int depth = (int)(y / SLICE_HEIGHT);
    if (depth < 0 || depth > async.maxDepth) {
      return Hover.NONE;
    }

    mouseXpos = x;
    mouseYpos = depth * SLICE_HEIGHT;
    long t = state.pxToTime(x);
    for (int i = 0; i < data.starts.length; i++) {
      long tStart = data.starts[i];
      long tEnd = data.ends[i];
      if (data.depths[i] == depth && tStart <= t && t <= tEnd) {
        hoveredTitle = data.titles[i];
        hoveredCategory = data.categories[i];
        if (hoveredTitle.isEmpty()) {
          if (hoveredCategory.isEmpty()) {
            return Hover.NONE;
          }
          hoveredTitle = hoveredCategory;
          hoveredCategory = "";
        }
        hoveredTitle += " (" + TimeSpan.timeToString(tEnd - tStart) + ")";

        hoveredSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
            m.measure(Fonts.Style.Normal, hoveredTitle),
            hoveredCategory.isEmpty() ? Size.ZERO : m.measure(Fonts.Style.Normal, hoveredCategory));
        mouseYpos = Math.max(0, Math.min(mouseYpos - (hoveredSize.h - SLICE_HEIGHT) / 2,
            (1 + async.maxDepth) * SLICE_HEIGHT - hoveredSize.h));
        long id = data.ids[i];
        String concatedId = i < data.getExtraStrings("concatedIds").length ?
            data.getExtraStrings("concatedIds")[i] : "";

        return new Hover() {
          @Override
          public Area getRedraw() {
            double redrawW = HOVER_MARGIN + hoveredSize.w + 2 * HOVER_PADDING;
            double redrawX = x;
            if (redrawX >= state.getWidth() - redrawW) {
              redrawX = x - redrawW;
            }
            return new Area(redrawX, mouseYpos, redrawW, hoveredSize.h);
          }

          @Override
          public void stop() {
            hoveredTitle = hoveredCategory = null;
          }

          @Override
          public Cursor getCursor(Display display) {
            return (id < 0 && concatedId.isEmpty()) ? null : display.getSystemCursor(SWT.CURSOR_HAND);
          }

          @Override
          public boolean click() {
            if (id > 0) {
              if ((mods & SWT.MOD1) == SWT.MOD1) {
                state.addSelection(Selection.Kind.Async, track.getSlice(id));
              } else {
                state.setSelection(Selection.Kind.Async, track.getSlice(id));
              }
              return true;
            } else if (!concatedId.isEmpty()) {
              if ((mods & SWT.MOD1) == SWT.MOD1) {
                state.addSelection(Selection.Kind.Async, track.getSlices(concatedId));
              } else {
                state.setSelection(Selection.Kind.Async, track.getSlices(concatedId));
              }
              return true;
            }
            return false;
          }
        };
      }
    }
    return Hover.NONE;
  }

  @Override
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts) {
    int startDepth = (int)(area.y / SLICE_HEIGHT);
    int endDepth = (int)((area.y + area.h) / SLICE_HEIGHT);
    if (startDepth == endDepth && area.h / SLICE_HEIGHT < SELECTION_THRESHOLD) {
      return;
    }
    if (((startDepth + 1) * SLICE_HEIGHT - area.y) / SLICE_HEIGHT < SELECTION_THRESHOLD) {
      startDepth++;
    }
    if ((area.y + area.h - endDepth * SLICE_HEIGHT) / SLICE_HEIGHT < SELECTION_THRESHOLD) {
      endDepth--;
    }
    if (startDepth > endDepth) {
      return;
    }

    if (endDepth >= 0) {
      if (endDepth >= async.maxDepth) {
        endDepth = Integer.MAX_VALUE;
      }
      builder.add(Selection.Kind.Async, track.getSlices(ts, startDepth, endDepth));
    }
  }

  private static class Highlight {
    public final RGBA color;
    public final double x, y, w;

    public Highlight(RGBA color, double x, double y, double w) {
      this.color = color;
      this.x = x;
      this.y = y;
      this.w = w;
    }
  }
}
