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

import static com.google.gapid.perfetto.views.StyleConstants.LABEL_OFFSET;
import static com.google.gapid.perfetto.views.StyleConstants.LABEL_WIDTH;
import static com.google.gapid.perfetto.views.StyleConstants.TITLE_HEIGHT;
import static com.google.gapid.perfetto.views.StyleConstants.TOGGLE_ICON_OFFSET;
import static com.google.gapid.perfetto.views.StyleConstants.arrowDown;
import static com.google.gapid.perfetto.views.StyleConstants.arrowRight;
import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.perfetto.views.StyleConstants.unfoldLess;
import static com.google.gapid.perfetto.views.StyleConstants.unfoldMore;

import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.PanelGroup;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.TrackConfig;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

import java.util.function.BiConsumer;
import java.util.function.Consumer;

// TODO: dedupe some of the below code.
/**
 * Containers of {@link TrackPanel TrackPanels}.
 */
public class TrackContainer {
  private TrackContainer() {
  }

  public static TrackConfig.Track.UiFactory<Panel> single(
      TrackConfig.Track.UiFactory<TrackPanel> track, boolean sep) {
    return state -> new Single(track.createPanel(state), sep, null, true);
  }

  public static <T extends TrackPanel> TrackConfig.Track.UiFactory<Panel> single(
      TrackConfig.Track.UiFactory<T> track, boolean sep, BiConsumer<T, Boolean> filter,
      boolean initial) {
    return state -> {
      T panel = track.createPanel(state);
      if (initial) {
        filter.accept(panel, initial);
      }
      return new Single(panel, sep, filtered -> filter.accept(panel, filtered), initial);
    };
  }

  public static TrackConfig.Group.UiFactory group(
      TrackConfig.Track.UiFactory<TitledPanel> summary, boolean expanded) {
    return (state, detail) -> {
      PanelGroup group = new PanelGroup();
      for (Panel track : detail) {
        group.add(track);
      }
      return new Group(summary.createPanel(state), group, expanded, null, false);
    };
  }

  public static TrackConfig.Group.UiFactory group(TrackConfig.Track.UiFactory<TitledPanel> summary,
      boolean expanded, BiConsumer<PanelGroup, Boolean> filter, boolean initial) {
    return (state, detail) -> {
      PanelGroup group = new PanelGroup();
      for (Panel track : detail) {
        group.add(track);
      }
      if (initial) {
        filter.accept(group, true);
      }
      return new Group(summary.createPanel(state), group, expanded,
          filtered -> filter.accept(group, filtered), initial);
    };
  }

  private static class Single extends Panel.Base {
    private final TrackPanel track;
    private final boolean sep;
    protected final Consumer<Boolean> filter;

    protected boolean filtered;

    public Single(TrackPanel track, boolean sep, Consumer<Boolean> filter, boolean filtered) {
      this.track = track;
      this.sep = sep;
      this.filter = filter;
      this.filtered = filtered;
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
        ctx.drawTextLeftTruncate(Fonts.Style.Normal, track.getTitle(),
            10, 0, LABEL_WIDTH - 10 - ((filter != null) ? TOGGLE_ICON_OFFSET : 0), TITLE_HEIGHT);
        if (filter != null) {
          ctx.drawIcon(filtered ? unfoldMore(ctx.theme) : unfoldLess(ctx.theme),
              LABEL_WIDTH - TOGGLE_ICON_OFFSET, 0, TITLE_HEIGHT);
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
      if (filter != null &&
          y < TITLE_HEIGHT && x >= LABEL_WIDTH - TOGGLE_ICON_OFFSET && x < LABEL_WIDTH) {
        return new FilterToggler(track.onMouseMove(m, x, y));
      } else {
        return track.onMouseMove(m, x, y);
      }
    }

    private class FilterToggler implements Hover {
      private final Hover child;

      public FilterToggler(Hover child) {
        this.child = child;
      }

      @Override
      public Area getRedraw() {
        return child.getRedraw();
      }

      @Override
      public void stop() {
        child.stop();
      }

      @Override
      public boolean click() {
        child.click();
        filtered = !filtered;
        filter.accept(filtered);
        return true;
      }

      @Override
      public Cursor getCursor(Display display) {
        return display.getSystemCursor(SWT.CURSOR_HAND);
      }
    }
  }

  private static class Group extends Panel.Base {
    private final TitledPanel summary;
    private final Panel detail;
    protected final Consumer<Boolean> filter;

    protected boolean expanded;
    protected boolean filtered;

    public Group(TitledPanel summary, Panel detail, boolean expanded, Consumer<Boolean> filter,
        boolean filtered) {
      this.summary = summary;
      this.detail = detail;
      this.expanded = expanded;
      this.filter = filter;
      this.filtered = filtered;
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
        if (filter != null) {
          ctx.drawIcon(filtered ? unfoldMore(ctx.theme) : unfoldLess(ctx.theme),
              LABEL_WIDTH - TOGGLE_ICON_OFFSET, 0, TITLE_HEIGHT);
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
              LABEL_OFFSET, 0, LABEL_WIDTH - LABEL_OFFSET, TITLE_HEIGHT);
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
      if (expanded) {
        if (y < TITLE_HEIGHT) {
          if (filter != null && x >= LABEL_WIDTH - TOGGLE_ICON_OFFSET && x < LABEL_WIDTH) {
            return new FilterToggler();
          } else if (x < LABEL_OFFSET + m.measure(Fonts.Style.Normal, summary.getTitle()).w) {
            return new ExpansionToggler(Hover.NONE);
          } else {
            return Hover.NONE;
          }
        } else {
          return detail.onMouseMove(m, x, y - TITLE_HEIGHT).translated(0, TITLE_HEIGHT);
        }
      } else {
        if (y < TITLE_HEIGHT &&
            x < Math.min(LABEL_WIDTH, LABEL_OFFSET + m.measure(Fonts.Style.Normal, summary.getTitle()).w)) {
          return new ExpansionToggler(summary.onMouseMove(m, x, y));
        } else {
          return summary.onMouseMove(m, x, y);
        }
      }
    }

    private class ExpansionToggler implements Hover {
      private final Hover child;

      public ExpansionToggler(Hover child) {
        this.child = child;
      }

      @Override
      public Area getRedraw() {
        return child.getRedraw();
      }

      @Override
      public void stop() {
        child.stop();
      }

      @Override
      public boolean click() {
        child.click();
        expanded = !expanded;
        return true;
      }

      @Override
      public Cursor getCursor(Display display) {
        return display.getSystemCursor(SWT.CURSOR_HAND);
      }
    }

    private class FilterToggler implements Hover {
      public FilterToggler() {
      }

      @Override
      public boolean click() {
        filtered = !filtered;
        filter.accept(filtered);
        return true;
      }

      @Override
      public Cursor getCursor(Display display) {
        return display.getSystemCursor(SWT.CURSOR_HAND);
      }
    }
  }
}
