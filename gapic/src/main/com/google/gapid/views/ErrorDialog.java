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

import static com.google.common.base.Throwables.getStackTraceAsString;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withSizeHints;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.common.escape.Escaper;
import com.google.common.net.UrlEscapers;
import com.google.gapid.util.Messages;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.dialogs.IconAndMessageDialog;
import org.eclipse.jface.layout.GridDataFactory;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.program.Program;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.ExpandBar;
import org.eclipse.swt.widgets.ExpandItem;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Shell;

/**
 * Shows a dialog to the user for errors where there are no other alternatives.
 */
public class ErrorDialog {
  private static final int MAX_DETAILS_SIZE = 300;
  private static final int DETAILS_STYLE =
      SWT.MULTI | SWT.BORDER | SWT.H_SCROLL | SWT.V_SCROLL | SWT.READ_ONLY;
  private static final String FILE_BUG_URL =
      "https://github.com/google/gapid/issues/new?title=%s&body=%s&labels=user";

  public static void showErrorDialog(Shell shell, String text, Throwable exception) {
    showErrorDialog(shell, text, getStackTraceAsString(exception));
  }

  public static void showErrorDialog(Shell shell, String text, String detailString) {
    new IconAndMessageDialog(shell) {
      private Group details;

      @Override
      protected void configureShell(Shell newShell) {
        super.configureShell(newShell);
        newShell.setText("Error");
      }

      @Override
      protected boolean isResizable() {
        return true;
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite container = (Composite)super.createDialogArea(parent);
        ((GridLayout)container.getLayout()).numColumns = 2;
        createMessageArea(container);

        String msg = String.format(Messages.ERROR_MESSAGE, text);
        withLayoutData(createTextbox(container, SWT.WRAP | SWT.READ_ONLY, msg),
            withSizeHints(
                new GridData(SWT.FILL, SWT.CENTER, true, false), getWidthHint(), SWT.DEFAULT))
          .setBackground(container.getBackground());

        if (detailString != null) {
          ExpandBar bar = withLayoutData(new ExpandBar(container, SWT.NONE),
              withSpans(new GridData(SWT.FILL, SWT.TOP, true, false), 2, 1));
          new ExpandItem(bar, SWT.NONE, 0).setText("Details...");

          bar.addListener(SWT.Expand, e -> {
            createDetails(container);
            Point curr = getShell().getSize();
            Point want = getShell().computeSize(SWT.DEFAULT, SWT.DEFAULT);
            if (want.y > curr.y) {
              getShell().setSize(
                  new Point(curr.x, curr.y + Math.min(MAX_DETAILS_SIZE, want.y - curr.y)));
            } else {
              details.requestLayout();
            }
          });

          bar.addListener(SWT.Collapse, e -> {
            Point curr = getShell().getSize();
            if (details != null) {
              details.dispose();
              details = null;
            }
            Point want = getShell().computeSize(SWT.DEFAULT, SWT.DEFAULT);
            if (want.y < curr.y) {
              getShell().setSize(new Point(curr.x, want.y));
            }
          });
        }

        return container;
      }

      private int getWidthHint() {
        return convertHorizontalDLUsToPixels(IDialogConstants.MINIMUM_MESSAGE_AREA_WIDTH);
      }

      private void createDetails(Composite container) {
        details = createGroup(container, "");
        GridDataFactory.fillDefaults().grab(true, true).span(2, 1).applyTo(details);
        Composite inner = createComposite(details, new GridLayout(1, false));
        withLayoutData(createTextbox(inner, DETAILS_STYLE, detailString),
            new GridData(SWT.FILL, SWT.FILL, true, true));
        withLayoutData(createLink(inner, "<a>File a bug</a> on GitHub", e -> {
          Program.launch(getFileBugUrl());
        }), new GridData(SWT.RIGHT, SWT.BOTTOM, false, false));
        withLayoutData(createLink(inner, "<a>Show logs</a> directory", e -> {
          AboutDialog.showLogDir();
        }), new GridData(SWT.RIGHT, SWT.BOTTOM, false, false));
      }

      private String getFileBugUrl() {
        Escaper esc = UrlEscapers.urlFormParameterEscaper();
        return String.format(FILE_BUG_URL, esc.escape(text), esc.escape(Messages.BUG_BODY));
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true).setFocus();
      }

      @Override
      protected Image getImage() {
        return getErrorImage();
      }
    }.open();
  }
}
