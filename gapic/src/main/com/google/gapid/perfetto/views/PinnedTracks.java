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

public class PinnedTracks extends Panel.Base {
  private final double BORDER_WIDTH = 3;

  private final PanelGroup group = new PanelGroup();

  public void pin(Panel panel) {
    group.add(panel);
  }

  public void unpin(Panel panel) {
    group.remove(panel);
  }

  public void clear() {
    group.clear();
  }

  @Override
  public double getPreferredHeight() {
    double h = group.getPreferredHeight();
    return (h == 0) ? 0 : h + BORDER_WIDTH;
  }

  @Override
  public void setSize(double w, double h) {
    super.setSize(w, h);
    group.setSize(w, Math.max(0, h - BORDER_WIDTH));
  }

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    group.render(ctx, repainter);
    if (height > 0) {
      ctx.setBackgroundColor(StyleConstants.colors().panelBorder);
      ctx.fillRect(0, height - BORDER_WIDTH, width, BORDER_WIDTH);
    }
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
}
