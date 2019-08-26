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

import static com.google.gapid.perfetto.views.StyleConstants.LABEL_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.TimelinePanel.drawGridLines;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.RenderContext;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

/**
 * {@link Panel} displaying a {@link Track}.
 */
public abstract class TrackPanel extends Panel.Base implements TitledPanel {
  protected final State state;

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
  }

  protected abstract void renderTrack(RenderContext ctx, Repainter repainter, double w, double h);

  @Override
  public void visit(Visitor v, Area area) {
    area.intersect(LABEL_WIDTH, TRACK_MARGIN, width - LABEL_WIDTH, height - 2 * TRACK_MARGIN)
      .ifNotEmpty(a -> v.visit(this, a));
  }

  @Override
  public Dragger onDragStart(double x, double y, int mods, double scrollTop) {
    if (x < LABEL_WIDTH || mods != SWT.BUTTON1) {
      // TODO: implement dragging of a track.
      return Dragger.NONE;
    }
    return new TrackDragger(state, x);
  }

  @Override
  public Hover onMouseMove(TextMeasurer m, double x, double y, double scrollTop) {
    if (x < LABEL_WIDTH || y < TRACK_MARGIN || y > height - TRACK_MARGIN) {
      return Hover.NONE;
    }
    return onTrackMouseMove(m, x - LABEL_WIDTH, y - TRACK_MARGIN)
        .translated(LABEL_WIDTH, TRACK_MARGIN);
  }

  protected abstract Hover onTrackMouseMove(TextMeasurer m, double x, double y);

  public static class TrackDragger implements Panel.Dragger {
    private final State state;
    private final double startX;
    private final TimeSpan atStart;

    public TrackDragger(State state, double startX) {
      this.state = state;
      this.startX = startX;
      this.atStart = state.getVisibleTime();
    }

    @Override
    public Area onDrag(double x, double y) {
      return (state.drag(atStart, x - startX)) ? Area.FULL : Area.NONE;
    }

    @Override
    public Area onDragEnd(double x, double y) {
      return onDrag(x, y);
    }

    @Override
    public Cursor getCursor(Display display) {
      return display.getSystemCursor(SWT.CURSOR_SIZEWE);
    }
  }
}
