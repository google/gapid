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
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.common.collect.Lists;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.CounterInfo;
import com.google.gapid.perfetto.models.CounterTrack;
import com.google.gapid.perfetto.models.CounterTrack.Values;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;
import com.google.gapid.perfetto.views.StyleConstants.Palette.BaseColor;

import java.util.List;

public class CounterPanel extends TrackPanel<CounterPanel> implements Selectable {
  private static final double HEIGHT = 45;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;

  protected final CounterTrack track;
  protected HoverCard hovered = null;
  protected double mouseXpos = 0;

  public CounterPanel(State state, CounterTrack track) {
    super(state);
    this.track = track;
  }

  @Override
  public CounterPanel copy() {
    return new CounterPanel(state, track);
  }

  @Override
  public String getTitle() {
    return track.getCounter().name;
  }

  @Override
  public String getTooltip() {
    CounterInfo counter = track.getCounter();
    StringBuilder sb = new StringBuilder().append("\\b").append(counter.name);
    if (!counter.description.isEmpty()) {
      sb.append("\n").append(counter.description);
    }
    return sb.toString();
  }

  @Override
  public double getHeight() {
    return HEIGHT;
  }

  @Override
  protected void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("Counter", () -> {
      CounterTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      CounterInfo counter = track.getCounter();
      double min = Math.min(0, counter.min), range = counter.max - min;

      Selection<Values.Key> selected = state.getSelection(Selection.Kind.Counter);
      List<Integer> visibleSelected = Lists.newArrayList();
      ctx.setBackgroundColor(BaseColor.LIGHT_BLUE.rgb);
      ctx.setForegroundColor(BaseColor.PACIFIC_BLUE.rgb);
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
          if (selected.contains(new Values.Key(track.getCounter().name, data.ts[i]))) {
            visibleSelected.add(i);
          }
        }
        path.lineTo(lastX, h);
        ctx.fillPath(path);
        ctx.drawPath(path);
      });

      // Draw highlight line after the whole graph is rendered, so that the highlight is on the top.
      ctx.setBackgroundColor(BaseColor.INDIGO.rgb);
      for (int index : visibleSelected) {
        double startX = state.timeToPx(data.ts[index]);
        double endX = (index >= data.ts.length - 1) ? startX : state.timeToPx(data.ts[index + 1]);
        double y = (HEIGHT - 1) * (1 - (data.values[index] - min) / range);
        ctx.fillRect(startX, y - 1, endX - startX, 3);
      }

      if (hovered != null) {
        double y = (HEIGHT - 1) * (1 - (hovered.value - min) / range);
        ctx.setBackgroundColor(BaseColor.INDIGO.rgb);
        ctx.fillRect(hovered.startX, y - 1, hovered.endX - hovered.startX, 3);
        ctx.setForegroundColor(colors().textMain);
        ctx.drawCircle(mouseXpos, y, CURSOR_SIZE / 2);
      }

      String label = String.format("%,d", Math.round(counter.max));
      Size labelSize = ctx.measure(Fonts.Style.Normal, label);
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(0, 0, labelSize.w + 8, labelSize.h + 8);
      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, label, 4, 4);

      if (hovered != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(mouseXpos + HOVER_MARGIN, 0, 2 * HOVER_PADDING + hovered.size.w, HEIGHT);
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Normal, hovered.label,
            mouseXpos + HOVER_MARGIN + HOVER_PADDING, (HEIGHT - hovered.size.h) / 2);
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y) {
    CounterTrack.Data data = track.getData(state.toRequest(), onUiThread());
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

    if (idx >= data.ts.length) {
      return Hover.NONE;
    }

    long t = data.ts[idx];
    double startX = state.timeToPx(data.ts[idx]);
    double endX = (idx >= data.ts.length - 1) ? startX : state.timeToPx(data.ts[idx + 1]);
    hovered = new HoverCard(m, data.values[idx], startX, endX);
    mouseXpos = x;

    return new Hover() {
      @Override
      public Area getRedraw() {
        double start = Math.min(mouseXpos - CURSOR_SIZE / 2, startX);
        double end = Math.max(mouseXpos + CURSOR_SIZE / 2 +
            HOVER_MARGIN + HOVER_PADDING + hovered.size.w + HOVER_PADDING,
            endX);
        return new Area(start, -TRACK_MARGIN, end - start, HEIGHT + 2 * TRACK_MARGIN);
      }

      @Override
      public void stop() {
        hovered = null;
      }

      @Override
      public boolean click() {
        state.setSelection(Selection.Kind.Counter,
            transform(track.getValue(t), d -> new CounterTrack.Values(track.getCounter().name, d)));
        return true;
      }
    };
  }

  @Override
  public void computeSelection(CombiningBuilder builder, Area area, TimeSpan ts) {
    builder.add(Selection.Kind.Counter, transform(track.getValues(ts),
        data -> new CounterTrack.Values(track.getCounter().name, data)));
  }

  private static class HoverCard {
    private static final double MIN_DOUBLE_AS_LONG = 100_000.0;
    private static final double MAX_DOUBLE_AS_LONG = 9.2233720368547748E18;

    public final double value;
    public final double startX, endX;
    public final String label;
    public final Size size;

    public HoverCard(Fonts.TextMeasurer tm, double value, double startX, double endX) {
      this.value = value;
      this.startX = startX;
      this.endX = endX;
      this.label = "Value: " + format(value);
      this.size = tm.measure(Fonts.Style.Normal, label);
    }

    private static String format(double v) {
      double abs = Math.abs(v);
      if (abs >= MIN_DOUBLE_AS_LONG && abs <= MAX_DOUBLE_AS_LONG) {
        return String.format("%,d", Math.round(v));
      } else {
        return String.format("%,g", v);
      }
    }
  }
}
