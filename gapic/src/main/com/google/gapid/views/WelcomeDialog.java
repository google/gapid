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
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createMenuItem;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.gapid.models.Models;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Shell;

import java.io.File;

/**
 * Welcome dialog shown when the application is run without a capture as an argument.
 */
public class WelcomeDialog {
  private WelcomeDialog() {
  }

  public static void showWelcomeDialog(Shell shell, Models models, Widgets widgets) {
    new DialogBase(shell, widgets.theme) {
      private Button showWelcome;

      @Override
      public String getTitle() {
        return Messages.WELCOME_TITLE;
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = (Composite)super.createDialogArea(parent);

        Composite container = createComposite(area, new GridLayout(1, false));
        container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

        createLabel(container, "", widgets.theme.dialogLogo())
            .setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

        Label title = createLabel(container, Messages.WELCOME_TEXT);
        title.setFont(widgets.theme.bigBoldFont());
        title.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

        Label version = createLabel(container, "Version " + GAPID_VERSION.toFriendlyString());
        version.setForeground(widgets.theme.welcomeVersionColor());
        version.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

        createLink(container, "<a>Open Trace...</a>", e -> {
          close(true);
          showOpenTraceDialog(shell, models);
        });
        String[] files = models.settings.getRecent();
        if (files.length > 0) {
          createLink(container, "<a>Open Recent...</a>", e -> {
            Menu popup = new Menu(container);
            for (String file : models.settings.recentFiles) {
              createMenuItem(popup, file, 0, ev -> {
                close(true);
                models.capture.loadCapture(new File(file));
              });
            }
            popup.addListener(SWT.Hide, ev -> scheduleIfNotDisposed(popup, popup::dispose));

            popup.setLocation(container.toDisplay(bottomLeft(((Link)e.widget).getBounds())));
            popup.setVisible(true);
          });
        }
        createLink(container, "<a>Capture Trace...</a>", e -> {
          close(true);
          showTracingDialog(shell, models, widgets);
        });
        createLink(container, "<a>Help...</a>", e -> showHelp());

        showWelcome = Widgets.createCheckbox(
            container, "Show on startup", !models.settings.skipWelcomeScreen);
        return area;
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
}
