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

import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.RenderContext;

/**
 * A {@link Panel} displaying a highlighted title in the UI.
 */
public class TitlePanel extends Panel.Base implements TitledPanel, CopyablePanel<TitlePanel> {
  private final String title;

  public TitlePanel(String title) {
    this.title = title;
  }

  @Override
  public TitlePanel copy() {
    // We are stateless, bum ba-dum bum bum bum.
    return this;
  }

  @Override
  public double getPreferredHeight() {
    return 25;
  }

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    // Do nothing.
  }

  @Override
  public void decorateTitle(RenderContext ctx, Repainter repainter) {
    ctx.setForegroundColor(colors().panelBorder);
    ctx.drawLine(0, 24, width, 24);
  }

  @Override
  public String getTitle() {
    return title;
  }
}
