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

import static com.google.gapid.perfetto.canvas.Tooltip.LocationComputer.fixedLocation;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.TimelinePanel.drawGridLines;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Tooltip;
import com.google.gapid.perfetto.models.Track;

import java.util.function.Consumer;

/**
 * {@link Panel} displaying a {@link Track}.
 */
public abstract class TrackPanel<T extends TrackPanel<T>> extends Panel.Base
    implements TitledPanel, CopyablePanel<T> {
  private static final double HOVER_X_OFF = 10;
  private static final double HOVER_Y_OFF = 7;

  protected final State state;
  protected Tooltip tooltip;

  public TrackPanel(State state) {
    this.state = state;
  }

  @Override
  public final double getPreferredHeight() {
    return getHeight() + 2 * TRACK_MARGIN;
  }

  public abstract double getHeight();

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    double w = width - LABEL_WIDTH, h = height - 2 * TRACK_MARGIN;
    drawGridLines(ctx, state, LABEL_WIDTH, 0, w, height);
    ctx.withTranslation(LABEL_WIDTH, TRACK_MARGIN, () ->
      ctx.withClip(0, -TRACK_MARGIN, w, h + 2 * TRACK_MARGIN, () ->
        renderTrack(ctx, repainter, w, h)));

    if (tooltip != null) {
      ctx.addOverlay(() -> {
        tooltip.render(ctx);
      });
    }
  }

  protected abstract void renderTrack(RenderContext ctx, Repainter repainter, double w, double h);

  @Override
  public void visit(Visitor v, Area area) {
    area.intersect(LABEL_WIDTH, TRACK_MARGIN, width - LABEL_WIDTH, height - 2 * TRACK_MARGIN)
      .ifNotEmpty(a -> v.visit(this, a));
  }

  @Override
  public Hover onMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    if (x < LABEL_WIDTH) {
      String text = getTooltip();
      if (text.isEmpty()) {
        return Hover.NONE;
      }

      tooltip = Tooltip.forText(m, text, fixedLocation(x + HOVER_X_OFF, y + HOVER_Y_OFF));
      return new Hover() {
        @Override
        public Area getRedraw() {
          return tooltip.getArea();
        }

        @Override
        public boolean isOverlay() {
          return true;
        }

        @Override
        public void stop() {
          tooltip = null;
        }
      };
    } else if (y < TRACK_MARGIN || y > height - TRACK_MARGIN) {
      return Hover.NONE;
    }
    return onTrackMouseMove(m, repainter.translated(LABEL_WIDTH, TRACK_MARGIN),
        x - LABEL_WIDTH, y - TRACK_MARGIN, mods
      ).translated(LABEL_WIDTH, TRACK_MARGIN);
  }

  protected abstract Hover onTrackMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods);

  // Helper functions for the track.getData(..) calls.
  protected <D> Track.OnUiThread<D> onUiThread(Repainter repainter) {
    return onUiThread(state, () -> repainter.repaint(new Area(0, 0, width, height)));
  }

  public static <D> Track.OnUiThread<D> onUiThread(State state, Runnable repaint) {
    return new Track.OnUiThread<D>() {
      @Override
      public void onUiThread(ListenableFuture<D> future, Consumer<D> callback) {
        state.thenOnUiThread(future, callback);
      }

      @Override
      public void repaint() {
        repaint.run();
      }
    };
  }

  // Helper function to determine the color of a slice.
  protected StyleConstants.Gradient getSliceColor(String title, int depth) {
    int commaIndex = title.indexOf(',');
    int colorCode = (commaIndex == -1) ? title.hashCode() :
        title.substring(0, commaIndex).hashCode();
    return StyleConstants.gradient(colorCode ^ depth);
  }

  protected StyleConstants.Gradient getSliceColor(String title) {
    return getSliceColor(title, 0);
  }
}
