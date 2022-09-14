/*
 * Copyright (C) 2022 Google Inc.
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
import static com.google.gapid.perfetto.views.StyleConstants.mainGradient;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.CounterInfo;
import com.google.gapid.perfetto.models.EnergyBreakdownTrack;
import com.google.gapid.perfetto.models.Selection;
import java.util.List;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

public class EnergyBreakdownPanel extends TrackPanel<EnergyBreakdownPanel> implements Selectable {
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;

  protected final EnergyBreakdownTrack track;
  protected final double trackHeight;
  protected HoverCard hovered = null;
  private EnergyBreakdownTrack.Stats cachedStats;

  public EnergyBreakdownPanel(State state, EnergyBreakdownTrack track, double trackHeight) {
    super(state);
    this.track = track;
    this.trackHeight = trackHeight;
    this.cachedStats = new EnergyBreakdownTrack.Stats(track.getCounter());
  }

  @Override
  public EnergyBreakdownPanel copy() {
    return new EnergyBreakdownPanel(state, track, trackHeight);
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
    return trackHeight;
  }

  @Override
  protected void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace(
        "Energy Counter",
        () -> {
          EnergyBreakdownTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
          drawLoading(ctx, data, state, h);

          if (data == null || data.ts.length == 0) {
            return;
          }

          CounterInfo counter = track.getCounter();

          double min = counter.range.min;
          double range = counter.range.range();

          Selection<?> selected = state.getSelection(Selection.Kind.Counter);
          List<Integer> visibleSelected = Lists.newArrayList();
          mainGradient().applyBaseAndBorder(ctx);
          ctx.path(
              path -> {
                double lastX = state.timeToPx(data.ts[0]), lastY = h;
                path.moveTo(lastX, lastY);
                for (int i = 0; i < data.ts.length; i++) {
                  double nextX = state.timeToPx(data.ts[i]);
                  double nextY = (trackHeight - 1) * (1 - (data.values[i] - min) / range);
                  path.lineTo(nextX, lastY);
                  path.lineTo(nextX, nextY);
                  lastX = nextX;
                  lastY = nextY;
                  if (selected.contains(data.ids[i])) {
                    visibleSelected.add(i);
                  }
                }
                path.lineTo(lastX, h);
                ctx.fillPath(path);
                ctx.drawPath(path);
              });

          // Draw highlight line after the whole graph is rendered, so that the highlight is on the
          // top.
          ctx.setBackgroundColor(mainGradient().highlight);
          for (int index : visibleSelected) {
            double startX = state.timeToPx(data.ts[index]);
            double endX =
                (index >= data.ts.length - 1) ? startX : state.timeToPx(data.ts[index + 1]);
            double y = (trackHeight - 1) * (1 - (data.values[index] - min) / range);
            ctx.fillRect(startX, y - 1, endX - startX, 3);
          }

          if (hovered != null) {
            double y = (trackHeight - 1) * (1 - (hovered.value - min) / range);
            ctx.setBackgroundColor(mainGradient().highlight);
            ctx.fillRect(hovered.startX, y - 1, hovered.endX - hovered.startX, trackHeight - y + 1);
            ctx.setForegroundColor(colors().textMain);
            ctx.drawCircle(hovered.mouseX, y, CURSOR_SIZE / 2);

            ctx.setBackgroundColor(colors().hoverBackground);
            double bgH = Math.max(hovered.size.h, trackHeight);
            // The left x-axis coordinate of HoverCard.
            double hx = hovered.mouseX + CURSOR_SIZE / 2 + HOVER_MARGIN;
            if (hx >= w - (2 * HOVER_PADDING + hovered.size.w)) {
              hx =
                  hovered.mouseX
                      - CURSOR_SIZE / 2
                      - HOVER_MARGIN
                      - 2 * HOVER_PADDING
                      - hovered.size.w;
            }
            ctx.fillRect(
                hx, Math.min((trackHeight - bgH) / 2, 0), 2 * HOVER_PADDING + hovered.size.w, bgH);
            ctx.setForegroundColor(colors().textMain);
            // The left x-axis coordinate of the  label.
            double x = hx + HOVER_PADDING;
            y = (trackHeight - hovered.size.h) / 2;
            ctx.drawText(Fonts.Style.Normal, hovered.text, x, y);
          }
        });
  }

  @Override
  protected Hover onTrackMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    EnergyBreakdownTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
    if (data == null || data.ts.length == 0) {
      return Hover.NONE;
    }

    long time = state.pxToTime(x);
    if (time < data.ts[0] || time > data.ts[data.ts.length - 1]) {
      return Hover.NONE;
    }

    int idx = 0;
    for (; idx < data.ts.length - 1; idx++) {
      if (data.ts[idx + 1] > time) {
        break;
      }
    }

    if (idx >= data.ts.length) {
      return Hover.NONE;
    }

    long id = data.ids[idx];
    double startX = state.timeToPx(data.ts[idx]);
    double endX = (idx >= data.ts.length - 1) ? startX : state.timeToPx(data.ts[idx + 1]);
    hovered = new HoverCard(m, track.getCounter(), data.values[idx], startX, endX, x);

    return new Hover() {
      @Override
      public Area getRedraw() {
        // The area of the sample can be larger than the hover card. Hence the redraw area needs to
        // at least cover the whole sample because when we move between samples, the highlight of
        // the previous sample needs to be redrawn without highlight. If the redraw area only
        // covers the hover card then the area that is not intersected with the hover card will
        // not be redrawn, and hence the highlight remains in that area.

        // First, calculate the default left boundary x-axis coordinate and width of the
        // redrawn area.
        final double defaultX = hovered.mouseX - CURSOR_SIZE / 2;
        final double defaultW =
            CURSOR_SIZE + HOVER_MARGIN + HOVER_PADDING + hovered.size.w + HOVER_PADDING;

        // Assuming the hover card is drawn on the right of the hover point.

        // Determine the x-axis coordinate of the left boundary of the redrawn area.
        double redrawLx = Math.min(defaultX, startX);

        // Determine the x-axis coordinate of the right boundary of the redrawn area by comparing
        // the right end of the sample with the right boundary of the default redrawn area.
        double redrawRx = Math.max(defaultX + defaultW, endX);

        // Calculate the real redrawn width.
        double redrawW = redrawRx - redrawLx;

        if (defaultX >= state.getWidth() - defaultW) {

          // re-calculate the left boundary of the redrawn area by comparing the start end of the
          // sample with the left boundary of the default redrawn area when the hover card is drawn
          // on the left side of the hover point.
          redrawLx = Math.min(startX, hovered.mouseX + CURSOR_SIZE / 2 - defaultW);

          // re-calculate the right boundary of the redrawn area by comparing the end of the sample
          // with the right boundary of the default redrawn area when the hover card is drawn on
          // the left side of the hover point.
          redrawRx = Math.max(hovered.mouseX + CURSOR_SIZE / 2, endX);

          // Finally, re-calculate the redrawn width, plus radius of the cursor to avoid the
          // precision issue at the right edge of the cursor.
          redrawW = redrawRx - redrawLx + CURSOR_SIZE / 2;
        }

        return new Area(redrawLx, -TRACK_MARGIN, redrawW, trackHeight + 2 * TRACK_MARGIN);
      }

      @Override
      public void stop() {
        hovered = null;
      }

      @Override
      public Cursor getCursor(Display display) {
        return display.getSystemCursor(SWT.CURSOR_HAND);
      }

      @Override
      public boolean click() {
        if ((mods & SWT.MOD1) == SWT.MOD1) {
          state.addSelection(
              Selection.Kind.Counter,
              transform(
                  track.getValue(id, track.getCounter().id),
                  d -> new EnergyBreakdownTrack.Values(track.getCounter().name, d)));
        } else {
          state.setSelection(
              Selection.Kind.Counter,
              transform(
                  track.getValue(id, track.getCounter().id),
                  d -> new EnergyBreakdownTrack.Values(track.getCounter().name, d)));
        }
        return true;
      }
    };
  }

  @Override
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts) {
    builder.add(Selection.Kind.Counter, computeSelection(ts));
  }

  private ListenableFuture<EnergyBreakdownTrack.Values> computeSelection(TimeSpan ts) {
    return transform(
        track.getValues(ts),
        data -> new EnergyBreakdownTrack.Values(track.getCounter().name, data));
  }

  private static class HoverCard {
    public final double value;
    public final double startX, endX;
    public final double mouseX;
    public final double width;
    public final Size size;
    public final String text;

    public HoverCard(
        Fonts.TextMeasurer tm,
        CounterInfo counter,
        double value,
        double startX,
        double endX,
        double mouseX) {
      this.value = value;
      this.startX = startX;
      this.endX = endX;
      this.mouseX = mouseX;
      text = counter.unit.format(value) + " uws";
      Size valueSize = tm.measure(Fonts.Style.Normal, text);
      this.width = valueSize.w;
      this.size = new Size(width + HOVER_PADDING, valueSize.h + HOVER_PADDING);
    }
  }
}
