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
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSelectableLabel;
import static com.google.gapid.widgets.Widgets.withMargin;
import static java.util.logging.Level.SEVERE;

import com.google.gapid.models.Analytics;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Info;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.util.Logging;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StyledText;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.program.Program;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;

import java.io.IOException;
import java.util.logging.Logger;

/**
 * Dialog showing some basic info about our application.
 */
public class AboutDialog {
  private static final String HELP_URL = "https://gpuinspector.dev";
  private static final Logger LOG = Logger.getLogger(AboutDialog.class.getName());

  private AboutDialog() {
  }

  public static void showHelp(Analytics analytics) {
    analytics.postInteraction(View.Main, ClientAction.ShowHelp);
    Program.launch(HELP_URL);
  }

  public static void showLogDir(Analytics analytics) {
    analytics.postInteraction(View.Main, ClientAction.ShowLogDir);
    try {
      OS.openFileInSystemExplorer(Logging.getLogDir());
    } catch (IOException e) {
      LOG.log(SEVERE, "Failed to open log directory in system explorer", e);
    }
  }

  public static void showAbout(Shell shell, Analytics analytics, Widgets widgets) {
    analytics.postInteraction(View.About, ClientAction.Show);
    new DialogBase(shell, widgets.theme) {
      @Override
      public String getTitle() {
        return Messages.ABOUT_TITLE;
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = (Composite)super.createDialogArea(parent);

        Composite container = createComposite(area, withMargin(new GridLayout(2, false), 20, 5));
        container.setLayoutData(new GridData(SWT.CENTER, SWT.FILL, true, false));

        Label logo = createLabel(container, "", theme.dialogLogo());
        logo.setLayoutData(new GridData(SWT.CENTER, SWT.FILL, true, false, 2, 1));

        StyledText title = createSelectableLabel(container, Messages.WINDOW_TITLE);
        title.setFont(theme.bigBoldFont());
        title.setLayoutData(new GridData(SWT.CENTER, SWT.FILL, true, false, 2, 1));

        createLabel(container, "").setLayoutData(new GridData(SWT.CENTER, SWT.FILL, true, true, 2, 1));
        Button clipboard = Widgets.createButton(container, "", e -> {
          String textData = "Version " + GAPID_VERSION;
          widgets.copypaste.setContents(textData);
        });

        clipboard.setImage(theme.clipboard());
        clipboard.setLayoutData(new GridData(SWT.CENTER, SWT.BEGINNING, true, true, 1, 3));

        createSelectableLabel(container, "Version " + GAPID_VERSION);
        createSelectableLabel(
            container, "Server: " + Info.getServerName() + ", Version: " + Info.getServerVersion());
        createSelectableLabel(container, Messages.ABOUT_COPY);

        return area;
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
      }
    }.open();
  }
}
