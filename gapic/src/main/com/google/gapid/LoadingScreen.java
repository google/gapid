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

import static com.google.common.base.Preconditions.checkNotNull;
import static com.google.gapid.util.GapidVersion.GAPID_VERSION;
import static com.google.gapid.util.GeoUtils.bottomLeft;
import static com.google.gapid.views.AboutDialog.showHelp;
import static com.google.gapid.views.TracerDialog.showOpenTraceDialog;
import static com.google.gapid.views.TracerDialog.showTracingDialog;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createMenuItem;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withMargin;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.server.Client;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
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

import java.io.File;

/**
 * The loading screen is a minimal view shown while the UI is loading, looking for gapis, etc.
 */
public class LoadingScreen extends Composite {
  private final Theme theme;
  private final Label statusLabel;
  private final Composite optionsContainer;
  private Link recentLink;
  private Models models;
  private Client client;
  private Widgets widgets;

  public LoadingScreen(Composite parent, Theme theme) {
    super(parent, SWT.NONE);
    this.theme = theme;
    setLayout(CenteringLayout.goldenRatio());

    Composite container = createComposite(this, new GridLayout(1, false));
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

    optionsContainer = createComposite(container, withMargin(new GridLayout(3, false), 15, 5));
    optionsContainer.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));
    createOptions();
  }

  public void setText(String status) {
    statusLabel.setText(status);
    statusLabel.requestLayout();
  }

  /**
   * Hide the messaging box and display the links after server set up.
   */
  public void showOptions(Client newClient, Models newModels, Widgets newWidgets) {
    this.client = newClient;
    this.models = newModels;
    this.widgets = newWidgets;

    statusLabel.setVisible(false);
    optionsContainer.setVisible(true);
    if (models.settings.getRecent().length <= 0) {
      recentLink.setEnabled(false);
    }
  }

  /**
   * Initialize the links for layout settings. Hide them until server set up.
   */
  private void createOptions() {
    createLabel(optionsContainer, "", theme.add());
    createLink(optionsContainer, "<a>Capture a new trace</a>", e -> {
      showTracingDialog(checkNotNull(client), getShell(), checkNotNull(models), checkNotNull(widgets));
    });
    Label captureHint = createLabel(optionsContainer, (OS.isMac ? "\u2318" : "Ctrl") + " + T");
    captureHint.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, true, false));
    captureHint.setForeground(theme.shortcutKeyHintColor());

    createLabel(optionsContainer, "", theme.open());
    createLink(optionsContainer, "<a>Open an existing trace</a>", e -> {
      showOpenTraceDialog(getShell(), checkNotNull(this.models));
    });
    Label openHint = createLabel(optionsContainer, (OS.isMac ? "\u2318" : "Ctrl") + " + O");
    openHint.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, true, false));
    openHint.setForeground(theme.shortcutKeyHintColor());

    createLabel(optionsContainer, "", theme.recent());
    recentLink = createLink(optionsContainer, "<a>Open recent traces</a>", e -> {
      Menu popup = new Menu(optionsContainer);
      for (String file : checkNotNull(models).settings.recentFiles) {
        createMenuItem(popup, file, 0, ev -> {
          checkNotNull(models).analytics.postInteraction(View.Welcome, ClientAction.OpenRecent);
          checkNotNull(models).capture.loadCapture(new File(file));
        });
      }
      popup.addListener(SWT.Hide, ev -> scheduleIfNotDisposed(popup, popup::dispose));
      popup.setLocation(optionsContainer.toDisplay(bottomLeft(((Link)e.widget).getBounds())));
      popup.setVisible(true);
    });
    recentLink.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, true, false, 2, 1));

    createLabel(optionsContainer, "", theme.help());
    createLink(optionsContainer, "<a>Help</a>", e -> showHelp(models.analytics));

    optionsContainer.setVisible(false);
  }
}
