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
import static com.google.gapid.perfetto.views.StyleConstants.SELECTION_THRESHOLD;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.util.Colors.hsl;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.GpuInfo;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;
import com.google.gapid.perfetto.models.SliceTrack;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

/**
 * Draws the GPU Queue slices.
 */
public class GpuQueuePanel extends TrackPanel implements Selectable {
  private static final double SLICE_HEIGHT = 25 - 2 * TRACK_MARGIN;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;

  private final GpuInfo.Gpu gpu;
  protected final SliceTrack track;

  protected double mouseXpos, mouseYpos;
  protected String hoveredTitle;
  protected String hoveredCategory;
  protected Size hoveredSize = Size.ZERO;

  public GpuQueuePanel(State state, GpuInfo.Gpu gpu, SliceTrack track) {
    super(state);
    this.gpu = gpu;
    this.track = track;
  }


  @Override
  public String getTitle() {
    return gpu.getDisplay();
  }

  @Override
  public double getHeight() {
    return gpu.maxDepth * SLICE_HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("GpuQueue", () -> {
      SliceTrack.Data data = track.getData(state, () -> {
        repainter.repaint(new Area(0, 0, width, height));
      });
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      TimeSpan visible = state.getVisibleTime();
      for (int i = 0; i < data.starts.length; i++) {
        long tStart = data.starts[i];
        long tEnd = data.ends[i];
        int depth = data.depths[i];
        //String cat = data.categories[i];
        String title = data.titles[i];
        if (tEnd <= visible.start || tStart >= visible.end) {
          continue;
        }
        double rectStart = state.timeToPx(tStart);
        double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);
        double y = depth * SLICE_HEIGHT;

        float hue = (title.hashCode() & 0x7fffffff) % 360;
        float saturation = Math.min(20 + depth * 10, 70) / 100f;
        ctx.setBackgroundColor(hsl(hue, saturation, .65f));
        ctx.fillRect(rectStart, y, rectWidth, SLICE_HEIGHT);

        // Don't render text when we have less than 7px to play with.
        if (rectWidth < 7) {
          continue;
        }

        ctx.setForegroundColor(colors().textInvertedMain);
        ctx.drawText(title, rectStart + 2, y + 2, rectWidth - 4, SLICE_HEIGHT - 4);
      }

      if (hoveredTitle != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(
            mouseXpos + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(
            hoveredTitle, mouseXpos + HOVER_MARGIN + HOVER_PADDING, mouseYpos + HOVER_PADDING / 2);
        if (!hoveredCategory.isEmpty()) {
          ctx.setForegroundColor(colors().textAlt);
          ctx.drawText(hoveredCategory, mouseXpos + HOVER_MARGIN + HOVER_PADDING,
              mouseYpos + hoveredSize.h / 2, hoveredSize.h / 2);
        }
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    SliceTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null) {
      return Hover.NONE;
    }

    int depth = (int)(y / SLICE_HEIGHT);
    if (depth < 0 || depth > gpu.maxDepth) {
      return Hover.NONE;
    }

    mouseXpos = x;
    mouseYpos = depth * SLICE_HEIGHT;
    long t = state.pxToTime(x);
    for (int i = 0; i < data.starts.length; i++) {
      if (data.depths[i] == depth && data.starts[i] <= t && t <= data.ends[i]) {
        hoveredTitle = data.titles[i];
        hoveredCategory = data.categories[i];
        if (hoveredTitle.isEmpty()) {
          if (hoveredCategory.isEmpty()) {
            return Hover.NONE;
          }
          hoveredTitle = hoveredCategory;
          hoveredCategory = "";
        }

        hoveredSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
            m.measure(hoveredTitle),
            hoveredCategory.isEmpty() ? Size.ZERO : m.measure(hoveredCategory));
        mouseYpos = Math.max(0, Math.min(mouseYpos - (hoveredSize.h - SLICE_HEIGHT) / 2,
            (1 + gpu.maxDepth) * SLICE_HEIGHT - hoveredSize.h));
        long id = data.ids[i];
        long ts = data.starts[i];

        return new Hover() {
          @Override
          public Area getRedraw() {
            return new Area(
                x + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);
          }

          @Override
          public void stop() {
            hoveredTitle = hoveredCategory = null;
          }

          @Override
          public Cursor getCursor(Display display) {
            return (id == 0) ? null : display.getSystemCursor(SWT.CURSOR_HAND);
          }

          @Override
          public boolean click() {
            if (id != 0) {
              state.setSelection(SliceTrack.getSlice(
                  state.getQueryEngine(), SliceTrack.SliceType.Gpu, id, ts));
            }
            return false;
          }
        };
      }
    }
    return Hover.NONE;
  }

  @Override
  public void computeSelection(CombiningBuilder builder, Area area, TimeSpan ts) {
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
      if (endDepth >= gpu.maxDepth) {
        endDepth = Integer.MAX_VALUE;
      }
      builder.add(Kind.Gpu, transform(
          SliceTrack.getGpuSlices(state.getQueryEngine(), gpu.id, ts, startDepth, endDepth),
          SliceTrack.Slices::new));
    }
  }
}
