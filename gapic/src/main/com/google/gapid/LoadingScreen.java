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

import static com.google.gapid.util.GapidVersion.GAPID_VERSION;
import static com.google.gapid.util.GeoUtils.bottomLeft;
import static com.google.gapid.views.AboutDialog.showHelp;
import static com.google.gapid.views.TracerDialog.showOpenTraceDialog;
import static com.google.gapid.views.TracerDialog.showTracingDialog;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createMenuItem;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.server.Client;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.CenteringLayout;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Shell;

import java.io.File;

/**
 * The loading screen is a minimal view shown while the UI is loading, looking for gapis, etc.
 */
public class LoadingScreen extends Composite {
  private final Label statusLabel;
  private final Composite container;

  public LoadingScreen(Composite parent, Theme theme) {
    super(parent, SWT.NONE);
    setLayout(CenteringLayout.goldenRatio());

    container = Widgets.createComposite(this, new GridLayout(1, false));
    createLabel(container, "", theme.dialogLogo())
        .setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

    Label titleLabel = createLabel(container, Messages.WINDOW_TITLE);
    titleLabel.setFont(theme.bigBoldFont());
    titleLabel.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

    Label versionLabel = createLabel(container, "Version " + GAPID_VERSION.toFriendlyString());
    versionLabel.setForeground(theme.welcomeVersionColor());
    versionLabel.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

    statusLabel = createLabel(container, "Starting up...");
    statusLabel.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));
    statusLabel.setVisible(false);
  }

  public void setText(String status) {
    statusLabel.setText(status);
    statusLabel.requestLayout();
  }

  public void createLinks(Client client, Shell shell, Models models, Widgets widgets) {
    createLink(container, "<a>Open Trace...</a>", e -> {
      showOpenTraceDialog(getShell(), models);
    });

    String[] files = models.settings.getRecent();
    if (files.length > 0) {
      createLink(container, "<a>Open Recent...</a>", e -> {
        Menu popup = new Menu(container);
        for (String file : models.settings.recentFiles) {
          createMenuItem(popup, file, 0, ev -> {
            models.analytics.postInteraction(View.Welcome, ClientAction.OpenRecent);
            models.capture.loadCapture(new File(file));
          });
        }
        popup.addListener(SWT.Hide, ev -> scheduleIfNotDisposed(popup, popup::dispose));

        popup.setLocation(container.toDisplay(bottomLeft(((Link)e.widget).getBounds())));
        popup.setVisible(true);
      });
    }

    createLink(container, "<a>Capture Trace...</a>", e -> {
      showTracingDialog(client, shell, models, widgets);
    });

    createLink(container, "<a>Help...</a>", e -> showHelp(models.analytics));
  }
}
