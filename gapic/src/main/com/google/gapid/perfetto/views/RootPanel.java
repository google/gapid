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

import static com.google.gapid.perfetto.views.State.MAX_ZOOM_SPAN_NSEC;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.PanelGroup;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.TrackConfig;

import org.eclipse.swt.SWT;

/**
 * The main {@link Panel} containing all track panels. Shows a {@link TimelinePanel} at the top,
 * and tracks below.
 */
public class RootPanel extends Panel.Base implements State.Listener {
  private final PanelGroup top = new PanelGroup();
  private final PanelGroup bottom = new PanelGroup();
  private final State state;

  private Area selection = Area.NONE;

  public RootPanel(State state) {
    this.state = state;
    state.addListener(this);
  }

  public void clear() {
    top.clear();
    bottom.clear();
  }

  @Override
  public void onDataChanged() {
    clear();

    top.add(new TimelinePanel(state));
    for (TrackConfig.Element<?> el : state.getData().tracks.elements) {
      bottom.add(el.createUi(state));
    }
  }

  @Override
  public double getPreferredHeight() {
    return top.getPreferredHeight() + bottom.getPreferredHeight();
  }

  @Override
  public void setSize(double w, double h) {
    super.setSize(w, h);
    double topHeight = top.getPreferredHeight(), bottomHeight = bottom.getPreferredHeight();
    top.setSize(w, topHeight);
    bottom.setSize(w, h - topHeight);
    state.setWidth(w - LABEL_WIDTH);
    state.setMaxScrollOffset(bottomHeight - h + topHeight);
  }

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    double topHeight = top.getPreferredHeight();
    Area clip = ctx.getClip();
    if (clip.y < topHeight) {
      top.render(ctx, repainter);
    }
    if (clip.y + clip.h > topHeight) {
      double newClipY = Math.max(clip.y, topHeight);
      ctx.withClip(clip.x, newClipY, clip.w, clip.h - (newClipY - clip.y), () -> {
        ctx.withTranslation(0, topHeight - state.getScrollOffset(), () -> {
          bottom.render(ctx, repainter.transformed(
              a -> a.translate(0, topHeight - state.getScrollOffset())));
        });
      });
    }

    if (!selection.isEmpty()) {
      ctx.setBackgroundColor(colors().selectionBackground);
      ctx.fillRect(selection.x, selection.y, selection.w, selection.h);
    }
  }

  @Override
  public void visit(Visitor v, Area area) {
    double topHeight = top.getPreferredHeight();
    area.intersect(0, 0, width, topHeight).ifNotEmpty(a -> top.visit(v, a));
    area.intersect(0, topHeight, width, height - topHeight)
        .ifNotEmpty(a -> bottom.visit(v, a.translate(0, -topHeight + state.getScrollOffset())));
  }

  @Override
  public Dragger onDragStart(double sx, double sy, int mods) {
    // TODO: top vs bottom
    double topHeight = top.getPreferredHeight();
    if (mods == (SWT.BUTTON1 | SWT.SHIFT)) {
      return new Dragger() {
        @Override
        public Area onDrag(double x, double y) {
          return updateSelection(sx, sy, x, y);
        }

        @Override
        public Area onDragEnd(double x, double y) {
          Area redraw = updateSelection(sx, sy, x, y);
          finishSelection();
          return redraw;
        }
      };
    }
    return bottom.onDragStart(sx, sy - topHeight + state.getScrollOffset(), mods)
        .translated(0, topHeight - state.getScrollOffset());
  }

  protected Area updateSelection(double sx, double sy, double x, double y) {
    Area old = selection;
    if (x < sx && y < sy) {
      selection = new Area(x, y, sx - x, sy - y);
    } else if (x < sx && y >= sy) {
      selection = new Area(x, sy, sx - x, y - sy);
    } else if (x >= sx && y < sy) {
      selection = new Area(sx, y, x - sx, sy - y);
    } else {
      selection = new Area(sx, sy, x - sx, y - sy);
    }
    return old.combine(selection);
  }

  protected void finishSelection() {
    Area onTrack = selection.intersect(LABEL_WIDTH, 0, width - LABEL_WIDTH, height)
        .translate(-LABEL_WIDTH, 0);
    TimeSpan ts = new TimeSpan(
        state.pxToTime(onTrack.x), state.pxToTime(onTrack.x + onTrack.w));

    Selection.CombiningBuilder builder = new Selection.CombiningBuilder();
    visit(Visitor.of(Selectable.class, (s, a) -> s.computeSelection(builder, a, ts)), selection);
    selection = Area.NONE;
    state.setSelection(builder.build());
  }

  @Override
  public Hover onMouseMove(Fonts.TextMeasurer m, double x, double y) {
    double topHeight = top.getPreferredHeight();
    if (y < topHeight) {
      return Hover.NONE;
    } else {
      return bottom.onMouseMove(m, x, y - topHeight + state.getScrollOffset())
          .transformed(a -> a.translate(0, topHeight - state.getScrollOffset()));
    }
  }

  public boolean zoom(double x, double zoomFactor) {
    TimeSpan visible = state.getVisibleTime();
    long cursorTime = state.pxToTime(x - LABEL_WIDTH);
    long curSpan = visible.getDuration();
    double newSpan = Math.max(curSpan * zoomFactor, MAX_ZOOM_SPAN_NSEC);
    long newStart = Math.round(cursorTime - (newSpan / curSpan) * (cursorTime - visible.start));
    long newEnd = Math.round(newStart + newSpan);
    return state.setVisibleTime(new TimeSpan(newStart, newEnd));
  }
}
