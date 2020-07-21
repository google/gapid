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
import static com.google.gapid.perfetto.views.StyleConstants.HIGHLIGHT_EDGE_NEARBY_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.StyleConstants.flag;
import static com.google.gapid.perfetto.views.StyleConstants.flagFilled;
import static com.google.gapid.perfetto.views.StyleConstants.flagGreyed;
import static com.google.gapid.widgets.Widgets.createToggleToolItem;
import static com.google.gapid.widgets.Widgets.exclusiveSelection;
import static java.util.Arrays.stream;

import com.google.common.collect.Maps;
import com.google.gapid.models.Settings;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.PanelGroup;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.TrackConfig;
import com.google.gapid.perfetto.models.VSync;
import com.google.gapid.widgets.Theme;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.util.SortedMap;
import java.util.TreeMap;
import java.util.function.Consumer;
import java.util.function.Function;
import java.util.function.IntConsumer;

/**
 * The main {@link Panel} containing all track panels. Shows a {@link TimelinePanel} at the top,
 * and tracks below.
 */
public abstract class RootPanel<S extends State> extends Panel.Base implements State.Listener {
  private static final double HIGHLIGHT_TOP = 22;
  private static final double HIGHLIGHT_BOTTOM = 30;
  private static final double HIGHLIGHT_CENTER = (HIGHLIGHT_TOP + HIGHLIGHT_BOTTOM) / 2;
  private static final double HIGHLIGHT_PADDING = 3;
  private static final double FLAG_WIDTH = 17;  // Width of the actual flag, not the image's pixel width
  public static final double FLAGS_Y = 30;

  protected final Settings settings;
  protected final TimelinePanel timeline;
  protected final PanelGroup top = new PanelGroup();
  protected final PanelGroup bottom = new PanelGroup();
  protected final S state;

  private MouseMode mouseMode = MouseMode.Pan;
  private boolean panOverride = false;
  protected boolean showVSync;
  private Area selection = Area.NONE;
  private boolean isHighlightStartHovered = false;
  private boolean isHighlightEndHovered = false;

  protected TreeMap<Long, Boolean> flags;
  protected boolean flagHovered = false;
  protected double flagHoverXpos;

  public RootPanel(S state, Settings settings) {
    this.settings = settings;
    this.timeline = new TimelinePanel(state);
    this.state = state;
    this.showVSync = settings.ui().getPerfetto().getShowVsync();
    this.flags = Maps.newTreeMap();
    state.addListener(this);
  }

  public void clear() {
    top.clear();
    bottom.clear();
  }

  @Override
  public void onDataChanged() {
    clear();
    createUi();
  }

  protected abstract void createUi();

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

  private SortedMap<Long, Boolean> searchForFlag(double x) {
    long time = state.pxToTime(x);
    // If the click falls within the flag icon's boundaries, the flag is considered to be hit.
    // Leave an additional two pixels to the left of the flag to get a better hit box
    long rightOffset = state.deltaPxToDuration(FLAG_WIDTH);
    long leftOffset = state.deltaPxToDuration(2);

    return flags.subMap(time - rightOffset, time + leftOffset);
  }

  protected void searchAndRemoveFlag(double x) {
    SortedMap<Long, Boolean> subMap = searchForFlag(x);
    if (!subMap.isEmpty()) {
      subMap.clear();
    }
  }

  protected void searchAndAddFlag(double x) {
    SortedMap<Long, Boolean> subMap = searchForFlag(x);
    if (subMap.isEmpty()) {
      flags.put(state.pxToTime(x), true);
    } else {
      toggleFlag(subMap);
    }
  }

  private static void toggleFlag(SortedMap<Long, Boolean> subMap) {
    subMap.replaceAll((k,v) -> v = !v);
  }

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    double topHeight = top.getPreferredHeight();
    Area clip = ctx.getClip();
    if (clip.y < topHeight) {
      preTopUiRender(ctx, repainter);
      top.render(ctx, repainter);
    }
    if (clip.y + clip.h > topHeight) {
      double newClipY = Math.max(clip.y, topHeight);
      ctx.withClip(clip.x, newClipY, clip.w, clip.h - (newClipY - clip.y), () -> {
        ctx.withTranslation(0, topHeight - state.getScrollOffset(), () -> {
          preMainUiRender(ctx, repainter);
          bottom.render(ctx, repainter.transformed(
              a -> a.translate(0, topHeight - state.getScrollOffset())));
        });
      });
    }

    postMainUiRender(ctx);

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

        // Draw drag-able hint on highlighted timeline's edge when hovered.
        ctx.setForegroundColor(colors().timeHighlightEmphasize);
        if (isHighlightStartHovered) {
          ctx.drawLine(x1, 0, x1, height, 3);
        } else if (isHighlightEndHovered) {
          ctx.drawLine(x2, 0, x2, height, 3);
        }
      });
    }

    if (!selection.isEmpty()) {
      ctx.setBackgroundColor(colors().selectionBackground);
      ctx.fillRect(selection.x, selection.y, selection.w, selection.h);
    }
  }

  protected abstract void preTopUiRender(RenderContext ctx, Repainter repainter);
  protected abstract void preMainUiRender(RenderContext ctx, Repainter repainter);
  protected abstract void postMainUiRender(RenderContext ctx);

  @Override
  public void visit(Visitor v, Area area) {
    double topHeight = top.getPreferredHeight();
    area.intersect(0, 0, width, topHeight).ifNotEmpty(a -> top.visit(v, a));
    area.intersect(0, topHeight, width, height - topHeight)
        .ifNotEmpty(a -> bottom.visit(v, a.translate(0, -topHeight + state.getScrollOffset())));
  }

  @Override
  public Dragger onDragStart(double sx, double sy, int mods) {
    if (sx < LABEL_WIDTH || (mods & SWT.BUTTON1) != SWT.BUTTON1) {
      return Dragger.NONE;
    }

    if (sy <= timeline.getPreferredHeight() || isHighlightStartHovered || isHighlightEndHovered) {
      return timeSelectDragger(sx);
    }

    MouseMode mode = mouseMode;
    if ((mods & SWT.SHIFT) == SWT.SHIFT) {
      mode = MouseMode.Select;
    } else if ((mods & SWT.MOD1) == SWT.MOD1 && mouseMode != MouseMode.Select) {
      mode = MouseMode.TimeSelect;
    } else if (panOverride) {
      mode = MouseMode.Pan;
    }

    switch (mode) {
      case Select: return selectDragger(sx, sy, mods);
      case Pan: return panDragger(sx, sy, mods);
      case Zoom: return zoomDragger(sx, sy);
      case TimeSelect: return timeSelectDragger(sx);
      default: return Dragger.NONE;
    }
  }

  private Dragger selectDragger(double sx, double sy, int mods) {
    return new Dragger() {
      @Override
      public Area onDrag(double x, double y) {
        return updateSelection(sx, sy, x, y);
      }

      @Override
      public Area onDragEnd(double x, double y) {
        Area redraw = updateSelection(sx, sy, x, y);
        finishSelection(mods);
        return redraw;
      }
    };
  }

  private Dragger panDragger(double sx, double sy, int mods) {
    double topHeight = top.getPreferredHeight();
    Dragger childDragger = (sy < topHeight) ? top.onDragStart(sx, sy, mods) :
        bottom.onDragStart(sx, sy - topHeight + state.getScrollOffset(), mods)
            .translated(0, topHeight - state.getScrollOffset());

    State st = state;
    return childDragger.or(new Dragger() {
      private final TimeSpan atStart = st.getVisibleTime();
      private double lastY = sy;

      @Override
      public Area onDrag(double x, double y) {
        Area areaX = st.dragX(atStart, x - sx) ? Area.FULL : Area.NONE;
        Area areaY = st.dragY(y - lastY) ? Area.FULL : Area.NONE;
        lastY = y;
        return areaX.combine(areaY);
      }

      @Override
      public Area onDragEnd(double x, double y) {
        return onDrag(x, y);
      }

      @Override
      public Cursor getCursor(Display display) {
        return display.getSystemCursor(SWT.CURSOR_SIZEWE);
      }
    });
  }

  private Dragger zoomDragger(double sx, double sy) {
    return new Dragger() {
      double lastY = sy;

      @Override
      public Area onDrag(double x, double y) {
        double mag = 1 - StyleConstants.ZOOM_FACTOR_SCALE_DRAG * (lastY - y);
        lastY = y;
        return zoom(sx, mag) ? Area.FULL : Area.NONE;
      }

      @Override
      public Area onDragEnd(double x, double y) {
        return onDrag(x, y);
      }
    };
  }

  private Dragger timeSelectDragger(double sx) {
    double hFixedEnd = findHighlightFixedEnd(sx);
    return new Dragger() {
      @Override
      public Area onDrag(double x, double y) {
        return updateHighlight(hFixedEnd, x);
      }

      @Override
      public Area onDragEnd(double x, double y) {
        return onDrag(x, y);
      }
    };
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

  protected void finishSelection(int mods) {
    Area onTrack = selection.intersect(LABEL_WIDTH, 0, width - LABEL_WIDTH, height)
        .translate(-LABEL_WIDTH, 0);
    TimeSpan ts = new TimeSpan(
        state.pxToTime(onTrack.x), state.pxToTime(onTrack.x + onTrack.w));

    if ((mods & SWT.MOD1) != SWT.MOD1) {
      state.clearSelectedThreads();
    }
    Selection.CombiningBuilder builder = new Selection.CombiningBuilder();
    visit(Visitor.of(Selectable.class, (s, a) -> s.computeSelection(builder, a, ts)), selection);
    selection = Area.NONE;
    if ((mods & SWT.MOD1) == SWT.MOD1) {
      state.addSelection(builder.build());
    } else {
      state.setSelection(builder.build());
    }
  }

  @Override
  public Hover onMouseMove(Fonts.TextMeasurer m, double x, double y, int mods) {
    if (y >= (timeline.getPreferredHeight()/2) && y <= timeline.getPreferredHeight() && x > LABEL_WIDTH) {
      return flagHover(x);
    }
    double topHeight = top.getPreferredHeight();
    Hover result = (y < topHeight) ? top.onMouseMove(m, x, y, mods) :
      bottom.onMouseMove(m, x, y - topHeight + state.getScrollOffset(), mods)
          .transformed(a -> a.translate(0, topHeight - state.getScrollOffset()));
    if (x >= LABEL_WIDTH && y >= topHeight && result == Hover.NONE) {
      result = result.withClick(() -> state.resetSelections());
    }
    if (x >= LABEL_WIDTH) {
      result = result.withClick(() -> {
        TimeSpan highlight = state.getHighlight();
        if (!highlight.isEmpty() && !highlight.contains(state.pxToTime(x - LABEL_WIDTH))) {
          state.setHighlight(TimeSpan.ZERO);
          return true;
        }
        return false;
      });
    }
    if (checkHighlightEdgeHovered(x)) {
      result = result.withRedraw(Area.FULL);
    }
    return result;
  }

  private Hover flagHover(double x) {
    if (searchForFlag(x - LABEL_WIDTH).isEmpty()) {
      flagHovered = true;
      flagHoverXpos = x;
    } else {
      flagHovered = false;
    }
    return new Panel.Hover() {
      @Override
      public Area getRedraw() {
        TimeSpan visible = state.getVisibleTime();
        // Redraw the entire visible range
        return new Area(
            state.timeToPx(visible.start), 0, bottom.getPreferredHeight(), state.timeToPx(visible.end));
      }

      @Override
      public boolean click() {
        searchAndAddFlag(x - LABEL_WIDTH);
        return true;
      }

      @Override
      public boolean rightClick() {
        searchAndRemoveFlag(x - LABEL_WIDTH);
        return true;
      }

      @Override
      public void stop() {
        flagHovered = false;
      }
    };
  }

  private double findHighlightFixedEnd(double sx) {
    double hStart = state.timeToPx(state.getHighlight().start) + LABEL_WIDTH;
    double hEnd = state.timeToPx(state.getHighlight().end) + LABEL_WIDTH;
    boolean nearStart = Math.abs(sx - hStart) <= HIGHLIGHT_EDGE_NEARBY_WIDTH;
    boolean nearEnd = Math.abs(sx - hEnd) <= HIGHLIGHT_EDGE_NEARBY_WIDTH;
    if (nearStart && nearEnd) {
      return Math.abs(sx - hStart) < Math.abs(sx - hEnd) ? hEnd : hStart;
    } else if (nearStart) {
      return hEnd;
    } else if (nearEnd) {
      return hStart;
    } else {
      return sx;
    }
  }

  // Return true if the highlight edge's hovering status changes.
  private boolean checkHighlightEdgeHovered(double x) {
    boolean preStartStatus = isHighlightStartHovered;
    boolean preEndStatus = isHighlightEndHovered;
    double hStart = state.timeToPx(state.getHighlight().start) + LABEL_WIDTH;
    double hEnd = state.timeToPx(state.getHighlight().end) + LABEL_WIDTH;
    boolean nearStart = Math.abs(x - hStart) <= HIGHLIGHT_EDGE_NEARBY_WIDTH;
    boolean nearEnd = Math.abs(x - hEnd) <= HIGHLIGHT_EDGE_NEARBY_WIDTH;
    boolean closerToStart = Math.abs(x - hStart) < Math.abs(x - hEnd);
    isHighlightStartHovered = nearStart && closerToStart;
    isHighlightEndHovered = nearEnd && !closerToStart;
    return preStartStatus != isHighlightStartHovered || preEndStatus != isHighlightEndHovered;
  }

  public void setMouseMode(MouseMode mode) {
    this.mouseMode = mode;
  }

  public void setPanOverride(boolean panOverride) {
    this.panOverride = panOverride;
  }

  public void toggleVSync() {
    this.showVSync = !showVSync;
    settings.writeUi().getPerfettoBuilder().setShowVsync(showVSync);
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

  public void updateFilter(String search) {
    bottom.updateFilter(panel -> {
      if (panel instanceof FilterablePanel) {
        return ((FilterablePanel)panel).include(search) ?
            Panel.FilterValue.Include : Panel.FilterValue.DescendOrExclude;
      }
      // Panels that are not filterable, should be shown if they are not groups.
      return Panel.FilterValue.DescendOrInclude;
    });
  }

  public static class ForSystemTrace extends RootPanel<State.ForSystemTrace> {

    public ForSystemTrace(State.ForSystemTrace state, Settings settings) {
      super(state, settings);
    }

    @Override
    protected void createUi() {
      if (state.hasData()) {
        top.add(timeline);
        top.add(state.getPinnedTracks());
        for (TrackConfig.Element<?> el : state.getTracks().elements) {
          bottom.add(el.createUi(state));
        }
      }
    }

    @Override
    protected void preTopUiRender(RenderContext ctx, Repainter repainter) {
      if (showVSync && state.hasData() && state.getVSync().hasData()) {
        renderVSync(ctx, repainter, top, state.getVSync());
      }
    }

    @Override
    protected void preMainUiRender(RenderContext ctx, Repainter repainter) {
      if (showVSync && state.hasData() && state.getVSync().hasData()) {
        renderVSync(ctx, repainter, bottom, state.getVSync());
      }
    }

    @Override
    protected void postMainUiRender(RenderContext ctx) {
      // Render the Flag in the timeline panel and the vertical line in the bottom panel group
      renderFlags(ctx, bottom);
    }

    private void renderFlags(RenderContext ctx, Panel panel) {
      flags.forEach((k,v) -> {
        double x = Math.rint(LABEL_WIDTH + state.timeToPx(k));
        if (x > LABEL_WIDTH) {
          if (v) {
            ctx.drawIcon(flagFilled(ctx.theme), x - 5, FLAGS_Y, 0);
            ctx.setForegroundColor(colors().flagLine);
            ctx.drawLine(x, FLAGS_Y, x, panel.getPreferredHeight());
          } else {
            ctx.drawIcon(flag(ctx.theme), x - 5, FLAGS_Y, 0);
          }
        }
      });
      if (flagHovered) {
        double x = flagHoverXpos;
        ctx.drawIcon(flagGreyed(ctx.theme), x - 5, FLAGS_Y, 0);
        ctx.setForegroundColor(colors().flagHover);
        ctx.drawLine(x, FLAGS_Y, x, panel.getPreferredHeight());
      }
    }

    private void renderVSync(RenderContext ctx, Repainter repainter, Panel panel, VSync vsync) {
      ctx.trace("VSync", () -> {
        VSync.Data data = vsync.getData(state.toRequest(),
            TrackPanel.onUiThread(state, () -> repainter.repaint(new Area(0, 0, width, height))));
        if (data == null) {
          return;
        }

        TimeSpan visible = state.getVisibleTime();
        ctx.setBackgroundColor(colors().vsyncBackground);
        boolean fill = !data.fillFirst;
        double lastX = LABEL_WIDTH;
        double h = panel.getPreferredHeight();
        for (long time : data.ts) {
          fill = !fill;
          if (time < visible.start) {
            continue;
          }
          double x = LABEL_WIDTH + state.timeToPx(time);
          if (fill) {
            ctx.fillRect(lastX, 0, x - lastX, h);
          }
          lastX = x;
          if (time > visible.end) {
            break;
          }
        }
      });
    }
  }

  public static enum MouseMode {
    Select("Selection (1)", "Selection Mode (1): Drag to select items.", Theme::selectionMode),
    Pan("Pan (2)", "Pan Mode (2): Drag to pan the view.", Theme::panMode),
    Zoom("Zoom (3)", "Zoom Mode (3): Drag to zoom the view.", Theme::zoomMode),
    TimeSelect("Timing (4)", "Timing Mode (4): Drag to select a time range.", Theme::timingMode);

    private final String label;
    private final String toolTip;
    private final Function<Theme, Image> icon;

    private MouseMode(String label, String toolTip, Function<Theme, Image> icon) {
      this.label = label;
      this.toolTip = toolTip;
      this.icon = icon;
    }

    private ToolItem createItem(ToolBar bar, Theme theme, Consumer<MouseMode> onClick) {
      ToolItem item = createToggleToolItem(
          bar, icon.apply(theme), e -> onClick.accept(this), toolTip);
      item.setText(label);
      return item;
    }

    public static Consumer<MouseMode> createToolBar(
        ToolBar bar, Theme theme, Consumer<MouseMode> onClick) {
      IntConsumer itemSelector = exclusiveSelection(stream(values())
          .map(m -> m.createItem(bar, theme, onClick))
          .toArray(ToolItem[]::new));

      return mode -> {
        onClick.accept(mode);
        itemSelector.accept(mode.ordinal());
      };
    }
  }
}
