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
import static com.google.gapid.util.MoreFutures.transform;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.CpuSummaryTrack;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.Selection;

/**
 * Draws the CPU usage summary, aggregating the usage of all cores.
 */
public class CpuSummaryPanel extends TrackPanel implements Selectable {
  private static final double HEIGHT = 80;

  private final CpuSummaryTrack track;

  public CpuSummaryPanel(State state, CpuSummaryTrack track) {
    super(state);
    this.track = track;
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
      CpuSummaryTrack.Data data = track.getData(state, () -> {
        repainter.repaint(new Area(0, 0, width, height));
      });
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

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
    });
  }

  @Override
  protected Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    return Hover.NONE;
  }

  @Override
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts) {
    if (area.h / height >= SELECTION_THRESHOLD) {
      builder.add(Kind.Cpu, transform(CpuSummaryTrack.getSlices(state.getQueryEngine(), ts), r ->
          new CpuTrack.Slices(state, r)));
    }
  }
}
