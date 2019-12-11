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

import static com.google.gapid.perfetto.TimeSpan.timeToString;
import static com.google.gapid.perfetto.views.Loading.drawLoading;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.common.collect.Lists;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.CpuInfo;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.ProcessSummaryTrack;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.ThreadInfo;
import com.google.gapid.perfetto.views.StyleConstants.Palette.BaseColor;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

import java.util.List;

/**
 * Displays the CPU usage summary of a process, aggregating all threads.
 */
public class ProcessSummaryPanel extends TrackPanel<ProcessSummaryPanel> {
  private static final double HEIGHT = 50;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static final int BOUNDING_BOX_LINE_WIDTH = 1;

  private final ProcessSummaryTrack track;

  protected double mouseXpos;
  protected ThreadInfo.Display hoveredThread;
  protected double hoveredWidth;
  protected HoverCard hovered;

  public ProcessSummaryPanel(State state, ProcessSummaryTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public ProcessSummaryPanel copy() {
    return new ProcessSummaryPanel(state, track);
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
  public String getTooltip() {
    return "\\b" + getTitle() + "\n" + getSubTitle();
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
    int start = Math.max(0, (int)((state.getVisibleTime().start - tStart) / data.bucketSize));

    ctx.setBackgroundColor(BaseColor.LIGHT_BLUE.rgb);
    ctx.setForegroundColor(BaseColor.PACIFIC_BLUE.rgb);
    ctx.path(path -> {
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

    if (hovered != null && hovered.bucket >= start) {
      double x = state.timeToPx(tStart + hovered.bucket * data.bucketSize + data.bucketSize / 2);
      if (x < w) {
        double dx = HOVER_PADDING + hovered.size.w + HOVER_PADDING;
        double dy = HOVER_PADDING + hovered.size.h + HOVER_PADDING;
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(x + HOVER_MARGIN, h - HOVER_PADDING - dy, dx, dy);
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Normal, hovered.text, x + HOVER_MARGIN + HOVER_PADDING, h - dy);

        ctx.setForegroundColor(colors().textMain);
        ctx.drawCircle(x, h * (1 - hovered.utilization), CURSOR_SIZE / 2);
      }
    }
  }

  private void renderSlices(RenderContext ctx, ProcessSummaryTrack.Data data, double h) {
    // TODO: dedupe with CpuRenderer
    TimeSpan visible = state.getVisibleTime();
    Selection<Long> selected = state.getSelection(Selection.Kind.Cpu);
    List<Integer> visibleSelected = Lists.newArrayList();
    int cpuCount = state.getData().cpu.count();
    double cpuH = (h - cpuCount + 1) / cpuCount;
    for (int i = 0; i < data.starts.length; i++) {
      long tStart = data.starts[i];
      long tEnd = data.ends[i];
      CpuInfo.Cpu cpu = state.getData().cpu.getById(data.cpus[i]);
      long utid = data.utids[i];
      if (cpu == null || tEnd <= visible.start || tStart >= visible.end) {
        continue;
      }
      double rectStart = state.timeToPx(tStart);
      double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);

      double y = cpuH * cpu.index + cpu.index;
      ctx.setBackgroundColor(ThreadInfo.getColor(state, utid));
      ctx.fillRect(rectStart, y, rectWidth, cpuH);

      if (selected.contains(data.ids[i])) {
        visibleSelected.add(i);
      }
    }

    // Draw bounding rectangles after all the slices are rendered, so that the border is on the top.
    for (int index : visibleSelected) {
      ctx.setForegroundColor(ThreadInfo.getBorderColor(state, data.utids[index]));
      double rectStart = state.timeToPx(data.starts[index]);
      double rectWidth = Math.max(1, state.timeToPx(data.ends[index]) - rectStart);
      CpuInfo.Cpu cpu = state.getData().cpu.getById(data.cpus[index]);
      ctx.drawRect(
          rectStart, cpuH * cpu.index + cpu.index, rectWidth, cpuH, BOUNDING_BOX_LINE_WIDTH);
    }

    if (hoveredThread != null) {
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(mouseXpos + HOVER_MARGIN, 0, hoveredWidth + 2 * HOVER_PADDING, h);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, hoveredThread.title,
          mouseXpos + HOVER_MARGIN + HOVER_PADDING, 2, (h / 2) - 4);
      if (!hoveredThread.subTitle.isEmpty()) {
        ctx.drawText(Fonts.Style.Normal, hoveredThread.subTitle,
            mouseXpos + HOVER_MARGIN + HOVER_PADDING, (h / 2) + 2, (h / 2) - 4);
      }
    }
  }

  @Override
  public Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y) {
    ProcessSummaryTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null) {
      return Hover.NONE;
    }

    switch (data.kind) {
      case slice: return sliceHover(data, m, x, y);
      case summary: return summaryHover(data, m, x);
      default: return Hover.NONE;
    }
  }

  private Hover sliceHover(
      ProcessSummaryTrack.Data data, Fonts.TextMeasurer m, double x, double y) {
    int cpuCount = state.getData().cpu.count();
    int cpuIdx = (int)(y * cpuCount / HEIGHT);
    if (cpuIdx < 0 || cpuIdx >= cpuCount) {
      return Hover.NONE;
    }
    int cpu = state.getData().cpu.get(cpuIdx).id;

    mouseXpos = x;
    long t = state.pxToTime(x);
    for (int i = 0; i < data.starts.length; i++) {
      if (data.cpus[i] == cpu && data.starts[i] <= t && t <= data.ends[i]) {
        hoveredThread = ThreadInfo.getDisplay(state.getData(), data.utids[i], true);
        if (hoveredThread == null) {
          return Hover.NONE;
        }
        hoveredWidth = Math.max(
            m.measure(Fonts.Style.Normal, hoveredThread.title).w,
            m.measure(Fonts.Style.Normal, hoveredThread.subTitle).w);
        long id = data.ids[i];
        long utid = data.utids[i];

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
            state.setSelection(Selection.Kind.Cpu, CpuTrack.getSlice(state.getQueryEngine(), id));
            state.setSelectedThread(state.getThreadInfo(utid));
            return true;
          }
        };
      }
    }
    return Hover.NONE;
  }

  private Hover summaryHover(ProcessSummaryTrack.Data data, Fonts.TextMeasurer m, double x) {
    long time = state.pxToTime(x);
    int bucket = (int)((time - data.request.range.start) / data.bucketSize);
    if (bucket < 0 || bucket >= data.utilizations.length) {
      return Hover.NONE;
    }

    double p = data.utilizations[bucket];
    String text = (int)(p * 100) + "% (" +
        timeToString(Math.round(p * data.bucketSize)) + " / " + timeToString(data.bucketSize) + ")";
    hovered = new HoverCard(bucket, p, text, m.measure(Fonts.Style.Normal, text));

    double mouseX = state.timeToPx(
        data.request.range.start + hovered.bucket * data.bucketSize + data.bucketSize / 2);
    double dx = HOVER_PADDING + hovered.size.w + HOVER_PADDING;
    double dy = height;
    return new Hover() {
      @Override
      public Area getRedraw() {
        return new Area(mouseX - CURSOR_SIZE, -TRACK_MARGIN, CURSOR_SIZE + HOVER_MARGIN + dx, dy);
      }

      @Override
      public void stop() {
        hovered = null;
      }
    };
  }

  private static class HoverCard {
    public final int bucket;
    public final double utilization;
    public final String text;
    public final Size size;

    public HoverCard(int bucket, double utilization, String text, Size size) {
      this.bucket = bucket;
      this.utilization = utilization;
      this.text = text;
      this.size = size;
    }
  }
}
