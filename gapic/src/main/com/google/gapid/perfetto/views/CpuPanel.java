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
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.StyleConstants.hueForCpu;
import static com.google.gapid.util.Colors.hsl;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;
import com.google.gapid.perfetto.models.ThreadInfo;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

/**
 * Draws the CPU usage or slices of a single core.
 */
public class CpuPanel extends TrackPanel implements Selectable {
  private static final double HEIGHT = 30;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;

  private final CpuTrack track;
  private final float hue;
  protected double mouseXpos;
  protected ThreadInfo.Display hoveredThread;
  protected double hoveredWidth;

  public CpuPanel(State state, CpuTrack track) {
    super(state);
    this.track = track;
    this.hue = hueForCpu(track.getCpu());
  }

  @Override
  public String getTitle() {
    return "CPU " + (track.getCpu() + 1);
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("CpuTrack", () -> {
      CpuTrack.Data data = track.getData(state, () -> {
        repainter.repaint(new Area(0, 0, width, height));
      });
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      switch (data.kind) {
        case slice: renderSlices(ctx, data, h); break;
        case summary: renderSummary(ctx, data, w, h); break;
      }
    });
  }

  private void renderSummary(RenderContext ctx, CpuTrack.Data data, double w, double h) {
    long tStart = data.request.range.start;
    ctx.setBackgroundColor(hsl(hue, .5f, .6f));
    ctx.path(path -> {
      int start = Math.max(0, (int)((state.getVisibleTime().start - tStart) / data.bucketSize));
      path.moveTo(0, h);
      double y = h, x = 0;
      for (int i = start; i < data.utilizations.length && x < w; i++) {
        x = state.timeToPx(tStart + i * data.bucketSize);
        double nextY = Math.round(h * (1 - data.utilizations[i]));
        path.lineTo(x, y);
        path.lineTo(x, nextY);
        y = nextY;
      }
      path.lineTo(x, h);
      path.close();
      ctx.fillPath(path);
    });
  }

  private void renderSlices(RenderContext ctx, CpuTrack.Data data, double h) {
    //boolean isHovering = feGlobals().getFrontendLocalState().hoveredUtid != -1;

    TimeSpan visible = state.getVisibleTime();
    for (int i = 0; i < data.starts.length; i++) {
      long tStart = data.starts[i];
      long tEnd = data.ends[i];
      long utid = data.utids[i];
      if (tEnd <= visible.start || tStart >= visible.end) {
        continue;
      }
      double rectStart = state.timeToPx(tStart);
      double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);

      ThreadInfo.Display threadInfo = ThreadInfo.getDisplay(state.getData(), utid, false);
      StyleConstants.HSL color = threadInfo.getColor();
      color = color.adjusted(color.h, color.s - 20, Math.min(color.l + 10,  60));

      ctx.setBackgroundColor(color.rgb());
      ctx.fillRect(rectStart, 0, rectWidth, h);

      // Don't render text when we have less than 7px to play with.
      if (rectWidth < 7) {
        continue;
      }

      ctx.setForegroundColor(colors().textInvertedMain);
      ctx.drawText(threadInfo.title, rectStart + 2, 2, rectWidth - 4, (h / 2) - 4);
      if (!threadInfo.subTitle.isEmpty()) {
        ctx.setForegroundColor(colors().textInvertedAlt);
        ctx.drawText(threadInfo.subTitle, rectStart + 2, (h / 2) + 2, rectWidth - 4, (h / 2) - 4);
      }
    }

    if (hoveredThread != null) {
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(mouseXpos + HOVER_MARGIN, 0, hoveredWidth + 2 * HOVER_PADDING, h);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(hoveredThread.title, mouseXpos + HOVER_MARGIN + HOVER_PADDING, 2, (h / 2) - 4);
      if (!hoveredThread.subTitle.isEmpty()) {
        ctx.drawText(hoveredThread.subTitle,
            mouseXpos+ HOVER_MARGIN + HOVER_PADDING, (h / 2) + 2, (h / 2) - 4);
      }
    }
  }

  @Override
  public Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    CpuTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null || data.kind == CpuTrack.Data.Kind.summary) {
      return Hover.NONE;
    }

    mouseXpos = x;
    long t = state.pxToTime(x);
    for (int i = 0; i < data.starts.length; i++) {
      if (data.starts[i] <= t && t <= data.ends[i]) {
        hoveredThread = ThreadInfo.getDisplay(state.getData(), data.utids[i], true);
        if (hoveredThread == null) {
          return Hover.NONE;
        }
        hoveredWidth =
            Math.max(m.measure(hoveredThread.title).w, m.measure(hoveredThread.subTitle).w);
        long id = data.ids[i];

        return new Hover() {
          @Override
          public Area getRedraw() {
            return new Area(x + HOVER_MARGIN, 0, hoveredWidth + 2 * HOVER_PADDING, HEIGHT);
          }

          @Override
          public void stop() {
            hoveredThread = null;
            mouseXpos = 0;
          }

          @Override
          public Cursor getCursor(Display display) {
            return display.getSystemCursor(SWT.CURSOR_HAND);
          }

          @Override
          public boolean click() {
            state.setSelection(CpuTrack.getSlice(state.getQueryEngine(), id));
            return false;
          }
        };
      }
    }
    return Hover.NONE;
  }

  @Override
  public void computeSelection(CombiningBuilder builder, Area area, TimeSpan ts) {
    if (area.h / height >= SELECTION_THRESHOLD) {
      builder.add(Kind.Cpu, transform(
          CpuTrack.getSlices(state.getQueryEngine(), track.getCpu(), ts),
          r -> new CpuTrack.Slices(state, r)));
    }
  }
}
