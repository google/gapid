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
import static com.google.gapid.perfetto.views.StyleConstants.SELECTION_THRESHOLD;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.StyleConstants.gradient;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.ThreadInfo;
import com.google.gapid.util.Arrays;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.widgets.Display;

import java.util.List;

/**
 * Draws the CPU usage or slices of a single core.
 */
public class CpuPanel extends TrackPanel<CpuPanel> implements Selectable {
  private static final double HEIGHT = 30;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static final int BOUNDING_BOX_LINE_WIDTH = 3;

  protected final CpuTrack track;
  protected double mouseXpos;
  protected ThreadInfo.Display hoveredThread;
  protected double hoveredWidth;
  protected HoverCard hovered;

  public CpuPanel(State state, CpuTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public CpuPanel copy() {
    return new CpuPanel(state, track);
  }

  @Override
  public String getTitle() {
    return "CPU " + track.getCpu().id;
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("CpuTrack", () -> {
      CpuTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      switch (data.kind) {
        case slice: renderSlices(ctx, data, w, h); break;
        case summary: renderSummary(ctx, data, w, h); break;
      }
    });
  }

  private void renderSummary(RenderContext ctx, CpuTrack.Data data, double w, double h) {
    long tStart = data.request.range.start;
    int start = Math.max(0, (int)((state.getVisibleTime().start - tStart) / data.bucketSize));
    Selection<?> selected = state.getSelection(Selection.Kind.Cpu);
    List<Integer> visibleSelected = Lists.newArrayList();

    gradient(track.getCpu().id).applyBase(ctx);
    ctx.path(path -> {
      path.moveTo(0, h);
      double y = h, x = 0;
      for (int i = start; i < data.utilizations.length && x < w; i++) {
        x = state.timeToPx(tStart + i * data.bucketSize);
        double nextY = Math.round(h * (1 - data.utilizations[i]));
        path.lineTo(x, y);
        path.lineTo(x, nextY);
        y = nextY;
        for (String id : Arrays.getOrDefault(data.concatedIds, i, "").split(",")) {
          if (!id.isEmpty() && !selected.isEmpty() && selected.contains(Long.parseLong(id))) {
            visibleSelected.add(i);
            break;
          }
        }
      }
      path.lineTo(x, h);
      path.close();
      ctx.fillPath(path);
    });

    // Draw Highlight line after the whole graph is rendered, so that the highlight is on the top.
    ctx.setBackgroundColor(gradient(track.getCpu().id).highlight);
    for (int index : visibleSelected) {
      ctx.fillRect(state.timeToPx(tStart + index * data.bucketSize),
          Math.round(h * (1 - data.utilizations[index])) - 1,
          state.durationToDeltaPx(data.bucketSize), 3);
    }

    if (hovered != null && hovered.bucket >= start) {
      double mouseX = state.timeToPx(tStart + hovered.bucket * data.bucketSize + data.bucketSize / 2);
      double dx = HOVER_PADDING + hovered.size.w + HOVER_PADDING;
      double dy = HOVER_PADDING + hovered.size.h + HOVER_PADDING;
      double cardX = mouseX + CURSOR_SIZE / 2 + HOVER_MARGIN;
      if (cardX >= w - dx) {
        cardX = mouseX - CURSOR_SIZE / 2 - HOVER_MARGIN - dx;
      }
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(cardX, h - HOVER_PADDING - dy, dx, dy);
      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, hovered.text, cardX + HOVER_PADDING, h - dy);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawCircle(mouseX, h * (1 - hovered.utilization), CURSOR_SIZE / 2);
    }
  }

  private void renderSlices(RenderContext ctx, CpuTrack.Data data, double w, double h) {
    TimeSpan visible = state.getVisibleTime();
    Selection<?> selected = state.getSelection(Selection.Kind.Cpu);
    List<Highlight> visibleSelected = Lists.newArrayList();
    for (int i = 0; i < data.starts.length; i++) {
      long tStart = data.starts[i];
      long tEnd = data.ends[i];
      long utid = data.utids[i];
      if (tEnd <= visible.start || tStart >= visible.end) {
        continue;
      }
      double rectStart = state.timeToPx(tStart);
      double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);

      ThreadInfo.Display threadInfo = ThreadInfo.getDisplay(state, utid, false);

      ctx.setBackgroundColor(state.getSliceColorForThread(threadInfo.thread));
      ctx.fillRect(rectStart, 0, rectWidth, h);

      if (selected.contains(data.ids[i])) {
        visibleSelected.add(
            new Highlight(threadInfo.thread.getColor().border, rectStart, rectWidth));
      }

      // Don't render text when we have less than 7px to play with.
      if (rectWidth < 7) {
        continue;
      }

      ctx.setForegroundColor(colors().textMain);
      ctx.drawTextLeftTruncate(Fonts.Style.Normal, threadInfo.title, threadInfo.shortTitle,
          rectStart + 2, 2, rectWidth - 4, (h / 2) - 4);
      if (!threadInfo.subTitle.isEmpty()) {
        ctx.setForegroundColor(colors().textAlt);
        ctx.drawTextLeftTruncate(Fonts.Style.Normal, threadInfo.subTitle, threadInfo.shortSubTitle,
            rectStart + 2, (h / 2) + 2, rectWidth - 4, (h / 2) - 4);
      }
    }

    // Draw bounding rectangles after all the slices are rendered, so that the border is on the top.
    for (Highlight highlight : visibleSelected) {
      ctx.setForegroundColor(highlight.color);
      ctx.drawRect(highlight.x, 0, highlight.w, h, BOUNDING_BOX_LINE_WIDTH);
    }

    if (hoveredThread != null) {
      double cardX = mouseXpos + HOVER_MARGIN;
      if (cardX >= w - (hoveredWidth + 2 * HOVER_PADDING)) {
        cardX = mouseXpos - HOVER_MARGIN - hoveredWidth - 2 * HOVER_PADDING;
      }
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(cardX, 0, hoveredWidth + 2 * HOVER_PADDING, h);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, hoveredThread.title, cardX + HOVER_PADDING, 2, (h / 2) - 4);
      if (!hoveredThread.subTitle.isEmpty()) {
        ctx.drawText(Fonts.Style.Normal, hoveredThread.subTitle, cardX + HOVER_PADDING,
            (h / 2) + 2, (h / 2) - 4);
      }
    }
  }

  @Override
  public Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y, int mods) {
    CpuTrack.Data data = track.getData(state.toRequest(), onUiThread());
    if (data == null) {
      return Hover.NONE;
    }

    switch (data.kind) {
      case slice: return sliceHover(data, m, x, mods);
      case summary: return summaryHover(data, m, x, mods);
      default: return Hover.NONE;
    }
  }

  private Hover sliceHover(CpuTrack.Data data, Fonts.TextMeasurer m, double x, int mods) {
    mouseXpos = x;
    long t = state.pxToTime(x);
    for (int i = 0; i < data.starts.length; i++) {
      if (data.starts[i] <= t && t <= data.ends[i]) {
        hoveredThread = ThreadInfo.getDisplay(state, data.utids[i], true);
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
            double redrawW = HOVER_MARGIN + hoveredWidth + 2 * HOVER_PADDING;
            double redrawX = x;
            if (redrawX >= state.getWidth() - redrawW) {
              redrawX = x - redrawW;
            }
            return new Area(redrawX, 0, redrawW, HEIGHT);
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
            if ((mods & SWT.MOD1) == SWT.MOD1) {
              state.addSelection(Selection.Kind.Cpu, track.getSlice(id));
              state.addSelectedThread(state.getThreadInfo(utid));
            } else {
              state.setSelection(Selection.Kind.Cpu, track.getSlice(id));
              state.setSelectedThread(state.getThreadInfo(utid));
            }
            return true;
          }
        };
      }
    }
    return Hover.NONE;
  }

  private Hover summaryHover(CpuTrack.Data data, Fonts.TextMeasurer m, double x, int mods) {
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
    String ids = Arrays.getOrDefault(data.concatedIds, bucket, "");

    return new Hover() {
      @Override
      public Area getRedraw() {
        double redrawW = CURSOR_SIZE + HOVER_MARGIN + dx;
        double redrawX = mouseX - CURSOR_SIZE / 2;
        if (redrawX >= state.getWidth() - redrawW) {
          redrawX = mouseX + CURSOR_SIZE / 2 - redrawW;
          redrawW += CURSOR_SIZE / 2;
        }
        return new Area(redrawX, -TRACK_MARGIN, redrawW, dy);
      }

      @Override
      public void stop() {
        hovered = null;
      }

      @Override
      public Cursor getCursor(Display display) {
        return ids.isEmpty() ? null : display.getSystemCursor(SWT.CURSOR_HAND);
      }

      @Override
      public boolean click() {
        if (ids.isEmpty()) {
          return false;
        }
        if ((mods & SWT.MOD1) == SWT.MOD1) {
          state.addSelection(Selection.Kind.Cpu, transform(track.getSlices(ids), slices -> {
            slices.utids.forEach(utid -> state.addSelectedThread(state.getThreadInfo(utid)));
            return slices;
          }));
        } else {
          state.clearSelectedThreads();
          state.setSelection(Selection.Kind.Cpu, transform(track.getSlices(ids), slices -> {
            slices.utids.forEach(utid -> state.addSelectedThread(state.getThreadInfo(utid)));
            return slices;
          }));
        }
        return true;
      }
    };
  }

  @Override
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts) {
    if (area.h / height >= SELECTION_THRESHOLD) {
      builder.add(Selection.Kind.Cpu, computeSelection(ts));
    }
  }

  private ListenableFuture<CpuTrack.Slices> computeSelection(TimeSpan ts) {
    return transform(track.getSlices(ts), slices -> {
      slices.utids.forEach(utid -> state.addSelectedThread(state.getThreadInfo(utid)));
      return slices;
    });
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

  private static class Highlight {
    public final RGBA color;
    public final double x, w;

    public Highlight(RGBA color, double x, double w) {
      this.color = color;
      this.x = x;
      this.w = w;
    }
  }
}
