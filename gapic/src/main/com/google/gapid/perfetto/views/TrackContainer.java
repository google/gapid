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

import static com.google.gapid.perfetto.views.StyleConstants.LABEL_ICON_SIZE;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_MARGIN;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_OFFSET;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_PIN_X;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_TOGGLE_X;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.TITLE_HEIGHT;
import static com.google.gapid.perfetto.views.StyleConstants.arrowDown;
import static com.google.gapid.perfetto.views.StyleConstants.arrowRight;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.StyleConstants.pinActive;
import static com.google.gapid.perfetto.views.StyleConstants.pinInactive;
import static com.google.gapid.perfetto.views.StyleConstants.unfoldLess;
import static com.google.gapid.perfetto.views.StyleConstants.unfoldMore;

import com.google.common.base.Supplier;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.TrackConfig;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.widgets.Display;

import java.util.function.BiConsumer;

// TODO: dedupe some of the below code.
/**
 * Containers of {@link TrackPanel TrackPanels}.
 */
public class TrackContainer {
  private TrackContainer() {
  }

  public static <T extends TrackPanel<T>> TrackConfig.Track.UiFactory<Panel> single(
      TrackConfig.Track.UiFactory<T> track, boolean sep) {
    return state -> new Single<T>(state, track.createPanel(state), sep, null, true);
  }

  public static <T extends TrackPanel<T>> TrackConfig.Track.UiFactory<Panel> single(
      TrackConfig.Track.UiFactory<T> track, boolean sep, BiConsumer<T, Boolean> filter,
      boolean initial) {
    return state -> {
      T panel = track.createPanel(state);
      if (initial) {
        filter.accept(panel, initial);
      }
      return new Single<T>(state, panel, sep, filter, initial);
    };
  }

  public static <T extends TitledPanel & CopyablePanel<T>> TrackConfig.Group.UiFactory group(
      TrackConfig.Track.UiFactory<T> summary, boolean expanded) {
    return (state, detail) -> {
      CopyablePanel.Group group = new CopyablePanel.Group();
      for (CopyablePanel<?> track : detail) {
        group.add(track);
      }
      return Group.of(state, summary.createPanel(state), group, expanded, null, false);
    };
  }

  public static <T extends TitledPanel & CopyablePanel<T>> TrackConfig.Group.UiFactory group(
      TrackConfig.Track.UiFactory<T> summary, boolean expanded,
      BiConsumer<CopyablePanel.Group, Boolean> filter, boolean initial) {
    return (state, detail) -> {
      CopyablePanel.Group group = new CopyablePanel.Group();
      for (CopyablePanel<?> track : detail) {
        group.add(track);
      }
      if (initial) {
        filter.accept(group, true);
      }
      return Group.of(state, summary.createPanel(state), group, expanded, filter, initial);
    };
  }

  private static class Single<T extends TrackPanel<T>> extends Panel.Base
      implements CopyablePanel<Single<T>> {
    private final T track;
    private final boolean sep;
    protected final BiConsumer<T, Boolean> filter;
    private final PinState pinState;

    protected boolean filtered;
    protected boolean hovered = false;

    public Single(State.ForSystemTrace state, T track, boolean sep, BiConsumer<T, Boolean> filter,
        boolean filtered) {
      this(track ,sep, filter, filtered, new PinState(state));
    }

    private Single(T track, boolean sep, BiConsumer<T, Boolean> filter,
        boolean filtered, PinState pinState) {
      this.track = track;
      this.sep = sep;
      this.filter = filter;
      this.pinState = pinState;
      this.filtered = filtered;
    }

    @Override
    public Single<T> copy() {
      return new Single<T>(track.copy(), sep, filter, filtered, pinState);
    }

    private Single<T> copyWithSeparator() {
      return new Single<T>(track.copy(), true, filter, filtered, pinState);
    }

    @Override
    public double getPreferredHeight() {
      return track.getPreferredHeight();
    }

    @Override
    public void setSize(double w, double h) {
      super.setSize(w, h);
      track.setSize(w, h);
    }

    @Override
    public void render(RenderContext ctx, Repainter repainter) {
      ctx.withClip(0, 0, LABEL_WIDTH, height, () -> {
        ctx.setForegroundColor(colors().textMain);
        ctx.drawTextLeftTruncate(Fonts.Style.Normal, track.getTitle(), LABEL_OFFSET, 0,
            ((filter == null) ? LABEL_PIN_X  : LABEL_TOGGLE_X) - LABEL_MARGIN - LABEL_OFFSET,
            TITLE_HEIGHT);
        if (filter != null) {
          ctx.drawIcon(filtered ? unfoldMore(ctx.theme) : unfoldLess(ctx.theme),
              LABEL_TOGGLE_X, 0, TITLE_HEIGHT);
        }
        if (hovered || pinState.isPinned()) {
          ctx.drawIcon(pinState.icon(ctx), LABEL_PIN_X, 0, TITLE_HEIGHT);
        }
      });

      ctx.setForegroundColor(colors().panelBorder);
      ctx.drawLine(LABEL_WIDTH - 1, 0, LABEL_WIDTH - 1, height);
      ctx.drawLine(sep ? 0 : LABEL_WIDTH, height - 1, width, height - 1);
      track.render(ctx, repainter);
    }

    @Override
    public void visit(Visitor v, Area area) {
      super.visit(v, area);
      track.visit(v, area);
    }

    @Override
    public Dragger onDragStart(double x, double y, int mods) {
      return track.onDragStart(x, y, mods);
    }

    @Override
    public Hover onMouseMove(Fonts.TextMeasurer m, double x, double y) {
      if (x < LABEL_WIDTH) {
        hovered = true;
        if (filter != null && y < TITLE_HEIGHT && x >= LABEL_TOGGLE_X && x < LABEL_PIN_X) {
          return new TrackTitleHover(track.onMouseMove(m, x, y), () -> {
            filtered = !filtered;
            filter.accept(track, filtered);
          });
        } else if (y < TITLE_HEIGHT && x >= LABEL_PIN_X) {
          return new TrackTitleHover(
              track.onMouseMove(m, x, y), () -> pinState.toggle(this::copyWithSeparator));
        } else {
          return new TrackTitleHover(track.onMouseMove(m, x, y), null);
        }
      } else {
        return track.onMouseMove(m, x, y);
      }
    }

    private class TrackTitleHover extends TrackContainer.TrackTitleHover {
      public TrackTitleHover(Panel.Hover child, Runnable click) {
        super(child, click);
      }

      @Override
      public void stop() {
        super.stop();
        hovered = false;
      }
    }
  }

  private static class Group<T extends CopyablePanel<T> & TitledPanel, D extends CopyablePanel<D>>
      extends Panel.Base implements CopyablePanel<Group<T, D>> {
    private final T summary;
    private final CopyablePanel.Group detail;
    protected final BiConsumer<CopyablePanel.Group, Boolean> filter;
    private final PinState pinState;

    protected boolean expanded;
    protected boolean filtered;
    protected boolean hovered = false;

    private Group(T summary, CopyablePanel.Group detail, boolean expanded,
        BiConsumer<CopyablePanel.Group, Boolean> filter, boolean filtered, PinState pinState) {
      this.summary = summary;
      this.detail = detail;
      this.expanded = expanded;
      this.filter = filter;
      this.filtered = filtered;
      this.pinState = pinState;
    }

    public static <T extends CopyablePanel<T> & TitledPanel, D extends CopyablePanel<D>>
        TrackContainer.Group<T, D> of(State.ForSystemTrace state, T summary,
            CopyablePanel.Group detail, boolean expanded,
            BiConsumer<CopyablePanel.Group, Boolean> filter, boolean filtered) {
      return new TrackContainer.Group<T, D>(
          summary, detail, expanded, filter, filtered, new PinState(state));
    }

    @Override
    public TrackContainer.Group<T, D> copy() {
      return new TrackContainer.Group<T, D>(
          summary.copy(), detail.copy(), expanded, filter, filtered, pinState);
    }

    @Override
    public double getPreferredHeight() {
      return expanded ? TITLE_HEIGHT + detail.getPreferredHeight() : summary.getPreferredHeight();
    }

    @Override
    public void setSize(double w, double h) {
      super.setSize(w, h);
      if (expanded) {
        detail.setSize(w, h - TITLE_HEIGHT);
      } else {
        summary.setSize(w, h);
      }
    }

    @Override
    public void render(RenderContext ctx, Repainter repainter) {
      if (expanded) {
        ctx.setBackgroundColor(colors().titleBackground);
        ctx.fillRect(0, 0, width, TITLE_HEIGHT);

        ctx.setForegroundColor(colors().textMain);
        ctx.drawIcon(arrowDown(ctx.theme), 0, 0, TITLE_HEIGHT);
        ctx.drawText(Fonts.Style.Normal, summary.getTitle(), LABEL_OFFSET, 0, TITLE_HEIGHT);

        double x = Math.max(LABEL_TOGGLE_X, LABEL_OFFSET +
            Math.ceil(ctx.measure(Fonts.Style.Normal, summary.getTitle()).w) + LABEL_MARGIN);
        if (filter != null) {
          ctx.drawIcon(
              filtered ? unfoldMore(ctx.theme) : unfoldLess(ctx.theme), x, 0, TITLE_HEIGHT);
          x += LABEL_ICON_SIZE;
        }
        if (hovered || pinState.isPinned()) {
          ctx.drawIcon(pinState.icon(ctx), Math.max(x, LABEL_PIN_X), 0, TITLE_HEIGHT);
        }

        ctx.setForegroundColor(colors().panelBorder);
        ctx.drawLine(0, height - 1, width - 1 , height - 1);
        summary.decorateTitle(ctx, repainter);

        ctx.withTranslation(0, TITLE_HEIGHT, () ->
          detail.render(ctx, repainter.translated(0, TITLE_HEIGHT)));
      } else {
        ctx.withClip(0, 0, LABEL_WIDTH, height, () -> {
          ctx.setForegroundColor(colors().textMain);
          ctx.drawIcon(arrowRight(ctx.theme), 0, 0, TITLE_HEIGHT);
          ctx.drawTextLeftTruncate(Fonts.Style.Normal, summary.getTitle(),
              LABEL_OFFSET, 0, LABEL_PIN_X - LABEL_MARGIN - LABEL_OFFSET, TITLE_HEIGHT);
          if (hovered || pinState.isPinned()) {
            ctx.drawIcon(pinState.icon(ctx), LABEL_PIN_X, 0, TITLE_HEIGHT);
          }
          if (!summary.getSubTitle().isEmpty()) {
            ctx.setForegroundColor(colors().textAlt);
            ctx.drawText(Fonts.Style.Normal, summary.getSubTitle(), LABEL_OFFSET, TITLE_HEIGHT);
          }
        });

        ctx.setForegroundColor(colors().panelBorder);
        ctx.drawLine(LABEL_WIDTH - 1, 0, LABEL_WIDTH - 1, height - 1);
        ctx.drawLine(0, height - 1, width , height - 1);
        summary.render(ctx, repainter);
      }
    }

    @Override
    public void visit(Visitor v, Area area) {
      super.visit(v, area);
      if (expanded) {
        area.intersect(0, TITLE_HEIGHT, width, height - TITLE_HEIGHT)
            .ifNotEmpty(a -> detail.visit(v, area.translate(0, -TITLE_HEIGHT)));
      } else {
        summary.visit(v, area);
      }
    }

    @Override
    public Dragger onDragStart(double x, double y, int mods) {
      if (expanded) {
        return (y < TITLE_HEIGHT) ? Dragger.NONE :
          detail.onDragStart(x, y - TITLE_HEIGHT, mods).translated(0, TITLE_HEIGHT);
      } else {
        return summary.onDragStart(x, y, mods);
      }
    }

    @Override
    public Hover onMouseMove(Fonts.TextMeasurer m, double x, double y) {
      if (y < TITLE_HEIGHT && (expanded || x < LABEL_WIDTH)) {
        hovered = true;
        double textEnd =
            Math.ceil(m.measure(Fonts.Style.Normal, summary.getTitle()).w) + LABEL_OFFSET;
        double gapEnd = expanded ? Math.max(textEnd, LABEL_TOGGLE_X) : LABEL_TOGGLE_X;
        double toggleEnd = (expanded && filter != null) ? gapEnd + LABEL_ICON_SIZE : gapEnd;
        double pinEnd = Math.max(LABEL_PIN_X, toggleEnd) + LABEL_ICON_SIZE;
        double redraw = (pinEnd > LABEL_WIDTH) ? pinEnd + LABEL_MARGIN : 0;
        if (expanded) {
          if (x < textEnd) {
            return new TrackTitleHover(Hover.NONE, redraw, () -> expanded = false);
          }
        } else {
          if (x < Math.min(textEnd, LABEL_PIN_X - LABEL_MARGIN)) {
            return new TrackTitleHover(summary.onMouseMove(m, x, y), redraw, () -> expanded = true);
          }
          toggleEnd = LABEL_PIN_X;
          pinEnd = LABEL_WIDTH;
        }
        if (expanded && filter != null && x >= gapEnd && x < toggleEnd) {
          return new TrackTitleHover(Hover.NONE, redraw, () -> {
            filtered = !filtered;
            filter.accept(detail, filtered);
          });
        }
        if (x >= toggleEnd && x < pinEnd) {
          return new TrackTitleHover(Hover.NONE, redraw, () -> pinState.toggle(this::copy));
        }
        return new TrackTitleHover(Hover.NONE, redraw, null);
      } else if (!expanded && x < LABEL_WIDTH) {
        hovered = true;
        return new TrackTitleHover(Hover.NONE, 0, null);
      }

      if (expanded) {
        return detail.onMouseMove(m, x, y - TITLE_HEIGHT).translated(0, TITLE_HEIGHT);
      } else {
        return summary.onMouseMove(m, x, y);
      }
    }

    private class TrackTitleHover extends TrackContainer.TrackTitleHover {
      private double redraw;

      public TrackTitleHover(Hover child, double width, Runnable click) {
        super(child, click);
        this.redraw = width;
      }

      @Override
      public Area getRedraw() {
        return super.getRedraw().combine(new Area(0, 0, redraw, TITLE_HEIGHT));
      }

      @Override
      public void stop() {
        super.stop();
        hovered = false;
      }
    }
  }

  private static class PinState {
    private final State.ForSystemTrace state;
    private Panel pinned;

    public PinState(State.ForSystemTrace state) {
      this.state = state;
      pinned = null;
    }

    public boolean isPinned() {
      return pinned != null;
    }

    public Image icon(RenderContext ctx) {
      return isPinned() ? pinActive(ctx.theme) : pinInactive(ctx.theme);
    }

    public void toggle(Supplier<Panel> copy) {
      PinnedTracks tracks = state.getPinnedTracks();
      double current =  tracks.getPreferredHeight();
      if (isPinned()) {
        tracks.unpin(pinned);
        pinned = null;
      } else {
        pinned = copy.get();
        tracks.pin(pinned);
      }
      state.dragY(current -  tracks.getPreferredHeight());
    }
  }

  private static class TrackTitleHover implements Panel.Hover {
    private final Panel.Hover child;
    private final Runnable click;

    public TrackTitleHover(Panel.Hover child, Runnable click) {
      this.child = child;
      this.click = click;
    }

    @Override
    public Area getRedraw() {
      return child.getRedraw().combine(new Area(0, 0, LABEL_WIDTH, TITLE_HEIGHT));
    }

    @Override
    public void stop() {
      child.stop();
    }

    @Override
    public boolean click() {
      boolean r = child.click();
      if (click != null) {
        click.run();
        return true;
      }
      return r;
    }

    @Override
    public Cursor getCursor(Display d) {
      return (click == null) ? child.getCursor(d) : d.getSystemCursor(SWT.CURSOR_HAND);
    }
  }
}
