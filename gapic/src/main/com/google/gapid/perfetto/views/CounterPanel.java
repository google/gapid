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
import static com.google.gapid.util.MoreFutures.transform;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.CounterTrack;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;

public class CounterPanel extends TrackPanel implements Selectable {
  private static final double HEIGHT = 30;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;

  private final CounterTrack track;
  private final String name;
  protected HoverCard hovered = null;
  protected double mouseXpos = 0;

  public CounterPanel(State state, CounterTrack track, String name) {
    super(state);
    this.track = track;
    this.name = name;
  }

  @Override
  public String getTitle() {
    return name;
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  protected void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("Counter", () -> {
      CounterTrack.Data data = track.getData(state, () -> {
        repainter.repaint(new Area(0, 0, width, height));
      });
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      double min = Math.min(0, track.getMin()), range = track.getMax() - min;
      ctx.setBackgroundColor(colors().counterFill);
      ctx.setForegroundColor(colors().counterStroke);
      ctx.path(path -> {
        path.moveTo(0, h);
        double lastX = 0, lastY = h;
        for (int i = 0; i < data.ts.length; i++) {
          double nextX = state.timeToPx(data.ts[i]);
          double nextY = (HEIGHT - 1) * (1 - (data.values[i] - min) / range);
          path.lineTo(nextX, lastY);
          path.lineTo(nextX, nextY);
          lastX = nextX;
          lastY = nextY;
        }
        path.lineTo(lastX, h);
        ctx.fillPath(path);
        ctx.drawPath(path);
      });

      if (hovered != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(mouseXpos + HOVER_MARGIN, 0, 2 * HOVER_PADDING + hovered.size.w, HEIGHT);
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(
            hovered.label, mouseXpos + HOVER_MARGIN + HOVER_PADDING, (HEIGHT - hovered.size.h) / 2);

        ctx.drawCircle(
            mouseXpos, (HEIGHT - 1) * (1 - (hovered.value - min) / range), CURSOR_SIZE / 2);
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    CounterTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null || data.ts.length == 0) {
      return Hover.NONE;
    }

    long time = state.pxToTime(x);
    int idx = 0;
    for (; idx < data.ts.length - 1; idx++) {
      if (data.ts[idx + 1] > time) {
        break;
      }
    }

    hovered = new HoverCard(m, data.values[idx]);
    mouseXpos = state.timeToPx(data.ts[idx]);
    return new Hover() {
      @Override
      public Area getRedraw() {
        return new Area(mouseXpos - CURSOR_SIZE, 0,
            CURSOR_SIZE + HOVER_MARGIN + HOVER_PADDING + hovered.size.w + HOVER_PADDING, HEIGHT);
      }

      @Override
      public void stop() {
        hovered = null;
      }
    };
  }

  @Override
  public void computeSelection(CombiningBuilder builder, Area area, TimeSpan ts) {
    builder.add(Kind.Counter, transform(
        track.getValues(state.getQueryEngine(), ts), data -> new CounterTrack.Values(name, data)));
  }

  private static class HoverCard {
    public final double value;
    public final String label;
    public final Size size;

    public HoverCard(TextMeasurer tm, double value) {
      this.value = value;
      this.label = String.format("Value: %,d", Math.round(value));
      this.size = tm.measure(label);
    }
  }
}
