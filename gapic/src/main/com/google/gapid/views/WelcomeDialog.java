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

import static com.google.gapid.util.GapidVersion.GAPID_VERSION;
import static com.google.gapid.util.GeoUtils.bottomLeft;
import static com.google.gapid.views.AboutDialog.showHelp;
import static com.google.gapid.views.TracerDialog.showOpenTraceDialog;
import static com.google.gapid.views.TracerDialog.showTracingDialog;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createMenuItem;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withLayoutData;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.server.Client;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.program.Program;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Shell;

import java.io.File;
import java.util.function.Consumer;

/**
 * Welcome dialog shown when the application is run without a capture as an argument.
 */
public class WelcomeDialog {
  private static final String API_TOS_URL = "https://developers.google.com/terms/";
  private static final String PRIVACY_POLICY_URL = "https://www.google.com/policies/privacy/";

  private WelcomeDialog() {
  }

  public static void showPolicy(Event evt) {
    if ("TOS".equals(evt.text)) {
      showApiTermsOfService();
    } else {
      showPrivacyPolicy();
    }
  }

  public static void showApiTermsOfService() {
    Program.launch(API_TOS_URL);
  }

  public static void showPrivacyPolicy() {
    Program.launch(PRIVACY_POLICY_URL);
  }

  public static void showFirstTimeDialog(
      Shell shell, Models models, Widgets widgets, Runnable next) {
    new WelcomeDialogBase(shell, widgets.theme) {
      private SettingsDialog.SettingsFormBase form;

      @Override
      protected Control createDialogArea(Composite parent) {
        return createDialogArea(Messages.WELCOME_SUBTITLE, super.createDialogArea(parent), c -> {
          createLabel(c, Messages.WELCOME_TEXT)
              .setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

          form = withLayoutData(new SettingsDialog.SettingsFormBase(models, c, 0, 10),
              new GridData(SWT.FILL, SWT.FILL, true, true));
         });
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        createButton(parent, IDialogConstants.OK_ID, Messages.WELCOME_BUTTON, true);
      }

      @Override
      protected void okPressed() {
        models.settings.skipFirstRunDialog = true;
        form.save();

        super.okPressed();
        next.run();
      }
    }.open();
  }

  public static void showWelcomeDialog(Client client, Shell shell, Models models, Widgets widgets) {
    models.analytics.postInteraction(View.Welcome, ClientAction.Show);
    new WelcomeDialogBase(shell, widgets.theme) {
      private Button showWelcome;

      @Override
      protected Control createDialogArea(Composite parent) {
        return createDialogArea(Messages.WINDOW_TITLE, super.createDialogArea(parent), c -> {
          Label version = createLabel(c, "Version " + GAPID_VERSION.toFriendlyString());
          version.setForeground(widgets.theme.welcomeVersionColor());
          version.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

          createLink(c, "<a>Open Trace...</a>", e -> {
            close(true);
            showOpenTraceDialog(shell, models);
          });
          String[] files = models.settings.getRecent();
          if (files.length > 0) {
            createLink(c, "<a>Open Recent...</a>", e -> {
              Menu popup = new Menu(c);
              for (String file : models.settings.recentFiles) {
                createMenuItem(popup, file, 0, ev -> {
                  models.analytics.postInteraction(View.Welcome, ClientAction.OpenRecent);
                  close(true);
                  models.capture.loadCapture(new File(file));
                });
              }
              popup.addListener(SWT.Hide, ev -> scheduleIfNotDisposed(popup, popup::dispose));

              popup.setLocation(c.toDisplay(bottomLeft(((Link)e.widget).getBounds())));
              popup.setVisible(true);
            });
          }
          createLink(c, "<a>Capture Trace...</a>", e -> {
            close(true);
            showTracingDialog(client, shell, models, widgets);
          });
          createLink(c, "<a>Help...</a>", e -> showHelp(models.analytics));

          showWelcome = createCheckbox(c, "Show on startup", !models.settings.skipWelcomeScreen);
        });
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        createButton(parent, IDialogConstants.CLOSE_ID, IDialogConstants.CLOSE_LABEL, true);
      }

      @Override
      protected void buttonPressed(int buttonId) {
        close(buttonId == IDialogConstants.CLOSE_ID);
      }

      private void close(boolean saveState) {
        if (saveState) {
          models.settings.skipWelcomeScreen = !showWelcome.getSelection();
        }
        close();
      }
    }.open();
  }

  private static class WelcomeDialogBase extends DialogBase {
    public WelcomeDialogBase(Shell shell, Theme theme) {
      super(shell, theme);
    }

    @Override
    public String getTitle() {
      return Messages.WELCOME_TITLE;
    }

    protected Control createDialogArea(String title, Control area, Consumer<Composite> create) {
      Composite container = createComposite((Composite)area, new GridLayout(1, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      createLabel(container, "", theme.dialogLogo())
          .setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

      Label titleLabel = createLabel(container, title);
      titleLabel.setFont(theme.bigBoldFont());
      titleLabel.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

      create.accept(container);

      return area;
    }
  }
}
