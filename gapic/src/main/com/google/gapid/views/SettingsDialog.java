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

import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.service.Service.ClientAction;
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
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Shell;

/**
 * Dialog that allows the user to modify application settings.
 */
public class SettingsDialog extends DialogBase {
  protected final Models models;
  private SettingsFormBase form;
  protected Button disableReplayOptimization;

  public SettingsDialog(Shell parent, Models models, Theme theme) {
    super(parent, theme);
    this.models = models;
  }

  public static void showSettingsDialog(Shell shell, Models models, Theme theme) {
    models.analytics.postInteraction(View.Settings, ClientAction.Show);
    new SettingsDialog(shell, models, theme).open();
  }

  @Override
  public String getTitle() {
    return Messages.SETTINGS_TITLE;
  }

  @Override
  protected Control createDialogArea(Composite parent) {
    Composite area = (Composite)super.createDialogArea(parent);

    form = new SettingsFormBase(models, area) {
      @Override
      protected void beforeAnalytics() {
        disableReplayOptimization = withLayoutData(createCheckbox(
            this, "Disable replay optimization", models.settings.disableReplayOptimization),
            withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
      }
    };
    form.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    return area;
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
      form.save();
    }
    super.buttonPressed(buttonId);
  }

  private void update() {
    models.settings.disableReplayOptimization = disableReplayOptimization.getSelection();
  }

  public static class SettingsFormBase extends Composite {
    private final Models models;
    private final FileTextbox adbPath;
    private final Button allowAnalytics;
    private final Button allowCrashReports;
    private final Button allowUpdateChecks;
    private final Button includePrerelease;

    public SettingsFormBase(Models models, Composite parent) {
      this(models, parent, 5, 5);
    }

    public SettingsFormBase(Models models, Composite parent, int marginWidth, int marginHeight) {
      super(parent, SWT.NONE);
      this.models = models;
      setLayout(withMargin(new GridLayout(2, false), marginWidth, marginHeight));

      createLabel(this, "Path to adb:");
      adbPath = withLayoutData(new FileTextbox.File(this, models.settings.adb) {
        @Override
        protected void configureDialog(FileDialog dialog) {
          dialog.setText("Path to adb:");
        }
      }, new GridData(SWT.FILL, SWT.FILL, true, false));

      beforeAnalytics();

      allowAnalytics = withLayoutData(
          createCheckbox(this, Messages.ANALYTICS_OPTION, models.settings.analyticsEnabled()),
          withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
      allowCrashReports = withLayoutData(
          createCheckbox(this, Messages.CRASH_REPORTING_OPTION, models.settings.reportCrashes),
          withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
      allowUpdateChecks = withLayoutData(
          createCheckbox(this, Messages.UPDATE_CHECK_OPTION, models.settings.autoCheckForUpdates),
          withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
      includePrerelease = withLayoutData(
          createCheckbox(this, Messages.UPDATE_CHECK_PRERELEASE_OPTION, models.settings.includePrereleases),
          withIndents(new GridData(SWT.LEFT, SWT.TOP, false, false), 20, 0));
      Label adbWarning = withLayoutData(createLabel(this, ""),
          withSpans(new GridData(SWT.FILL, SWT.FILL, true, false), 2, 1));
      adbWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_RED));
      withLayoutData(
          createLink(this, Messages.PRIVACY_POLICY, WelcomeDialog::showPolicy),
          withIndents(
              withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1), 0, 20));

      Listener adbListener = e -> {
        String error = Settings.checkAdbIsValid(adbPath.getText().trim());
        if (error == null) {
          adbWarning.setVisible(false);
        } else {
          adbWarning.setText("Path to adb is invalid. Please fix for Android support: " + error);
          adbWarning.setVisible(true);
        }
      };
      adbPath.addBoxListener(SWT.Modify, adbListener);
      adbListener.handleEvent(null);

      allowUpdateChecks.addListener(SWT.Selection, e -> {
        if (!allowUpdateChecks.getSelection()) {
          includePrerelease.setSelection(false);
        }
        includePrerelease.setEnabled(allowUpdateChecks.getSelection());
      });
    }

    protected void beforeAnalytics() {
      /* Do nothing */
    }

    public void save() {
      models.settings.adb = adbPath.getText().trim();
      models.settings.setAnalyticsEnabled(allowAnalytics.getSelection());
      models.settings.reportCrashes = allowCrashReports.getSelection();
      models.settings.autoCheckForUpdates = allowUpdateChecks.getSelection();
      models.settings.includePrereleases = includePrerelease.getSelection();
      // When settings are saved, reset the update timer, to force update check on next start
      models.settings.lastCheckForUpdates = 0;
      models.settings.save();
      models.analytics.updateSettings();
    }
  }
}
