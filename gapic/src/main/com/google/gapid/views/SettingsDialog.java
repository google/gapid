/*
 * Copyright (C) 2017 Google Inc.
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
package com.google.gapid.views;

import static com.google.gapid.proto.service.Service.ClientAction.Show;
import static com.google.gapid.views.WelcomeDialog.showPrivacyPolicy;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.FileTextbox;
import com.google.gapid.widgets.Theme;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;

/**
 * Dialog that allows the user to modify application settings.
 */
public class SettingsDialog extends DialogBase {
  private final Settings settings;
  private Button autoCheckForUpdates;
  private Button sendAnalytics;
  private Button disableReplayOptimization;
  private Button sendCrashReports;
  private FileTextbox adbPath;
  private Label restartLabel;

  public SettingsDialog(Shell parent, Settings settings, Theme theme) {
    super(parent, theme);
    this.settings = settings;
  }

  public static void showSettingsDialog(Shell shell, Models models, Theme theme) {
    models.analytics.postInteraction(View.Main, "settings", Show);
    new SettingsDialog(shell, models.settings, theme).open();
  }

  private void update() {
    settings.adb = adbPath.getText().trim();
    settings.disableReplayOptimization = disableReplayOptimization.getSelection();
    settings.setAnalyticsEnabled(sendAnalytics.getSelection());
    settings.reportCrashes = sendCrashReports.getSelection();
    settings.autoCheckForUpdates = autoCheckForUpdates.getSelection();
    settings.onChange();
  }

  @Override
  public String getTitle() {
    return Messages.SETTINGS_TITLE;
  }

  @Override
  protected Control createDialogArea(Composite parent) {
    Composite area = (Composite)super.createDialogArea(parent);

    Composite container = createComposite(area, new GridLayout(2, false));
    container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    createLabel(container, "Path to adb:");
    adbPath = withLayoutData(new FileTextbox.File(container, settings.adb) {
      @Override
      protected void configureDialog(FileDialog dialog) {
        dialog.setText("Path to adb:");
      }
    }, new GridData(SWT.FILL, SWT.FILL, true, false));
    adbPath.addBoxListener(SWT.Modify, this::onSettingChanged);

    disableReplayOptimization = withLayoutData(
        createCheckbox(container, "Disable replay optimization", settings.disableReplayOptimization),
        withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));

    sendAnalytics = withLayoutData(
        createCheckbox(container, Messages.ANALYTICS_OPTION, settings.analyticsEnabled()),
        withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));

    sendCrashReports = withLayoutData(
        createCheckbox(container, Messages.CRASH_REPORTING_OPTION, settings.reportCrashes),
        withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));

    autoCheckForUpdates = withLayoutData(
        createCheckbox(container, Messages.UPDATE_CHECK_OPTION, settings.autoCheckForUpdates),
        withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));

    withLayoutData(
        createLink(container, Messages.PRIVACY_POLICY, e -> showPrivacyPolicy()),
        withIndents(
            withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1), 0, 20));

    restartLabel = withLayoutData(
        createLabel(container, "Changes require restart to take effect"),
        withIndents(
            withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1), 0, 20));
    restartLabel.setVisible(false);
    restartLabel.setForeground(restartLabel.getDisplay().getSystemColor(SWT.COLOR_RED));

    return area;
  }

  protected void onSettingChanged(@SuppressWarnings("unused") Event event) {
    boolean restartNeeded = !settings.adb.equals(adbPath.getText());
    restartLabel.setVisible(restartNeeded);
  }

  @Override
  protected void createButtonsForButtonBar(Composite parent) {
    createButton(parent, IDialogConstants.CANCEL_ID, IDialogConstants.CANCEL_LABEL, false);
    createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
  }

  @Override
  protected void buttonPressed(int buttonId) {
    if (buttonId == IDialogConstants.OK_ID) {
      update();
      settings.save();
    }
    super.buttonPressed(buttonId);
  }
}
