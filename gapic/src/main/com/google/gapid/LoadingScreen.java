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
import static com.google.gapid.views.TracerDialog.showOpenTraceDialog;
import static com.google.gapid.views.TracerDialog.showTracingDialog;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createMenuItem;
import static com.google.gapid.widgets.Widgets.filling;
import static com.google.gapid.widgets.Widgets.recursiveAddListener;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMarginAndSpacing;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.server.Client;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.views.AboutDialog;
import com.google.gapid.widgets.CenteringLayout;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Menu;

import java.io.File;

/**
 * The loading screen is a minimal view shown while the UI is loading, looking for gapis, etc.
 */
public class LoadingScreen extends Composite {
  private static final int MAX_FILE_NAME = 80;

  private final Theme theme;
  private final Label statusLabel;
  private final Composite optionsContainer;
  private OptionBar recentBar;
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

    Label versionLabel = createLabel(container, "Version " + GAPID_VERSION.toStringWithYear(false));
    versionLabel.setForeground(theme.welcomeVersionColor());
    versionLabel.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

    statusLabel = createLabel(container, "Starting up...");
    statusLabel.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

    optionsContainer = createComposite(container, filling(new RowLayout(SWT.VERTICAL), true, true));
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
  }

  /**
   * Initialize the links for layout settings. Hide them until server set up.
   */
  private void createOptions() {
    OptionBar.withShortcut(optionsContainer, theme.add(), "Capture a new trace", "T", e -> {
      showTracingDialog(
          checkNotNull(client), getShell(), checkNotNull(models), checkNotNull(widgets));
    });
    OptionBar.withShortcut(optionsContainer, theme.open(), "Open an existing trace", "O", e -> {
      showOpenTraceDialog(getShell(), checkNotNull(models));
    });
    recentBar = OptionBar.withDropDown(theme, optionsContainer, theme.recent(), "Open recent trace",
        e -> {
      Menu popup = new Menu(optionsContainer);
      if (checkNotNull(models).settings.files().getRecentCount() == 0) {
        createMenuItem(popup, "No recent files.", 0,  ev -> { /* empty */ }).setEnabled(false);
      } else {
        for (String file : checkNotNull(models).settings.files().getRecentList()) {
          createMenuItem(popup, truncate(file), 0, ev -> {
            checkNotNull(models).analytics.postInteraction(View.Welcome, ClientAction.OpenRecent);
            checkNotNull(models).capture.loadCapture(new File(file));
          });
        }
      }
      popup.addListener(SWT.Hide, ev -> scheduleIfNotDisposed(popup, popup::dispose));
      popup.setLocation(optionsContainer.toDisplay(bottomLeft((recentBar.getBounds()))));
      popup.setVisible(true);
    });
    OptionBar.simple(optionsContainer, theme.help(), "Help", e -> {
      AboutDialog.showHelp(checkNotNull(models).analytics);
    });

    optionsContainer.setVisible(false);
  }

  private static String truncate(String file) {
    if (file.length() <= MAX_FILE_NAME) {
      return file;
    }
    for (int p = file.indexOf(File.separatorChar, 1); file.length() > MAX_FILE_NAME && p >= 0; ) {
      file = file.substring(p);
      p = file.indexOf(File.separatorChar, 1);
    }

    if (file.length() > MAX_FILE_NAME) {
      return "..." + file.substring(file.length() - MAX_FILE_NAME + 3);
    }
    return "..." + file;
  }

  private static class OptionBar extends Composite {
    private OptionBar(Composite parent, Image icon, String label, Image dropDown, String shortcut) {
      super(parent, SWT.NONE);
      setLayout(withMarginAndSpacing(new GridLayout(4, false), 10, 2, 0, 0));

      createLabel(this, "", icon);
      withLayoutData(createLabel(this, label),
          withIndents(new GridData(SWT.LEFT, SWT.CENTER, false, false), 10, 0));
      if (dropDown != null) {
        createLabel(this, "", dropDown);
      }
      if (shortcut != null) {
        withLayoutData(createLabel(this, (OS.isMac ? "\u2318 + " : "Ctrl + ") + shortcut),
            withIndents(new GridData(SWT.RIGHT, SWT.CENTER, true, false), 40, 0));
      }

      setCursor(getDisplay().getSystemCursor(SWT.CURSOR_HAND));

      Color fgColor = parent.getDisplay().getSystemColor(SWT.COLOR_LIST_SELECTION_TEXT);
      Color bgColor = parent.getDisplay().getSystemColor(SWT.COLOR_LIST_SELECTION);
      recursiveAddListener(this, SWT.MouseEnter, e -> {
        Widgets.recursiveSetForeground(this, fgColor);
        Widgets.recursiveSetBackground(this, bgColor);
      });
      recursiveAddListener(this, SWT.MouseExit, e -> {
        Widgets.recursiveSetForeground(this, null);
        Widgets.recursiveSetBackground(this, null);
      });
    }

    public static OptionBar simple(Composite parent, Image icon, String label, Listener onClick) {
      OptionBar bar = new OptionBar(parent, icon, label, null, null);
      recursiveAddListener(bar, SWT.MouseUp, onClick);
      return bar;
    }

    public static OptionBar withShortcut(
        Composite parent, Image icon, String label, String shortcut, Listener onClick) {
      OptionBar bar = new OptionBar(parent, icon, label, null, shortcut);
      recursiveAddListener(bar, SWT.MouseUp, onClick);
      return bar;
    }

    public static OptionBar withDropDown(
        Theme theme, Composite parent, Image icon, String label, Listener onClick) {
      OptionBar bar = new OptionBar(parent, icon, label, theme.arrowDropDownLight(), null);
      recursiveAddListener(bar, SWT.MouseUp, onClick);
      return bar;
    }
  }
}
