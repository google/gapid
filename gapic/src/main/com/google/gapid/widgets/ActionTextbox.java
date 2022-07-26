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
package com.google.gapid.widgets;

import static com.google.gapid.widgets.Widgets.createTextbox;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Text;

/**
 * A textbox-button combo, where clicking the button shows a dialog that will provide a value
 * for the textbox on closing.
 */
public abstract class ActionTextbox extends Composite {
  private final Text box;
  private final Button button;

  public ActionTextbox(Composite parent, String value) {
    this(parent, "...", value);
  }

  public ActionTextbox(Composite parent, String actionLabel, String value) {
    super(parent, SWT.NONE);

    setLayout(Widgets.withMarginOnly(new GridLayout(2, false), 0, 0));
    box = createTextbox(this, value);
    box.setLayoutData(new GridData(SWT.FILL, SWT.CENTER, true, false));

    button = Widgets.createButton(this, actionLabel, e -> showDialog());
  }

  private void showDialog() {
    String result = createAndShowDialog(box.getText());
    if (result != null) {
      box.setText(result);
    }
  }

  protected abstract String createAndShowDialog(String current);

  public String getText() {
    return box.getText();
  }

  public void setText(String text) {
    box.setText(text);
  }

  public void addBoxListener(int eventType, Listener listener) {
    box.addListener(eventType, listener);
  }

  public void setActionEnabled(boolean enabled) {
    button.setEnabled(enabled);
  }
}
