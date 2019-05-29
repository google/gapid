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
import static com.google.gapid.perfetto.views.StyleConstants.colorForThread;
import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.ProcessSummaryTrack;
import com.google.gapid.perfetto.models.ThreadInfo;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

/**
 * Displays the CPU usage summary of a process, aggregating all threads.
 */
public class ProcessSummaryPanel extends TrackPanel {
  private static final double HEIGHT = 50;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;

  private final ProcessSummaryTrack track;
  protected double mouseXpos;
  protected ThreadInfo.Display hoveredThread;
  protected double hoveredWidth;

  public ProcessSummaryPanel(State state, ProcessSummaryTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public String getTitle() {
    return track.getProcess().getDisplay();
  }

  @Override
  public String getSubTitle() {
    int count = track.getProcess().utids.size();
    return count + " Thread" + (count == 1 ? "" : "s");
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("ProcessSummaryPanel", () -> {
      ProcessSummaryTrack.Data data = track.getData(state, () -> {
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

  private void renderSummary(
      RenderContext ctx, ProcessSummaryTrack.Data data, double w, double h) {
    // TODO: dedupe with CpuRenderer
    long tStart = data.request.range.start;
    ctx.setBackgroundColor(colors().cpuUsageFill);
    ctx.setForegroundColor(colors().cpuUsageStroke);
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
      ctx.drawPath(path);
    });
  }

  private void renderSlices(RenderContext ctx, ProcessSummaryTrack.Data data, double h) {
    // TODO: dedupe with CpuRenderer
    TimeSpan visible = state.getVisibleTime();
    double cpuH = (h - state.getData().numCpus + 1) / state.getData().numCpus;
    for (int i = 0; i < data.starts.length; i++) {
      long tStart = data.starts[i];
      long tEnd = data.ends[i];
      int cpu = data.cpus[i];
      long utid = data.utids[i];
      if (tEnd <= visible.start || tStart >= visible.end) {
        continue;
      }
      double rectStart = state.timeToPx(tStart);
      double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);

      StyleConstants.HSL color = colorForThread(state.getThreadInfo(utid));
      color = color.adjusted(color.h, color.s - 20, Math.min(color.l + 10,  60));
      double y = cpuH * cpu + cpu;
      ctx.setBackgroundColor(color.rgb());
      ctx.fillRect(rectStart, y, rectWidth, cpuH);
    }

    if (hoveredThread != null) {
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(mouseXpos + HOVER_MARGIN, 0, hoveredWidth + 2 * HOVER_PADDING, h);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(hoveredThread.title, mouseXpos + HOVER_MARGIN + HOVER_PADDING, 2, (h / 2) - 4);
      if (!hoveredThread.subTitle.isEmpty()) {
        ctx.drawText(hoveredThread.subTitle,
            mouseXpos + HOVER_MARGIN + HOVER_PADDING, (h / 2) + 2, (h / 2) - 4);
      }
    }
  }

  @Override
  public Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    ProcessSummaryTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null || data.kind == ProcessSummaryTrack.Data.Kind.summary) {
      return Hover.NONE;
    }

    int cpu = (int)(y * state.getData().numCpus / HEIGHT);
    if (cpu < 0 || cpu >= state.getData().numCpus) {
      return Hover.NONE;
    }

    mouseXpos = x;
    long t = state.pxToTime(x);
    for (int i = 0; i < data.starts.length; i++) {
      if (data.cpus[i] == cpu && data.starts[i] <= t && t <= data.ends[i]) {
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
}
