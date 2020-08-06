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

import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.PanelGroup;
import com.google.gapid.perfetto.canvas.RenderContext;

/**
 * A {@link Panel} that can make copies of itself.
 */
public interface CopyablePanel<T extends CopyablePanel<T>> extends Panel {
  /**
   * Makes a copy of this panel and returns the new instance. This is used when showing the same
   * data in multiple places is desired (such as pinning). The data should be rendered the same
   * in each instance, so some render state may need to be shared, however, hover state, for
   * example, should not be shared.
   */
  public T copy();

  public static class Group implements CopyablePanel<Group>, Panel.Grouper {
    private final PanelGroup group = new PanelGroup();

    @Override
    public Group copy() {
      Group copy = new Group();
      for (int i = 0; i < group.getPanelCount(); i++) {
        copy.add(((CopyablePanel<?>)group.getPanel(i)).copy());
        copy.group.setVisible(i, group.isVisible(i));
      }
      return copy;
    }

    public void add(CopyablePanel<?> child) {
      group.add(child);
    }

    @Override
    public void setVisible(int idx, boolean visible) {
      group.setVisible(idx, visible);
    }

    @Override
    public double getPreferredHeight() {
      return group.getPreferredHeight();
    }

    @Override
    public void setSize(double w, double h) {
      group.setSize(w, h);
    }

    @Override
    public void render(RenderContext ctx, Repainter repainter) {
      group.render(ctx, repainter);
    }

    @Override
    public Dragger onDragStart(double x, double y, int mods) {
      return group.onDragStart(x, y, mods);
    }

    @Override
    public Hover onMouseMove(
        Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
      return group.onMouseMove(m, repainter, x, y, mods);
    }

    @Override
    public void visit(Visitor v, Area area) {
      group.visit(v, area);
    }

    @Override
    public int getPanelCount() {
      return group.getPanelCount();
    }

    @Override
    public Panel getPanel(int idx) {
      return group.getPanel(idx);
    }

    @Override
    public void setFiltered(int idx, boolean filtered) {
      group.setFiltered(idx, filtered);
    }
  }
}
