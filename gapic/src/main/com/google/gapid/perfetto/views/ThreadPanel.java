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
import static com.google.gapid.perfetto.views.StyleConstants.TRACK_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.util.Colors.hsl;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;
import com.google.gapid.perfetto.models.ThreadTrack;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

/**
 * Displays the thread state and slices of a thread.
 */
public class ThreadPanel extends TrackPanel implements Selectable {
  private static final double SLICE_HEIGHT = 25 - 2 * TRACK_MARGIN;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double MERGE_SLICE_THRESHOLD = 3;
  private static final double MERGE_GAP_THRESHOLD = 4;

  protected final ThreadTrack track;
  private boolean expanded;

  protected double mouseXpos, mouseYpos;
  protected String hoveredTitle;
  protected String hoveredCategory;
  protected Size hoveredSize = Size.ZERO;

  public ThreadPanel(State state, ThreadTrack track) {
    super(state);
    this.track = track;
    this.expanded = false;
  }

  public void setCollapsed(boolean collapsed) {
    this.expanded = !collapsed;
  }

  @Override
  public String getTitle() {
    return track.getThread().getDisplay();
  }

  @Override
  public double getHeight() {
    return (expanded ? 1 + track.getThread().maxDepth : 1) * SLICE_HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("ThreadPanel", () -> {
      ThreadTrack.Data data = track.getData(state, () -> {
        repainter.repaint(new Area(0, 0, width, height));
      });
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      TimeSpan visible = state.getVisibleTime();
      boolean merging = false;
      double mergeStartX = 0;
      double mergeEndX = 0;
      ThreadState mergeState = ThreadState.NONE;
      for (int i = 0; i < data.schedStarts.length; i++) {
        long tStart = data.schedStarts[i];
        long tEnd = data.schedEnds[i];
        if (tEnd <= visible.start || tStart >= visible.end) {
          continue;
        }
        double rectStart = state.timeToPx(tStart);
        double rectEnd = state.timeToPx(tEnd);
        double rectWidth = rectEnd - rectStart;

        if (merging && (rectStart - mergeEndX) > MERGE_GAP_THRESHOLD) {
          double mergeWidth = Math.max(1, mergeEndX - mergeStartX);
          ctx.setBackgroundColor(mergeState.color.get());
          ctx.fillRect(mergeStartX, 0, mergeWidth, SLICE_HEIGHT);
          if (mergeWidth > 7) {
            ctx.setForegroundColor(colors().textInvertedMain);
            ctx.drawText(mergeState.label, rectStart + 2, 2, rectWidth - 4, SLICE_HEIGHT - 4);
          }
          merging = false;
        }
        if (rectWidth < MERGE_SLICE_THRESHOLD) {
          if (merging) {
            mergeEndX = rectEnd;
            mergeState = mergeState.merge(data.schedStates[i]);
          } else {
            merging = true;
            mergeStartX = rectStart;
            mergeEndX = rectEnd;
            mergeState = data.schedStates[i];
          }
        } else {
          ThreadState ts = data.schedStates[i];
          if (merging) {
            rectStart = mergeStartX;
            rectWidth = rectEnd - rectStart;
            ts = ts.merge(mergeState);
            merging = false;
          }

          ctx.setBackgroundColor(ts.color.get());
          ctx.fillRect(rectStart, 0, rectWidth, SLICE_HEIGHT);
          if (rectWidth > 7) {
            ctx.setForegroundColor(colors().textInvertedMain);
            ctx.drawText(ts.label, rectStart + 2, 2, rectWidth - 4, SLICE_HEIGHT - 4);
          }
        }
      }
      if (merging) {
        ctx.setBackgroundColor(mergeState.color.get());
        ctx.fillRect(mergeStartX, 0, mergeEndX - mergeStartX, SLICE_HEIGHT);
      }

      if (expanded) {
        for (int i = 0; i < data.starts.length; i++) {
          long tStart = data.starts[i];
          long tEnd = data.ends[i];
          int depth = data.depths[i];
          //String cat = data.categories[i];
          String title = data.titles[i];
          if (tEnd <= visible.start || tStart >= visible.end) {
            continue;
          }
          double rectStart = state.timeToPx(tStart);
          double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);
          double y = (1 + depth) * SLICE_HEIGHT;

          float hue = (title.hashCode() & 0x7fffffff) % 360;
          float saturation = Math.min(20 + depth * 10, 70) / 100f;
          ctx.setBackgroundColor(hsl(hue, saturation, .65f));
          ctx.fillRect(rectStart, y, rectWidth, SLICE_HEIGHT);

          // Don't render text when we have less than 7px to play with.
          if (rectWidth < 7) {
            continue;
          }

          ctx.setForegroundColor(colors().textInvertedMain);
          ctx.drawText(title, rectStart + 2, y + 2, rectWidth - 4, SLICE_HEIGHT - 4);
        }
      }

      if (hoveredTitle != null) {
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(
            mouseXpos + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(
            hoveredTitle, mouseXpos + HOVER_MARGIN + HOVER_PADDING, mouseYpos + HOVER_PADDING / 2);
        if (!hoveredCategory.isEmpty()) {
          ctx.setForegroundColor(colors().textAlt);
          ctx.drawText(hoveredCategory, mouseXpos + HOVER_MARGIN + HOVER_PADDING,
              mouseYpos + hoveredSize.h / 2, hoveredSize.h / 2);
        }
      }
    });
  }

  @Override
  protected Hover onTrackMouseMove(TextMeasurer m, double x, double y) {
    ThreadTrack.Data data = track.getData(state, () -> { /* nothing */ });
    if (data == null) {
      return Hover.NONE;
    }

    int depth = (int)(y / SLICE_HEIGHT);
    if (depth < 0 || depth > track.getThread().maxDepth) {
      return Hover.NONE;
    }

    mouseXpos = x;
    mouseYpos = depth * SLICE_HEIGHT;
    long t = state.pxToTime(x);
    if (depth == 0) {
      for (int i = 0; i < data.schedStarts.length; i++) {
        if (data.schedStarts[i] <= t && t <= data.schedEnds[i]) {
          int index = i;
          hoveredTitle = data.schedStates[i].label;
          hoveredCategory = "";
          hoveredSize = m.measure(hoveredTitle);

          return new Hover() {
            @Override
            public Area getRedraw() {
              return new Area(
                  x + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);
            }

            @Override
            public Cursor getCursor(Display display) {
              return display.getSystemCursor(SWT.CURSOR_HAND);
            }

            @Override
            public void stop() {
              hoveredTitle = null;
            }

            @Override
            public boolean click() {
              if (data.schedIds[index] != 0) {
                state.setSelection(CpuTrack.getSlice(state.getQueryEngine(), data.schedIds[index]));
              } else {
                state.setSelection(new ThreadTrack.StateSlice(data.schedStarts[index],
                    data.schedEnds[index] - data.schedStarts[index], track.getThread().utid,
                    data.schedStates[index]));
              }
              return false;
            }
          };
        }
      }
    } else if (expanded) {
      depth--;
      for (int i = 0; i < data.starts.length; i++) {
        if (data.depths[i] == depth && data.starts[i] <= t && t <= data.ends[i]) {
          hoveredTitle = data.titles[i];
          hoveredCategory = data.categories[i];
          if (hoveredTitle.isEmpty()) {
            if (hoveredCategory.isEmpty()) {
              return Hover.NONE;
            }
            hoveredTitle = hoveredCategory;
            hoveredCategory = "";
          }

          hoveredSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
              m.measure(hoveredTitle),
              hoveredCategory.isEmpty() ? Size.ZERO : m.measure(hoveredCategory));
          mouseYpos = Math.max(0, Math.min(mouseYpos - (hoveredSize.h - SLICE_HEIGHT) / 2,
              (1 + track.getThread().maxDepth) * SLICE_HEIGHT - hoveredSize.h));
          long id = data.ids[i];
          long ts = data.starts[i];

          return new Hover() {
            @Override
            public Area getRedraw() {
              return new Area(
                  x + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);
            }

            @Override
            public void stop() {
              hoveredTitle = hoveredCategory = null;
            }

            @Override
            public Cursor getCursor(Display display) {
              return (id == 0) ? null : display.getSystemCursor(SWT.CURSOR_HAND);
            }

            @Override
            public boolean click() {
              if (id != 0) {
                state.setSelection(ThreadTrack.getSlice(state.getQueryEngine(), id, ts));
              }
              return false;
            }
          };
        }
      }
    }
    return Hover.NONE;
  }

  @Override
  public void computeSelection(CombiningBuilder builder, Area area, TimeSpan ts) {
    int startDepth = (int)(area.y / SLICE_HEIGHT);
    int endDepth = (int)((area.y + area.h) / SLICE_HEIGHT);
    if (startDepth == endDepth && area.h / SLICE_HEIGHT < SELECTION_THRESHOLD) {
      return;
    }
    if (((startDepth + 1) * SLICE_HEIGHT - area.y) / SLICE_HEIGHT < SELECTION_THRESHOLD) {
      startDepth++;
    }
    if ((area.y + area.h - endDepth * SLICE_HEIGHT) / SLICE_HEIGHT < SELECTION_THRESHOLD) {
      endDepth--;
    }
    if (startDepth > endDepth || !expanded && startDepth > 0) {
      return;
    }

    if (startDepth == 0) {
      builder.add(Kind.ThreadState,
          transform(track.getStates(state.getQueryEngine(), ts), ThreadTrack.StateSlices::new));
    }

    startDepth = Math.max(0, startDepth - 1);
    endDepth--;
    if (endDepth >= 0) {
      if (endDepth >= track.getThread().maxDepth) {
        endDepth = Integer.MAX_VALUE;
      }
      builder.add(Kind.Thread, transform(
          ThreadTrack.getSlices(
              state.getQueryEngine(), track.getThread().utid, ts, startDepth, endDepth),
          ThreadTrack.Slices::new));
    }
  }
}
