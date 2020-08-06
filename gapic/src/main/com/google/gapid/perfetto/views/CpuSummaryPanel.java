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
import static com.google.gapid.perfetto.views.StyleConstants.mainGradient;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.CpuSummaryTrack;
import com.google.gapid.perfetto.models.CpuTrack.Slices;
import com.google.gapid.perfetto.models.Selection;

/**
 * Draws the CPU usage summary, aggregating the usage of all cores.
 */
public class CpuSummaryPanel extends TrackPanel<CpuSummaryPanel> implements Selectable {
  private static final double HEIGHT = 80;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;

  private final CpuSummaryTrack track;
  protected HoverCard hovered = null;

  public CpuSummaryPanel(State state, CpuSummaryTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public CpuSummaryPanel copy() {
    return new CpuSummaryPanel(state, track);
  }

  @Override
  public String getTitle() {
    return "CPU Usage";
  }

  @Override
  public String getSubTitle() {
    return track.getNumCpus() + " cores";
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("CpuSummary", () -> {
      CpuSummaryTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      // TODO: dedupe with CpuRenderer
      long tStart = data.request.range.start;
      int start = Math.max(0, (int)((state.getVisibleTime().start - tStart) / data.bucketSize));

      mainGradient().applyBaseAndBorder(ctx);
      ctx.path(path -> {
        path.moveTo(0, h);
        double y = h, x = 0;
        for (int i = start; i < data.utilizations.length && x < w; i++) {
          x = state.timeToPx(tStart + i * data.bucketSize);
          double nextY = h * (1 - data.utilizations[i]);
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
    });
  }

  @Override
  protected Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y, int mods) {
    CpuSummaryTrack.Data data = track.getData(state.toRequest(), onUiThread());
    if (data == null || data.utilizations.length == 0) {
      return Hover.NONE;
    }

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
        double redrawW = CURSOR_SIZE + HOVER_MARGIN + dx;
        double redrawX = mouseX - CURSOR_SIZE / 2;
        if (redrawX >= state.getWidth() - redrawW) {
          redrawX = mouseX + CURSOR_SIZE / 2 - redrawW;
          // If the hover card is drawn on the left side of the hover point, when moving the mouse
          // from left to right, the right edge of the cursor doesn't seem to get redrawn all the
          // time, this looks like a precision issue. Plus the radius of cursor here to avoid it.
          redrawW += CURSOR_SIZE / 2;
        }
        return new Area(redrawX, -TRACK_MARGIN, redrawW, dy);
      }

      @Override
      public void stop() {
        hovered = null;
      }
    };
  }

  @Override
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts) {
    if (area.h / height >= SELECTION_THRESHOLD) {
      builder.add(Selection.Kind.Cpu, computeSelection(ts));
    }
  }

  private ListenableFuture<Slices> computeSelection(TimeSpan ts) {
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
}
