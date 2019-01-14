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
package com.google.gapid;

import static com.google.gapid.widgets.Widgets.createLabel;

import com.google.gapid.util.Messages;
import com.google.gapid.widgets.CenteringLayout;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Label;

/**
 * The loading screen is a minimal view shown while the UI is loading, looking for gapis, etc.
 */
public class LoadingScreen extends Composite {
  private final Label statusLabel;

  public LoadingScreen(Composite parent, Theme theme) {
    super(parent, SWT.NONE);
    setLayout(CenteringLayout.goldenRatio());

    Composite container = Widgets.createComposite(this, new GridLayout(1, false));
    createLabel(container, "", theme.dialogLogo())
        .setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

    Label titleLabel = createLabel(container, Messages.WINDOW_TITLE);
    titleLabel.setFont(theme.bigBoldFont());
    titleLabel.setLayoutData(new GridData(SWT.CENTER, SWT.TOP, true, false));

    statusLabel = createLabel(container, "Starting up...");
    statusLabel.setLayoutData(new GridData(SWT.LEFT, SWT.TOP, true, false));
  }

  public void setText(String status) {
    statusLabel.setText(status);
    statusLabel.requestLayout();
  }
}
