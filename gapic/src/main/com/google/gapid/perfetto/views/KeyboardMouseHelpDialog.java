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

import com.google.common.io.Resources;
import static java.util.logging.Level.SEVERE;
import static java.nio.charset.StandardCharsets.UTF_8;

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
import org.eclipse.swt.SWTError;
import org.eclipse.swt.browser.Browser;
import org.eclipse.swt.browser.LocationAdapter;
import org.eclipse.swt.browser.LocationEvent;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Text;

import java.io.IOException;
import java.util.logging.Logger;

public class KeyboardMouseHelpDialog {

  private static final Logger LOG = Logger.getLogger(KeyboardMouseHelpDialog.class.getName());

  public static void showHelp(Shell shell, Analytics analytics, Widgets widgets) {
    analytics.postInteraction(View.About, ClientAction.Show);
    new DialogBase(shell, widgets.theme) {
      @Override
      public String getTitle() {
        return Messages.KEYBOARD_MOUSE_HELP_TITLE;
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = (Composite)super.createDialogArea(parent);

        Browser browser;
        try {
          browser = new Browser(area, SWT.NONE);
        } catch (SWTError e) {
          // Failed to initialize the browser. Show it as a plain text widget.
          Text text = new Text(
              area, SWT.MULTI | SWT.READ_ONLY | SWT.BORDER | SWT.H_SCROLL | SWT.V_SCROLL);
          text.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
          text.setText(readKeyboardMouseHelp(false));
          return text;
        }

        GridData data = new GridData(SWT.FILL, SWT.FILL, true, true);
        data.widthHint = 850;
        data.heightHint = 650;
        browser.setLayoutData(data);
        browser.setText(readKeyboardMouseHelp(true));
        browser.addLocationListener(new LocationAdapter() {
          @Override
          public void changing(LocationEvent event) {
            if ("about:blank".equals(event.location)) {
              browser.setText(readKeyboardMouseHelp(true));
            }
          }
        });
        return area;
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
      }

    }.open();
  }

  protected static String readKeyboardMouseHelp(boolean html) {
    try {
      String result = Resources.toString(Resources.getResource("text/keyboard-mouse-help.html"), UTF_8);
      return result;
    } catch (IOException | IllegalArgumentException e) {
      LOG.log(SEVERE, "Failed to load help.", e);
      return "Failed to load help.";
    }
  }
}