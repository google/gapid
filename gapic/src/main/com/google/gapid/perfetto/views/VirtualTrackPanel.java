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
import static com.google.gapid.util.Colors.hsl;
import static java.util.logging.Level.INFO;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.VirtualTrack;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;
import com.google.gapid.perfetto.models.SliceTrack;
import com.google.gapid.perfetto.models.VirtualTrack;
import java.util.logging.Level;
import java.util.logging.Logger;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.widgets.Display;

/**
 * Displays the slices of a virtual track.
 */
public class VirtualTrackPanel extends TrackPanel implements Selectable {
  private static final double HEIGHT = 30;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;

  private static String hoveredSlice;
  protected double mouseXpos;
  protected double hoveredWidth;

  private final VirtualTrack track;

  public VirtualTrackPanel(State state, VirtualTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public String getTitle() {
    return track.getName();
  }

  @Override
  public String getSubTitle() {
    return "";
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("VirtualTrackPanel", () -> {
      VirtualTrack.Data data =
          track.getData(state, () -> { repainter.repaint(new Area(0, 0, width, height)); });
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      renderSlices(ctx, data, h);
    });
  }

  private void renderSlices(RenderContext ctx, VirtualTrack.Data data, double h) {
    TimeSpan visible = state.getVisibleTime();
    for (int i = 0; i < data.ts.length; i++) {
      long tStart = data.ts[i];
      long tEnd = data.ts[i] + data.dur[i];
      if (tStart >= visible.end && tEnd <= visible.start) {
        continue;
      }

      double startPx = state.timeToPx(tStart);
      float hue = (data.eventNames[i].hashCode() & 0x7fffffff) % 360;
      RGBA color = hsl(hue, 0.9f, .65f);
      if (data.dur[i] > 0) {
        double rectWidth = Math.max(1, state.timeToPx(tEnd) - startPx);
        ctx.setBackgroundColor(color);
        ctx.fillRect(startPx, 0, rectWidth, height);
      } else {
        ctx.setForegroundColor(color);
        double radius = height / 5.5;
        ctx.drawCircle(startPx, height / 2 - radius, radius);
        if (data.eventNames[i].length() > 0) {
          String firstChar = data.eventNames[i].substring​(0, 1);
          ctx.drawText(firstChar, startPx - (radius/2), radius);
        }
      }
    }
    if (hoveredSlice != null) {
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(mouseXpos + HOVER_MARGIN, 0, hoveredWidth + 2 * HOVER_PADDING, h);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(hoveredSlice, mouseXpos + HOVER_MARGIN + HOVER_PADDING, 2, (h / 2) - 4);
    }
  }

  @Override
  public Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    VirtualTrack.Data data = track.getData(state, () -> {/* nothing */});
    if (data == null) {
      return Hover.NONE;
    }

    mouseXpos = x;
    long t = state.pxToTime(x);
    for (int i = 0; i < data.ts.length; i++) {
      double ts = state.timeToPx(data.ts[i]);
      if (ts <= x && x <= ts + Math.max(state.timeToPx(data.dur[i]) - ts, 5)) {
        hoveredSlice = data.eventNames[i];
        if (hoveredSlice == null) {
          return Hover.NONE;
        }
        hoveredWidth = Math.max(m.measure(hoveredSlice).w, m.measure(hoveredSlice).w);
        long id = data.sliceId[i];

        return new Hover() {
          @Override
          public Area getRedraw() {
            return new Area(x + HOVER_MARGIN, 0, hoveredWidth + 2 * HOVER_PADDING, HEIGHT);
          }

          @Override
          public void stop() {
            hoveredSlice = null;
            mouseXpos = 0;
          }

          @Override
          public Cursor getCursor(Display display) {
            return display.getSystemCursor(SWT.CURSOR_HAND);
          }

          @Override
          public boolean click() {
            state.setSelection(VirtualTrack.getSliceAndArgs(state.getQueryEngine(), id));
            return false;
          }
        };
      }
    }
    return Hover.NONE;
  }

  @Override
  public void computeSelection(CombiningBuilder builder, Area area, TimeSpan ts) {}
}
