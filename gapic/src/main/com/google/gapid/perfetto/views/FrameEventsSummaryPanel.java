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
import static com.google.gapid.perfetto.views.StyleConstants.mainGradient;

import com.google.common.collect.Lists;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.ArgSet;
import com.google.gapid.perfetto.models.FrameEventsTrack;
import com.google.gapid.perfetto.models.FrameEventsTrack.FrameSelection;
import com.google.gapid.perfetto.models.FrameInfo;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.Selection.CombiningBuilder;
import com.google.gapid.util.Arrays;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.widgets.Display;

import java.util.List;

/**
 * Draws the instant events and the Displayed Frame track for the Surface Flinger Frame Events
 */
public class FrameEventsSummaryPanel extends TrackPanel<FrameEventsSummaryPanel>
    implements Selectable {
  private static final double SLICE_HEIGHT = 30;
  private static final double HOVER_MARGIN = 10;
  private static final double HOVER_PADDING = 4;
  private static final double CURSOR_SIZE = 5;
  private static final int BOUNDING_BOX_LINE_WIDTH = 3;

  private final FrameInfo.Buffer buffer;
  protected final FrameEventsTrack track;

  protected double mouseXpos, mouseYpos;
  protected String hoveredTitle;
  protected String hoveredCategory;
  protected Size hoveredSize = Size.ZERO;
  protected HoverCard hovered;

  public FrameEventsSummaryPanel(State state, FrameInfo.Buffer buffer, FrameEventsTrack track) {
    super(state);
    this.buffer = buffer;
    this.track = track;
  }

  @Override
  public String getTitle() {
    return buffer.getDisplay();
  }

  @Override
  public String getTooltip() {
    return "\\b" + buffer.getDisplay();
  }

  @Override
  public double getHeight() {
    return buffer.maxDepth * SLICE_HEIGHT;
  }

  @Override
  public void renderTrack(RenderContext ctx, Repainter repainter, double w, double h) {
    ctx.trace("FrameEventsSummary", () -> {
      FrameEventsTrack.Data data = track.getData(state.toRequest(), onUiThread(repainter));
      drawLoading(ctx, data, state, h);

      if (data == null) {
        return;
      }

      switch(data.kind) {
        case slices: renderSlices(ctx, data); break;
        case summary: renderSummary(ctx, data, w, h); break;
      }
    });
  }


  private void renderSummary(
      RenderContext ctx, FrameEventsTrack.Data data, double w, double h) {
    long tStart = data.request.range.start;
    int start = Math.max(0, (int)((state.getVisibleTime().start - tStart) / data.bucketSize));
    Selection<?> selected = state.getSelection(Selection.Kind.FrameEvents);
    List<Integer> visibleSelected = Lists.newArrayList();

    mainGradient().applyBaseAndBorder(ctx);
    ctx.path(path -> {
      path.moveTo(0, h);
      double y = h, x = 0;
      for (int i = start; i < data.numEvents.length && x < w; i++) {
        x = state.timeToPx(tStart + i * data.bucketSize);
        double nextY = Math.round(Math.max(0,h - (h * (data.numEvents[i]))));
        path.lineTo(x, y);
        path.lineTo(x, nextY);
        y = nextY;
        for (String id : Arrays.getOrDefault(data.concatedIds, i, "").split(",")) {
          if (!id.isEmpty() && !selected.isEmpty() && selected.contains(Long.parseLong(id))) {
            visibleSelected.add(i);
            break;
          }
        }
      }
      path.lineTo(x, h);
      path.close();
      ctx.fillPath(path);
      ctx.drawPath(path);
    });

    // Draw Highlight line after the whole graph is rendered, so that the highlight is on the top.
    ctx.setBackgroundColor(mainGradient().highlight);
    for (int index : visibleSelected) {
      ctx.fillRect(state.timeToPx(tStart + index * data.bucketSize),
          Math.round(Math.max(0,h - (h * (data.numEvents[index])))) - 1,
          state.durationToDeltaPx(data.bucketSize), 3);
    }

    if (hovered != null && hovered.bucket >= start) {
      double x = state.timeToPx(tStart + hovered.bucket * data.bucketSize + data.bucketSize / 2);
      if (x < w) {
        double dx = HOVER_PADDING + hovered.size.w + HOVER_PADDING;
        double dy = HOVER_PADDING + hovered.size.h + HOVER_PADDING;
        ctx.setBackgroundColor(colors().hoverBackground);
        ctx.fillRect(x + HOVER_MARGIN, h - HOVER_PADDING - dy, dx, dy);
        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(Fonts.Style.Normal, hovered.text, x + HOVER_MARGIN + HOVER_PADDING, h - dy);

        ctx.setForegroundColor(colors().textMain);
        ctx.drawCircle(x, Math.round(Math.max(0,h - (h * (hovered.numEvents)))), CURSOR_SIZE / 2);
      }
    }
  }


  public void renderSlices(RenderContext ctx, FrameEventsTrack.Data data) {
    TimeSpan visible = state.getVisibleTime();
    Selection<?> selected = state.getSelection(Selection.Kind.FrameEvents);
    List<Highlight> visibleSelected = Lists.newArrayList();

    FrameSelection selectedFrames = getSelectedFrames(state);

    for (int i = 0; i < data.starts.length; i++) {
      long tStart = data.starts[i];
      long tEnd = data.ends[i];
      String title = buildSliceTitle(data.titles[i], data.args[i]);

      if (tEnd <= visible.start || tStart >= visible.end) {
        continue;
      }
      double rectStart = state.timeToPx(tStart);
      StyleConstants.Gradient color = getSliceColor(data.titles[i]);
      color.applyBase(ctx);
      if (!selectedFrames.isEmpty()) {
        ctx.setBackgroundColor(color.disabled);
        if (selectedFrames.contains(data.frameNumbers[i], data.layerNames[i])) {
          color.applyBase(ctx);
        }
      }

      if (tEnd - tStart > 3 ) {
        double rectWidth = Math.max(1, state.timeToPx(tEnd) - rectStart);
        double y = 0;
        ctx.fillRect(rectStart, y, rectWidth, SLICE_HEIGHT);

        if (selected.contains(data.ids[i])) {
          visibleSelected.add(Highlight.slice(color.border, rectStart, y, rectWidth));
        }

        // Don't render text when we have less than 7px to play with.
        if (rectWidth < 7) {
          continue;
        }

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(
            Fonts.Style.Normal, title, rectStart + 2, y + 2, rectWidth - 4, SLICE_HEIGHT - 4);
      } else {
        double rectWidth = 20;
        double y = 0;
        double[] diamondX = { rectStart - (rectWidth / 2), rectStart, rectStart + (rectWidth / 2), rectStart};
        double[] diamondY = { y + (SLICE_HEIGHT / 2), y, y + (SLICE_HEIGHT / 2), SLICE_HEIGHT };
        ctx.fillPolygon(diamondX, diamondY, 4);

        if (selected.contains(data.ids[i])) {
          visibleSelected.add(Highlight.diamond(color.highlight, diamondX, diamondY));
        }

        ctx.setForegroundColor(colors().textMain);
        ctx.drawText(
            Fonts.Style.Normal, title.substring(0,1), rectStart - rectWidth/2 + 1, y + 1, rectWidth - 1, SLICE_HEIGHT - 4);
      }
    }

    for (Highlight highlight : visibleSelected) {
      highlight.draw(ctx);
    }

    if (hoveredTitle != null) {
      ctx.setBackgroundColor(colors().hoverBackground);
      ctx.fillRect(
          mouseXpos + HOVER_MARGIN, mouseYpos, hoveredSize.w + 2 * HOVER_PADDING, hoveredSize.h);

      ctx.setForegroundColor(colors().textMain);
      ctx.drawText(Fonts.Style.Normal, hoveredTitle,
          mouseXpos + HOVER_MARGIN + HOVER_PADDING, mouseYpos + HOVER_PADDING / 2);
      if (!hoveredCategory.isEmpty()) {
        ctx.setForegroundColor(colors().textAlt);
        ctx.drawText(Fonts.Style.Normal, hoveredCategory,
            mouseXpos + HOVER_MARGIN + HOVER_PADDING,
            mouseYpos + hoveredSize.h / 2, hoveredSize.h / 2);
      }
    }
  }

  private static String buildSliceTitle(String title, ArgSet args) {
    Object w = args.get("width"), h = args.get("height");
    return (w == null || h == null) ? title : title + " (" + w + "x" + h + ")";
  }

  @Override
  protected Hover onTrackMouseMove(Fonts.TextMeasurer m, double x, double y, int mods) {
    FrameEventsTrack.Data data = track.getData(state.toRequest(), onUiThread());
    if (data == null) {
      return Hover.NONE;
    }

    switch (data.kind) {
      case slices: return sliceHover(data, m, x, y, mods);
      case summary: return summaryHover(data, m, x, mods);
      default: return Hover.NONE;
    }
  }

  private Hover summaryHover(FrameEventsTrack.Data data, Fonts.TextMeasurer m, double x, int mods) {
    long time = state.pxToTime(x);
    int bucket = (int)((time - data.request.range.start) / data.bucketSize);
    if (bucket < 0 || bucket >= data.numEvents.length) {
      return Hover.NONE;
    }

    long p = data.numEvents[bucket];
    String text;
    if (p == 0) {
      text = "Nothing presented";
    } else if (p == 1) {
      text = Long.toString(p) + " frame presented";
    } else {
      text = Long.toString(p) + " frames presented";
    }
    hovered = new HoverCard(bucket, p, text, m.measure(Fonts.Style.Normal, text));

    double mouseX = state.timeToPx(
        data.request.range.start + hovered.bucket * data.bucketSize + data.bucketSize / 2);
    double dx = HOVER_PADDING + hovered.size.w + HOVER_PADDING;
    double dy = height;
    String ids = Arrays.getOrDefault(data.concatedIds, bucket, "");

    return new Hover() {
      @Override
      public Area getRedraw() {
        return new Area(mouseX - CURSOR_SIZE, -TRACK_MARGIN, CURSOR_SIZE + HOVER_MARGIN + dx, dy);
      }

      @Override
      public void stop() {
        hovered = null;
      }

      @Override
      public Cursor getCursor(Display display) {
        return p == 0 ? null : display.getSystemCursor(SWT.CURSOR_HAND);
      }

      @Override
      public boolean click() {
        if (ids.isEmpty()) {
          return false;
        }
        if ((mods & SWT.MOD1) == SWT.MOD1) {
          state.addSelection(Selection.Kind.FrameEvents, track.getSlices(ids));
        } else {
          state.setSelection(Selection.Kind.FrameEvents, track.getSlices(ids));
        }
        return true;
      }
    };
  }

  private Hover sliceHover(FrameEventsTrack.Data data, Fonts.TextMeasurer m, double x, double y, int mods) {
    int depth = (int)(y / SLICE_HEIGHT);
    if (depth < 0 || depth > buffer.maxDepth) {
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
      if (x >= ts && x<= endts) {
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
            (1 + buffer.maxDepth) * SLICE_HEIGHT - hoveredSize.h));
        long id = data.ids[i];
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
    if (startDepth > endDepth) {
      return;
    }

    if (endDepth >= 0) {
      if (endDepth >= buffer.maxDepth) {
        endDepth = Integer.MAX_VALUE;
      }
      builder.add(Selection.Kind.FrameEvents, track.getSlices(ts, startDepth, endDepth));
    }
  }

  private static FrameSelection getSelectedFrames(State state) {
    Selection<?> selection = state.getSelection(Selection.Kind.FrameEvents);
    if (selection instanceof FrameEventsTrack.Slices) {
      return ((FrameEventsTrack.Slices) selection).getSelection();
    }
    return FrameSelection.EMPTY;
  }

  private static class HoverCard {
    public final int bucket;
    public final long numEvents;
    public final String text;
    public final Size size;

    public HoverCard(int bucket, long numEvents, String text, Size size) {
      this.bucket = bucket;
      this.numEvents = numEvents;
      this.text = text;
      this.size = size;
    }
  }

  @Override
  public FrameEventsSummaryPanel copy() {
    return new FrameEventsSummaryPanel(state, buffer, track);
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
