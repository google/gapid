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
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.action.Action;
import org.eclipse.jface.action.MenuManager;
import org.eclipse.jface.window.SameShellProvider;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Layout;
import org.eclipse.swt.widgets.Shell;

/**
 * Main view shown when a Perfetto trace is loaded.
 */
public class PerfettoTraceView extends Composite implements MainWindow.MainView {
  protected final Models models;
  protected final Widgets widgets;

  public PerfettoTraceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;

    setLayout(new FillLayout());
    new TraceView(this, models, widgets);
  }

  @Override
  public void updateViewMenu(MenuManager manager) {
    manager.removeAll();

    Action darkMode = MainWindow.MenuItems.ViewDarkMode.createCheckbox((dark) -> {
      models.settings.writeUi().getPerfettoBuilder().setDarkMode(dark);
      StyleConstants.setDark(dark);
      Rectangle size = getClientArea();
      redraw(size.x, size.y, size.width, size.height, true);
    });
    boolean dark = models.settings.ui().getPerfettoOrBuilder().getDarkMode();
    darkMode.setChecked(dark);
    StyleConstants.setDark(dark);

    Action queryView = MainWindow.MenuItems.ViewQueryShell.create(() -> {
      Window window = new Window(new SameShellProvider(this)) {
        @Override
        protected void configureShell(Shell shell) {
          shell.setText(Messages.QUERY_VIEW_WINDOW_TITLE);
          shell.setImages(widgets.theme.windowLogo());
          super.configureShell(shell);
        }

        @Override
        protected Control createContents(Composite parent) {
          return new QueryViewer(parent, models);
        }

        @Override
        protected Layout getLayout() {
          return new FillLayout();
        }

        @Override
        protected Point getInitialSize() {
          return new Point(800, 600);
        }
      };
      window.open();
    });

    manager.add(darkMode);
    manager.add(queryView);
  }
}
