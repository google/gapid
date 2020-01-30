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

import com.google.common.collect.Lists;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Fonts.TextMeasurer;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.GpuInfo;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.VulkanEventTrack;
import java.util.List;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

public class VulkanEventPanel extends TrackPanel<VulkanEventPanel> {
  private static final double ARROW_HEIGHT = 10;
  private static final double ARROW_WIDTH = 10;
  private static final double ARROW_TIP = 2;
  private static final double SLICE_Y = ARROW_HEIGHT;
  private static final double SLICE_HEIGHT = 25;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final int BOUNDING_BOX_LINE_WIDTH = 3;

  private final GpuInfo.VkEvent vkEvent;
  protected final VulkanEventTrack track;

  protected double mouseXpos, mouseYpos;
  protected String hoveredName;
  protected Size hoveredSize = Size.ZERO;

  public VulkanEventPanel(State state, GpuInfo.VkEvent vkEvent, VulkanEventTrack track) {
    super(state);
    this.vkEvent = vkEvent;
    this.track = track;
  }

  @Override
  public VulkanEventPanel copy() {
    return new VulkanEventPanel(state, vkEvent, track);
  }

  @Override
  public String getTitle() {
    return vkEvent.getDisplay();
  }

  @Override
  public double getHeight() {
    return SLICE_Y + vkEvent.maxDepth * SLICE_HEIGHT;
  }

  @Override
  protected void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("VulkanEvents", () -> {
      VulkanEventTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      TimeSpan visible = state.getVisibleTime();
      Selection<Long> selected = state.getSelection(Selection.Kind.VulkanEvent);
      List<Integer> visibleSelected = Lists.newArrayList();

      for (int i = 0; i < data.starts.length; i++) {
        long tStart = data.starts[i];
        long tEnd = data.ends[i];
        int depth = data.depths[i];

        if (tEnd <= visible.start || tStart >= visible.end) {
          continue;
        }
        double rectStart = state.timeToPx(tStart);
        double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);
        double y = SLICE_Y + depth * SLICE_HEIGHT;

        ctx.setBackgroundColor(VulkanEventTrack.getColor(data.names[i]));
        ctx.fillRect(rectStart, y, rectWidth, SLICE_HEIGHT);

        if (selected.contains(data.ids[i])) {
          visibleSelected.add(i);
        }

        // Don't render text when we have less than 7px to play with.
        if (rectWidth < 7) {
          continue;
        }

        ctx.setForegroundColor(colors().textInvertedMain);
        ctx.drawText(
            Fonts.Style.Normal, data.names[i], rectStart + 2, y + 2, rectWidth - 4, SLICE_HEIGHT - 4);

      }

      // Draw bounding rectangles after all the slices are rendered, so that the border is on the top.
      for (int index : visibleSelected) {
        ctx.setForegroundColor(VulkanEventTrack.getBorderColor(data.names[index]));
        double rectStart = state.timeToPx(data.starts[index]);
        double rectWidth = Math.max(1, state.timeToPx(data.ends[index]) - rectStart);
        double depth = data.depths[index];
        ctx.drawRect(rectStart, SLICE_Y + depth * SLICE_HEIGHT, rectWidth, SLICE_HEIGHT, BOUNDING_BOX_LINE_WIDTH);

        double mid = rectStart + rectWidth / 2;
        ctx.drawLine(mid, ARROW_TIP, mid, SLICE_Y);
        ctx.drawLine(mid, ARROW_TIP, mid + ARROW_WIDTH, ARROW_TIP);
        ctx.drawLine(mid + ARROW_WIDTH, ARROW_TIP, mid + ARROW_WIDTH - ARROW_TIP, 0);
        ctx.drawLine(mid + ARROW_WIDTH, ARROW_TIP, mid + ARROW_WIDTH - ARROW_TIP, ARROW_TIP * 2);
      }

      if (hoveredName != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(
            mouseXpos + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Normal, hoveredName,
            mouseXpos + HOVER_MARGIN + HOVER_PADDING, mouseYpos + HOVER_PADDING / 2);
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(TextMeasurer m, double x, double y, int mods) {
    VulkanEventTrack.Data data = track.getData(state.toRequest(), onUiThread());
    if (data == null) {
      return Hover.NONE;
    }

    int depth = y < SLICE_Y ? -1 : (int)((y - SLICE_Y) / SLICE_HEIGHT);
    if (depth < 0 || depth > vkEvent.maxDepth) {
      return Hover.NONE;
    }

    mouseXpos = x;
    mouseYpos = SLICE_Y + depth * SLICE_HEIGHT;
    long t = state.pxToTime(x);
    for (int i = 0; i < data.starts.length; i++) {
      if (data.depths[i] == depth && data.starts[i] <= t && t <= data.ends[i]) {
        hoveredName = data.names[i];
        hoveredSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
            m.measure(Fonts.Style.Normal, hoveredName));
        mouseYpos = Math.max(0, Math.min(mouseYpos - (hoveredSize.h - SLICE_HEIGHT) / 2,
            (1 + vkEvent.maxDepth) * SLICE_HEIGHT - hoveredSize.h));
        long id = data.ids[i];

        return new Hover() {
          @Override
          public Area getRedraw() {
            return new Area(
                x + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);
          }

          @Override
          public void stop() {
            hoveredName =  null;
          }

          @Override
          public Cursor getCursor(Display display) {
            return (id < 0) ? null : display.getSystemCursor(SWT.CURSOR_HAND);
          }

          @Override
          public boolean click() {
            if (id < 0) {
              return false;
            }
            if ((mods & SWT.MOD1) == SWT.MOD1) {
              state.addSelection(Selection.Kind.VulkanEvent, track.getSlice(id));
            } else {
              state.setSelection(Selection.Kind.VulkanEvent, track.getSlice(id));
            }
            return true;
          }
        };
      }
    }
    return Hover.NONE;
  }
}
