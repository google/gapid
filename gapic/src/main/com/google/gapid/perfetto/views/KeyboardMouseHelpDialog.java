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
package com.google.gapid.perfetto.views;

import static com.google.gapid.widgets.Widgets.withSizeHints;
import static java.nio.charset.StandardCharsets.UTF_8;
import static java.util.logging.Level.SEVERE;

import com.google.common.io.Resources;
import com.google.gapid.models.Analytics;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;

import java.io.IOException;
import java.util.logging.Logger;

/**
 * Dialog showing keyboard and mouse help for the trace view.
 */
public class KeyboardMouseHelpDialog {

  private static final Logger LOG = Logger.getLogger(KeyboardMouseHelpDialog.class.getName());

  public static void showHelp(Shell shell, Analytics analytics, Theme theme) {
    analytics.postInteraction(View.Help, ClientAction.Show);
    new DialogBase(shell, theme) {
      @Override
      public String getTitle() {
        return Messages.KEYBOARD_MOUSE_HELP_TITLE;
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = (Composite)super.createDialogArea(parent);
        Control browser = Widgets.createBrowser(area, readKeyboardMouseHelp());
        browser.setLayoutData(
            withSizeHints(new GridData(SWT.FILL, SWT.FILL, true, true), 1024, 768));
        return area;
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
      }

    }.open();
  }

  protected static String readKeyboardMouseHelp() {
    try {
      return Resources.toString(Resources.getResource("text/keyboard-mouse-help.html"), UTF_8);
    } catch (IOException | IllegalArgumentException e) {
      LOG.log(SEVERE, "Failed to load help.", e);
      return "Failed to load help.";
    }
  }
}
