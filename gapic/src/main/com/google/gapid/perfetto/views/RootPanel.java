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
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.TrackConfig;

import org.eclipse.swt.SWT;

/**
 * The main {@link Panel} containing all track panels. Shows a {@link TimelinePanel} at the top,
 * and tracks below.
 */
public class RootPanel extends Panel.Base implements State.Listener {
  private static final double HIGHLIGHT_TOP = 20;
  private static final double HIGHLIGHT_BOTTOM = 28;
  private static final double HIGHLIGHT_CENTER = (HIGHLIGHT_TOP + HIGHLIGHT_BOTTOM) / 2;
  private static final double HIGHLIGHT_PADDING = 3;

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

    TimeSpan highlight = state.getHighlight();
    if (highlight != TimeSpan.ZERO) {
      double newClipX = Math.max(clip.x, LABEL_WIDTH);
      ctx.withClip(newClipX, clip.y, clip.w - (newClipX - clip.x), clip.h, () -> {
        double x1 = Math.rint(LABEL_WIDTH + state.timeToPx(highlight.start));
        double x2 = Math.rint(LABEL_WIDTH + state.timeToPx(highlight.end));

        ctx.setForegroundColor(colors().timeHighlight);
        ctx.drawLine(x1, HIGHLIGHT_TOP, x1, HIGHLIGHT_BOTTOM);
        ctx.drawLine(x2, HIGHLIGHT_TOP, x2, HIGHLIGHT_BOTTOM);
        ctx.drawLine(x1, HIGHLIGHT_CENTER, x2, HIGHLIGHT_CENTER);

        String label = TimeSpan.timeToString(highlight.getDuration());
        Size labelSize = ctx.measure(Fonts.Style.Normal, label);
        double labelX = (x1 + x2 - labelSize.w) / 2;
        double labelY = HIGHLIGHT_CENTER - labelSize.h / 2;
        if (labelSize.w + 3 * HIGHLIGHT_PADDING >= (x2 - x1)) {
          labelX = x2 + HIGHLIGHT_PADDING;
          if (labelX + labelSize.w > width) {
            labelX = x1 - labelSize.w - HIGHLIGHT_PADDING;
          }
        } else {
          double min = LABEL_WIDTH + 2 * HIGHLIGHT_PADDING;
          double max = width - labelSize.w - 2 * HIGHLIGHT_PADDING;
          if (labelX < min) {
            labelX = Math.min(x2 - 2 * HIGHLIGHT_PADDING - labelSize.w, min);
          } else if (labelX > max) {
            labelX = Math.max(x1 + 2 * HIGHLIGHT_PADDING, max);
          }
       }

        ctx.setBackgroundColor(colors().background);
        ctx.fillRect(labelX - HIGHLIGHT_PADDING + 1, labelY - HIGHLIGHT_PADDING,
            labelSize.w + 2 * HIGHLIGHT_PADDING - 1, labelSize.h + 2 * HIGHLIGHT_PADDING);
        ctx.setForegroundColor(colors().timeHighlight);
        ctx.drawText(Fonts.Style.Normal, label, labelX, labelY);

        ctx.setForegroundColor(colors().timeHighlightBorder);
        ctx.drawLine(x1, topHeight, x1, height);
        ctx.drawLine(x2, topHeight, x2, height);

        ctx.setBackgroundColor(colors().timeHighlightCover);
        if (x1 >= width || x2 <= LABEL_WIDTH) {
          ctx.fillRect(LABEL_WIDTH, 0, width - LABEL_WIDTH, height);
        } else {
          if (x1 > LABEL_WIDTH) {
            ctx.fillRect(LABEL_WIDTH, 0, x1 - LABEL_WIDTH, height);
          }
          if (x2 > LABEL_WIDTH && x2 < width) {
            ctx.fillRect(x2, 0, width - x2, height);
          }
        }
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
      if (sx < LABEL_WIDTH) {
        return Dragger.NONE;
      }

      boolean onTop = sy <= topHeight;
      return new Dragger() {
        @Override
        public Area onDrag(double x, double y) {
          return onTop ? updateHighlight(sx, x) : updateSelection(sx, sy, x, y);
        }

        @Override
        public Area onDragEnd(double x, double y) {
          Area redraw = onTop ? updateHighlight(sx, x) : updateSelection(sx, sy, x, y);
          if (!onTop) {
            finishSelection();
          }
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

  protected Area updateHighlight(double sx, double x) {
    double start = Math.max(0, Math.min(sx, x) - LABEL_WIDTH);
    double end = Math.max(sx, x) - LABEL_WIDTH;
    if (end <= start) {
      state.setHighlight(TimeSpan.ZERO);
    } else {
      state.setHighlight(new TimeSpan(state.pxToTime(start), state.pxToTime(end)));
    }
    return Area.FULL;
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
    Hover result = Hover.NONE;
    if (y >= topHeight) {
      result = bottom.onMouseMove(m, x, y - topHeight + state.getScrollOffset())
          .transformed(a -> a.translate(0, topHeight - state.getScrollOffset()));
    }
    return (x >= LABEL_WIDTH) ? result.withClick(() -> {
      TimeSpan highlight = state.getHighlight();
      if (!highlight.isEmpty() && !highlight.contains(state.pxToTime(x - LABEL_WIDTH))) {
        state.setHighlight(TimeSpan.ZERO);
        return true;
      }
      return false;
    }) : result;
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
