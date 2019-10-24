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

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.ArgSet;
import com.google.gapid.perfetto.models.GpuInfo;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;
import com.google.gapid.perfetto.models.SliceTrack;
import com.google.gapid.perfetto.views.State.Location;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

import java.util.List;

/**
 * Draws the GPU Queue slices.
 */
public class GpuQueuePanel extends TrackPanel implements Selectable {
  private static final double SLICE_HEIGHT = 25 - 2 * TRACK_MARGIN;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;

  private final GpuInfo.Queue queue;
  protected final SliceTrack track;

  protected double mouseXpos, mouseYpos;
  protected String hoveredTitle;
  protected String hoveredCategory;
  protected Size hoveredSize = Size.ZERO;

  public GpuQueuePanel(State state, GpuInfo.Queue queue, SliceTrack track) {
    super(state);
    this.queue = queue;
    this.track = track;
  }


  @Override
  public String getTitle() {
    return queue.getDisplay();
  }

  @Override
  public double getHeight() {
    return queue.maxDepth * SLICE_HEIGHT;
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
        String title = buildSliceTitle(data.titles[i], data.args[i]);

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
        ctx.drawText(
            Fonts.Style.Normal, title, rectStart + 2, y + 2, rectWidth - 4, SLICE_HEIGHT - 4);
      }

      renderMarks(ctx, h);

      if (hoveredTitle != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(
            mouseXpos + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Normal, hoveredTitle,
            mouseXpos + HOVER_MARGIN + HOVER_PADDING, mouseYpos + HOVER_PADDING / 2);
        if (!hoveredCategory.isEmpty()) {
          ctx.setForegroundColor(colors().textAlt);
          ctx.drawText(Fonts.Style.Normal, hoveredCategory,
              mouseXpos + HOVER_MARGIN + HOVER_PADDING,
              mouseYpos + hoveredSize.h / 2, hoveredSize.h / 2);
        }
      }
    });
  }

  private static String buildSliceTitle(String title, ArgSet args) {
    Object w = args.get("width"), h = args.get("height");
    return (w == null || h == null) ? title : title + " (" + w + "x" + h + ")";
  }

  @Override
  protected Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y) {
    SliceTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null) {
      return Hover.NONE;
    }

    int depth = (int)(y / SLICE_HEIGHT);
    if (depth < 0 || depth > queue.maxDepth) {
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
        hoveredTitle = buildSliceTitle(hoveredTitle, data.args[i]);

        hoveredSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
            m.measure(Fonts.Style.Normal, hoveredTitle),
            hoveredCategory.isEmpty() ? Size.ZERO : m.measure(Fonts.Style.Normal, hoveredCategory));
        mouseYpos = Math.max(0, Math.min(mouseYpos - (hoveredSize.h - SLICE_HEIGHT) / 2,
            (1 + queue.maxDepth) * SLICE_HEIGHT - hoveredSize.h));
        long id = data.ids[i];
        long start = data.starts[i];
        long end = data.ends[i];

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
            return (id < 0) ? null : display.getSystemCursor(SWT.CURSOR_HAND);
          }

          @Override
          public boolean click() {
            if (id >= 0) {
              state.setSelection(track.getSlice(state.getQueryEngine(), id));
              state.addMarkLocation(GpuQueuePanel.this, new Location(start, end, depth));
            }
            return true;
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
      if (endDepth >= queue.maxDepth) {
        endDepth = Integer.MAX_VALUE;
      }

      builder.add(Kind.Gpu, transform(
          track.getSlices(state.getQueryEngine(), ts, startDepth, endDepth),
          SliceTrack.Slices::new));
    }
  }

  @Override
  public void updateMarkLocations(List<ListenableFuture<Void>> updateTasks, Area area, TimeSpan ts) {
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
      if (endDepth >= queue.maxDepth) {
        endDepth = Integer.MAX_VALUE;
      }

      updateTasks.add(transform(
          track.getSlices(state.getQueryEngine(), ts, startDepth, endDepth), slices -> {
            slices.stream().forEach(s -> state.addMarkLocation(GpuQueuePanel.this,
                new Location(s.time, s.time + s.dur, s.depth)));
            return null;
          }));
    }
  }

  @Override
  public void renderMarks(RenderContext ctx, double h) {
    if (state.getMarkLocations().containsKey(GpuQueuePanel.this)) {
      ctx.setForegroundColor(SWT.COLOR_BLACK);
      for (Location location : state.getMarkLocations().get(GpuQueuePanel.this)) {
        double rectStart = state.timeToPx(location.xTimeSpan.start);
        double rectWidth = Math.max(1, state.timeToPx(location.xTimeSpan.end) - rectStart);
        double depth = location.yOffset;
        ctx.drawRect(rectStart, depth * SLICE_HEIGHT, rectWidth, SLICE_HEIGHT, 3);
      }
    }
  }
}
