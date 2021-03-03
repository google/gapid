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
import com.google.gapid.proto.SettingsProto;
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
  protected Button disableFrameLooping;
  protected Button enableAllExperimentalFeatures;

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

    form = new SettingsFormBase(models, area, false) {
      @Override
      protected void beforeAnalytics() {
        disableReplayOptimization = withLayoutData(createCheckbox(this,
            "Disable replay optimization",
            models.settings.preferences().getDisableReplayOptimization()),
            withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
        disableFrameLooping = withLayoutData(createCheckbox(this,
            "Disable frame looping",
            models.settings.preferences().getUseFrameLooping() == false),
            withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
        enableAllExperimentalFeatures = withLayoutData(createCheckbox(this,
            "Enable all unsupported experimental features (requires restart)",
            models.settings.preferences().getEnableAllExperimentalFeatures()),
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
    models.settings.writePreferences().setDisableReplayOptimization(
        disableReplayOptimization.getSelection());
    models.settings.writePreferences().setUseFrameLooping(
        disableFrameLooping.getSelection() == false);
    models.settings.writePreferences().setEnableAllExperimentalFeatures(
        enableAllExperimentalFeatures.getSelection());
  }

  public static class SettingsFormBase extends Composite {
    private final Models models;
    private final FileTextbox adbPath;
    private final Button allowAnalytics;
    private final Button allowCrashReports;
    private final Button allowUpdateChecks;
    private final Button includeDevReleases;

    public SettingsFormBase(Models models, Composite parent, boolean override) {
      this(models, parent, 5, 5, override);
    }

    public SettingsFormBase(
        Models models, Composite parent, int marginWidth, int marginHeight, boolean override) {
      super(parent, SWT.NONE);
      this.models = models;
      setLayout(withMargin(new GridLayout(2, false), marginWidth, marginHeight));

      SettingsProto.PreferencesOrBuilder prefs = models.settings.preferences();
      createLabel(this, "Path to adb:");
      adbPath = withLayoutData(new FileTextbox.File(this, prefs.getAdb()) {
        @Override
        protected void configureDialog(FileDialog dialog) {
          dialog.setText("Path to adb:");
        }
      }, new GridData(SWT.FILL, SWT.FILL, true, false));

      beforeAnalytics();

      allowAnalytics = withLayoutData(
          createCheckbox(this, Messages.ANALYTICS_OPTION,
              override || models.settings.analyticsEnabled()),
          withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
      allowCrashReports = withLayoutData(
          createCheckbox(this, Messages.CRASH_REPORTING_OPTION,
              override || prefs.getReportCrashes()),
          withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
      allowUpdateChecks = withLayoutData(
          createCheckbox(this, Messages.UPDATE_CHECK_OPTION,
              override || prefs.getCheckForUpdates()),
          withSpans(new GridData(SWT.LEFT, SWT.TOP, false, false), 2, 1));
      includeDevReleases = withLayoutData(
          createCheckbox(this, Messages.UPDATE_CHECK_DEV_RELEASE_OPTION,
              prefs.getIncludeDevReleases()),
          withSpans(withIndents(new GridData(SWT.LEFT, SWT.TOP, false, false), 20, 0), 2, 1));
      Label adbWarning = withLayoutData(createLabel(this, ""),
          withSpans(new GridData(SWT.FILL, SWT.FILL, true, false), 2, 1));
      adbWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_RED));
      withLayoutData(
          createLink(this, Messages.PRIVACY_POLICY, WelcomeDialog::showPolicy),
          withSpans(withIndents(new GridData(SWT.LEFT, SWT.TOP, false, false), 0, 20), 2, 1));

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
          includeDevReleases.setSelection(false);
        }
        includeDevReleases.setEnabled(allowUpdateChecks.getSelection());
      });
    }

    protected void beforeAnalytics() {
      /* Do nothing */
    }

    public void save() {
      SettingsProto.Preferences.Builder prefs = models.settings.writePreferences();
      prefs.setAdb(adbPath.getText().trim());
      models.settings.setAnalyticsEnabled(allowAnalytics.getSelection());
      prefs.setReportCrashes(allowCrashReports.getSelection());
      prefs.setCheckForUpdates(allowUpdateChecks.getSelection());
      prefs.setIncludeDevReleases(includeDevReleases.getSelection());
      // When settings are saved, reset the update timer, to force update check on next start
      prefs.setLastCheckForUpdates(0);

      models.settings.save();
      models.analytics.updateSettings();
    }
  }
}
