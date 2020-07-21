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
package com.google.gapid.perfetto.canvas;

import com.google.common.collect.Lists;

import java.util.Collections;
import java.util.List;

/**
 * Contains a list of child {@link Panel panels}, which are laid out vertically.
 */
public class PanelGroup extends Panel.Base implements Panel.Grouper {
  private final List<PanelGroup.Child> panels = Lists.newArrayList();

  public PanelGroup() {
  }

  @Override
  public int getPanelCount() {
    return panels.size();
  }

  @Override
  public Panel getPanel(int idx) {
    return panels.get(idx).panel;
  }

  public void add(Panel panel) {
    double y = panels.isEmpty() ? 0 : panels.get(panels.size() - 1).getNextY();
    panels.add(new Child(panel, y, 0));
  }

  public void remove(int idx) {
    double y = panels.remove(idx).y;
    for (int i = idx; i < panels.size(); i++) {
      Child child = panels.get(i);
      child.y = y;
      y += child.h;
    }
  }

  public void remove(Panel panel) {
    for (int i = 0; i < panels.size(); i++) {
      if (panels.get(i).panel == panel) {
        remove(i);
        return;
      }
    }
  }

  public void clear() {
    panels.clear();
  }

  public boolean isVisible(int idx) {
    return panels.get(idx).isVisible();
  }

  @Override
  public void setVisible(int idx, boolean visible) {
    panels.get(idx).setVisibile(visible);
  }

  @Override
  public void setFiltered(int idx, boolean include) {
    panels.get(idx).setFiltered(include);
  }

  @Override
  public double getPreferredHeight() {
    double y = 0;
    for (Child child : panels) {
      double want = child.isVisible() ? child.panel.getPreferredHeight() : 0;
      child.y = y;
      child.h = want;
      y += want;
    }
    return y;
  }

  @Override
  public void setSize(double w, double h) {
    super.setSize(w, h);
    for (PanelGroup.Child child : panels) {
      if (child.isVisible()) {
        child.panel.setSize(w, child.h);
      }
    }
  }

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    Area clip = ctx.getClip();
    int first = findPanelIdx(clip.y);

    for (int i = first; i < panels.size(); i++) {
      PanelGroup.Child child = panels.get(i);
      if (!child.isVisible()) {
        continue;
      }

      ctx.withTranslation(0, child.y, () -> {
        ctx.withClip(clip.x, 0, clip.w, child.h, () -> {
          child.panel.render(ctx, repainter.translated(0, child.y));
        });
      });
      if (child.getNextY() >= clip.y + clip.h) {
        break;
      }
    }
  }

  @Override
  public void visit(Visitor v, Area area) {
    super.visit(v, area);
    int first = findPanelIdx(area.y);
    for (int i = first; i < panels.size(); i++) {
      PanelGroup.Child child = panels.get(i);
      if (!child.isVisible()) {
        continue;
      }
      area.intersect(0, child.y, width, child.h)
          .ifNotEmpty(a -> child.panel.visit(v, a.translate(0, -child.y)));
    }
  }

  @Override
  public Dragger onDragStart(double x, double y, int mods) {
    Child child = findPanel(y);
    if (child == null) {
      return Dragger.NONE;
    }
    return child.panel.onDragStart(x, y - child.y, mods).translated(0, child.y);
  }

  @Override
  public Hover onMouseMove(Fonts.TextMeasurer m, double x, double y, int mods) {
    Child child = findPanel(y);
    if (child == null) {
      return Hover.NONE;
    }
    return child.panel.onMouseMove(m, x, y - child.y, mods).translated(0, child.y);
  }

  private int findPanelIdx(double y) {
    int first = Collections.binarySearch(panels, null, (c1, ign) -> {
      if (c1.isVisible()) {
        return (y >= c1.y && y < c1.getNextY()) ? 0 : Double.compare(c1.y, y);
      } else {
        return c1.y < y ? -1 : 1;
      }
    });
    return (first < 0) ? -first - 1 : first;
  }

  private Child findPanel(double y) {
    int idx = findPanelIdx(y);
    return (idx < panels.size()) ? panels.get(idx) : null;
  }

  private static class Child {
    public final Panel panel;
    public double y, h;
    private boolean visible;
    private boolean filtered;

    public Child(Panel panel, double y, double h) {
      this.panel = panel;
      this.y = y;
      this.h = Math.max(0, h);
      this.visible = true;
      this.filtered = false;
    }

    public double getNextY() {
      return y + h;
    }

    public void setVisibile(boolean visible) {
      this.visible = visible;
    }

    public void setFiltered(boolean filtered) {
      this.filtered = filtered;
    }

    public boolean isVisible() {
      return visible && !filtered;
    }
  }
}
