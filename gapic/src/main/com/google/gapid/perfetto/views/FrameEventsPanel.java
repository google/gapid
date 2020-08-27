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

import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.ArgSet;
import com.google.gapid.perfetto.models.FrameEventsTrack;
import com.google.gapid.perfetto.models.FrameInfo;
import com.google.gapid.perfetto.models.Selection;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.widgets.Display;

import java.util.List;
import java.util.Set;

/**
 * Draws the instant events and the Displayed Frame track for the Surface Flinger Frame Events
 */
public class FrameEventsPanel extends TrackPanel<FrameEventsPanel>
    implements Selectable {
  private static final double SLICE_HEIGHT = 30;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final int BOUNDING_BOX_LINE_WIDTH = 3;

  private final FrameInfo.Event event;
  protected final FrameEventsTrack track;

  protected double mouseXpos, mouseYpos;
  protected String hoveredTitle;
  protected String hoveredCategory;
  protected Size hoveredSize = Size.ZERO;

  public FrameEventsPanel(State state, FrameInfo.Event event, FrameEventsTrack track) {
    super(state);
    this.event = event;
    this.track = track;
  }

  @Override
  public String getTitle() {
    return event.getDisplay();
  }

  @Override
  public String getTooltip() {
    return "\\b" + event.tooltip;
  }

  @Override
  public double getHeight() {
    return event.maxDepth * SLICE_HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("FrameEvents", () -> {
      FrameEventsTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }
      renderSlices(ctx, data, w);
    });
  }

  public void renderSlices(RenderContext ctx, FrameEventsTrack.Data data, double w) {
    TimeSpan visible = state.getVisibleTime();
    Selection<?> selected = state.getSelection(Selection.Kind.FrameEvents);
    List<Highlight> visibleSelected = Lists.newArrayList();

    Set<Long> selectedFrameNumbers = getSelectedFrameNumbers(state);

    for (int i = 0; i < data.starts.length; i++) {
      long tStart = data.starts[i];
      long tEnd = data.ends[i];
      int depth = data.depths[i];
      String title = buildSliceTitle(data.titles[i], data.args[i]);

      if (tEnd <= visible.start || tStart >= visible.end) {
        continue;
      }
      double rectStart = state.timeToPx(tStart);
      StyleConstants.Gradient color = getSliceColor(data.titles[i]);
      if (!selectedFrameNumbers.isEmpty() && !selectedFrameNumbers.contains(data.frameNumbers[i])) {
        ctx.setBackgroundColor(color.disabled);
      } else {
        color.applyBase(ctx);
      }

      if (tEnd - tStart > 0 ) { // Slice
        double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);
        double y = depth * SLICE_HEIGHT;
        ctx.fillRect(rectStart, y, rectWidth, SLICE_HEIGHT);

        if (selected.contains(data.ids[i]) || selectedFrameNumbers.contains(data.frameNumbers[i])) {
          visibleSelected.add(Highlight.slice(color.border, rectStart, y, rectWidth));
        }

        // Don't render text when we have less than 7px to play with.
        if (rectWidth < 7) {
          continue;
        }

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(
            Fonts.Style.Normal, title, rectStart + 2, y + 2, rectWidth - 4, SLICE_HEIGHT - 4);
      } else { // Instant event (diamond)
        double rectWidth = 20;
        double y = depth * SLICE_HEIGHT;
        double[] diamondX = { rectStart - (rectWidth / 2), rectStart, rectStart + (rectWidth / 2),
            rectStart};
        double[] diamondY = { y + (SLICE_HEIGHT / 2), y, y + (SLICE_HEIGHT / 2), y + SLICE_HEIGHT };
        ctx.fillPolygon(diamondX, diamondY, 4);

        if (selected.contains(data.ids[i]) || selectedFrameNumbers.contains(data.frameNumbers[i])) {
          visibleSelected.add(Highlight.diamond(color.highlight, diamondX, diamondY));
        }

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(
            Fonts.Style.Normal, title.substring(0,1), rectStart - rectWidth/2 + 1, y + 1,
            rectWidth - 1, SLICE_HEIGHT - 4);
      }
    }

    for (Highlight highlight : visibleSelected) {
      highlight.draw(ctx);
    }

    if (hoveredTitle != null) {
      double cardX = mouseXpos + HOVER_MARGIN;
      if (cardX >= w - (hoveredSize.w + 2 * HOVER_PADDING)) {
        cardX = mouseXpos - HOVER_MARGIN - hoveredSize.w - 2 * HOVER_PADDING;
      }
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(cardX, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, hoveredTitle, cardX + HOVER_PADDING,
          mouseYpos + HOVER_PADDING / 2);
      if (!hoveredCategory.isEmpty()) {
        ctx.setForegroundColor(colors().textAlt);
        ctx.drawText(Fonts.Style.Normal, hoveredCategory, cardX + HOVER_PADDING,
            mouseYpos + hoveredSize.h / 2, hoveredSize.h / 2);
      }
    }
  }

  private static String buildSliceTitle(String title, ArgSet args) {
    Object w = args.get("width"), h = args.get("height");
    return (w == null || h == null) ? title : title + " (" + w + "x" + h + ")";
  }

  @Override
  protected Hover onTrackMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    FrameEventsTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
    if (data == null) {
      return Hover.NONE;
    }
    return sliceHover(data, m, x, y, mods);
  }

  private Hover sliceHover(FrameEventsTrack.Data data, Fonts.TextMeasurer m, double x, double y, int mods) {
    int depth = (int)(y / SLICE_HEIGHT);
    if (depth < 0 || depth > event.maxDepth) {
      return Hover.NONE;
    }

    mouseXpos = x;
    mouseYpos = depth * SLICE_HEIGHT;
    for (int i = 0; i < data.starts.length; i++) {
      double ts = state.timeToPx(data.starts[i]);
      double endts = state.timeToPx(data.ends[i]);
      if (ts == endts) {
        endts = ts + 12;
        ts -= 12;
      }
      if (data.depths[i] == depth && x >= ts && x<= endts) {
        hoveredTitle = data.titles[i];
        hoveredCategory = "";
        if (hoveredTitle.isEmpty()) {
          if (hoveredCategory.isEmpty()) {
            return Hover.NONE;
          }
          hoveredCategory = "";
        }
        hoveredTitle = buildSliceTitle(hoveredTitle, data.args[i]);

        hoveredSize = Size.vertCombine(HOVER_PADDING, HOVER_PADDING / 2,
            m.measure(Fonts.Style.Normal, hoveredTitle),
            hoveredCategory.isEmpty() ? Size.ZERO : m.measure(Fonts.Style.Normal, hoveredCategory));
        mouseYpos = Math.max(0, Math.min(mouseYpos - (hoveredSize.h - SLICE_HEIGHT) / 2,
            (1 + event.maxDepth) * SLICE_HEIGHT - hoveredSize.h));
        long id = data.ids[i];
        return new Hover() {
          @Override
          public Area getRedraw() {
            double redrawW = HOVER_MARGIN + hoveredSize.w + 2 * HOVER_PADDING;
            double redrawX = x;
            if (redrawX >= state.getWidth() - redrawW) {
              redrawX = x - redrawW;
            }
            return new Area(redrawX, mouseYpos, redrawW, hoveredSize.h);
          }

          @Override
          public void stop() {
            hoveredTitle = hoveredCategory = null;
          }

          @Override
          public Cursor getCursor(Display display) {
            return (id < 0) ? null : display.getSystemCursor(SWT.CURSOR_HAND);
          }

          @Override
          public boolean click() {
            if (id < 0) {
              return false;
            }
            if ((mods & SWT.MOD1) == SWT.MOD1) {
              state.addSelection(Selection.Kind.FrameEvents, track.getSlice(id));
            } else {
              state.setSelection(Selection.Kind.FrameEvents, track.getSlice(id));
            }
            return true;
          }
        };
      }
    }
    return Hover.NONE;
  }

  @Override
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts) {
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
    if (startDepth > endDepth) {
      return;
    }

    if (endDepth >= 0) {
      if (endDepth >= event.maxDepth) {
        endDepth = Integer.MAX_VALUE;
      }

      builder.add(Selection.Kind.FrameEvents, track.getSlices(ts, startDepth, endDepth));
    }
  }

  private static Set<Long> getSelectedFrameNumbers(State state) {
    Selection<?> selection = state.getSelection(Selection.Kind.FrameEvents);
    Set<Long> selected = Sets.newHashSet();
    if (selection instanceof FrameEventsTrack.Slices) {
      selected = ((FrameEventsTrack.Slices)selection).getSelectedFrameNumbers();
    }
    return selected;
  }

  @Override
  public FrameEventsPanel copy() {
    return new FrameEventsPanel(state, event, track);
  }

  private interface Highlight {
    public void draw(RenderContext ctx);

    public static Highlight slice(RGBA color, double x, double y, double w) {
      return ctx -> {
        ctx.setForegroundColor(color);
        ctx.drawRect(x, y, w, SLICE_HEIGHT, BOUNDING_BOX_LINE_WIDTH);
      };
    }

    public static Highlight diamond(RGBA color, double[] xs, double[] ys) {
      return ctx -> {
        ctx.setForegroundColor(color);
        ctx.drawPolygon(xs, ys, 4);
      };
    }
  }
}
