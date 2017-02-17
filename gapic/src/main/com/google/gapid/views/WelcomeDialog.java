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

import static com.google.gapid.views.TracerDialog.showOpenTraceDialog;
import static com.google.gapid.views.TracerDialog.showTracingDialog;
import static com.google.gapid.widgets.AboutDialog.showHelp;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;

import com.google.gapid.models.Models;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.dialogs.TitleAreaDialog;
import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;

public class WelcomeDialog {
  private WelcomeDialog() {
  }

  public static void showWelcomeDialog(Shell shell, Models models, Widgets widgets) {
    new TitleAreaDialog(shell) {
      private Button showWelcome;

      @Override
      public void create() {
        super.create();
        setTitle(Messages.WELCOME_TITLE);
      }

      @Override
      protected void configureShell(Shell newShell) {
        super.configureShell(newShell);
        newShell.setText(Messages.WELCOME_WINDOW_TITLE);
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = (Composite)super.createDialogArea(parent);

        Composite container = createComposite(area, new GridLayout(1, false));
        container.setLayoutData(new GridData(SWT.CENTER, SWT.FILL, true, true));

        createLabel(container, "", widgets.theme.logoBig())
            .setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

        Label title = createLabel(container, Messages.WINDOW_TITLE);
        title.setFont(JFaceResources.getFontRegistry().getBold(JFaceResources.DEFAULT_FONT));
        title.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

        Widgets.createLink(container, "<a>Open Trace...</a>", e -> {
          close(true);
          showOpenTraceDialog(shell, models);
        });
        Widgets.createLink(container, "<a>Capture Trace...</a>", e -> {
          close(true);
          showTracingDialog(shell, models, widgets);
        });
        Widgets.createLink(container, "<a>Help...</a>", e -> showHelp());

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
