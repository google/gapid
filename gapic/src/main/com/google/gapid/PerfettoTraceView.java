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
package com.google.gapid;

import com.google.gapid.models.Models;
import com.google.gapid.perfetto.QueryViewer;
import com.google.gapid.perfetto.TraceView;
import com.google.gapid.perfetto.views.StyleConstants;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.action.Action;
import org.eclipse.jface.action.MenuManager;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.TabItem;

/**
 * Main view shown when a Perfetto trace is loaded.
 */
public class PerfettoTraceView extends Composite implements MainWindow.MainView {
  private final Models models;

  public PerfettoTraceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout());

    TabFolder folder = new TabFolder(this, SWT.TOP);
    TabItem main = new TabItem(folder, SWT.NONE);
    main.setText("System Trace");
    main.setControl(new TraceView(folder, models, widgets));

    TabItem query = new TabItem(folder, SWT.NONE);
    query.setText("Query");
    query.setControl(new QueryViewer(folder, models));
  }

  @Override
  public void updateViewMenu(MenuManager manager) {
    manager.removeAll();

    Action darkMode = MainWindow.MenuItems.ViewDarkMode.createCheckbox((dark) -> {
      models.settings.perfettoDarkMode = dark;
      StyleConstants.setDark(dark);
      Rectangle size = getClientArea();
      redraw(size.x, size.y, size.width, size.height, true);
    });
    darkMode.setChecked(models.settings.perfettoDarkMode);
    StyleConstants.setDark(models.settings.perfettoDarkMode);

    manager.add(darkMode);
  }
}
