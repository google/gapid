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
import static com.google.gapid.widgets.Widgets.addListenerToComposite;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createMenuItem;
import static com.google.gapid.widgets.Widgets.filling;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpacing;

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
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.events.MouseTrackListener;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Menu;

import java.io.File;

/**
 * The loading screen is a minimal view shown while the UI is loading, looking for gapis, etc.
 */
public class LoadingScreen extends Composite {
  protected final Theme theme;
  private final Label statusLabel;
  private final Composite optionsContainer;
  private Link recentLink;
  private Models models;
  private Client client;
  private Widgets widgets;

  public LoadingScreen(Composite parent, Theme theme) {
    super(parent, SWT.NONE);
    this.theme = theme;

    // Divide the screen into two parts: main window and help link.
    setLayout(new GridLayout(1, false));
    Composite goldenRatioContainer = withLayoutData(createComposite(this, CenteringLayout.goldenRatio()),
        new GridData(SWT.FILL, SWT.FILL, true, true));
    Composite helpLinkContainer = withLayoutData(createComposite(this, withMargin(new RowLayout(), 10, 0)),
        new GridData(SWT.RIGHT, SWT.BOTTOM, true, false));

    // Initialize the constant part in the main window, LOGO, title and version.
    Composite container = createComposite(goldenRatioContainer, withSpacing(new GridLayout(1, false), 0, 0));
    createLabel(container, "", theme.dialogLogo());
    createLabel(container, Messages.WINDOW_TITLE, theme.welcomeTitleFont(), theme.welcomeTitleColor());
    createLabel(container, "Version " + GAPID_VERSION.toFriendlyString(),
        theme.welcomeVersionFont(), theme.welcomeVersionColor());

    // Initialize the dynamic part in the main window, status label and option panel.
    createLabel(container, "");
    statusLabel = createLabel(container, "Starting up...", theme.welcomeLabelFont(), theme.welcomeLabelColor());
    optionsContainer = createComposite(container, withSpacing(filling(new RowLayout(SWT.VERTICAL), true, true), 6));
    createOptions();

    // Override default GridData for each child to avoid components jumping when resizing.
    for (Control control : container.getChildren()) {
      control.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));
    }

    createLink(helpLinkContainer, "<a>Learn more about GAPID</a>", e -> showHelp(models.analytics),
        theme.welcomeLabelFont(), theme.welcomeHelpColor());
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
    OptionBar addOption = new OptionBar(optionsContainer, theme.add(), Messages.WINDOW_CAPTURE,
        (OS.isMac ? "\u2318" : "Ctrl") + " + T");
    addOption.addClickListener(e -> {
      showTracingDialog(checkNotNull(client), getShell(), checkNotNull(models), checkNotNull(widgets));
    });

    OptionBar openOption = new OptionBar(optionsContainer, theme.open(), Messages.WINDOW_OPEN,
        (OS.isMac ? "\u2318" : "Ctrl") + " + O");
    openOption.addClickListener(e -> {
      showOpenTraceDialog(getShell(), checkNotNull(this.models));
    });

    OptionBar recentOption = new OptionBar(optionsContainer, theme.recent(), Messages.WINDOW_RECENT, "");
    recentOption.addClickListener(e -> {
      Menu popup = new Menu(optionsContainer);
      for (String file : checkNotNull(models).settings.recentFiles) {
        createMenuItem(popup, file, 0, ev -> {
          checkNotNull(models).analytics.postInteraction(View.Welcome, ClientAction.OpenRecent);
          checkNotNull(models).capture.loadCapture(new File(file));
        });
      }
      popup.addListener(SWT.Hide, ev -> scheduleIfNotDisposed(popup, popup::dispose));
      popup.setLocation(optionsContainer.toDisplay(bottomLeft(recentOption.getBounds())));
      popup.setVisible(true);
    });

    optionsContainer.setVisible(false);
  }

  private class OptionBar extends Composite{
    public OptionBar(Composite parent, Image iconImage, String msgString, String shortcutString) {
      super(parent, SWT.NONE);

      setLayout(withMargin(new GridLayout(3, false), 25, 0));
      createLabel(this, "", iconImage);
      createLabel(this, msgString, theme.welcomeLabelFont(), theme.welcomeLabelColor());
      createLabel(this, shortcutString, theme.welcomeLabelFont(), theme.shortcutKeyHintColor())
        .setLayoutData(withIndents(new GridData(SWT.RIGHT, SWT.CENTER, true, false), 40, 0));
      addSwipeColorListener();
    }

    public void addClickListener(Listener listener) {
      addListenerToComposite(this, SWT.MouseUp, listener);
    }

    private void addSwipeColorListener() {
      Composite highlightedComposite = this;
      highlightedComposite.setBackgroundMode(SWT.INHERIT_FORCE);
      Color defaultColor = this.getBackground();
      MouseTrackListener listener = new MouseTrackListener() {
        @Override
        public void mouseHover(MouseEvent e) {
          highlightedComposite.setBackground(theme.welcomeHoveringBackgroundColor());
        }

        @Override
        public void mouseExit(MouseEvent e) {
          highlightedComposite.setBackground(defaultColor);
        }

        @Override
        public void mouseEnter(MouseEvent e) {
          highlightedComposite.setBackground(theme.welcomeHoveringBackgroundColor());
        }
      };
      addListenerToComposite(this, listener);
    }
  }
}
