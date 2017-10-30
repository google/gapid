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

import org.eclipse.jface.dialogs.TrayDialog;
import org.eclipse.jface.window.IShellProvider;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.widgets.Shell;

/**
 * Base class for GAPID dialogs.
 */
public abstract class DialogBase extends TrayDialog {
  private static final int MIN_DIALOG_WIDTH = 350;  // In DLUs
  private static final int MIN_DIALOG_HEIGHT = 150; // In DLUs

  private final Theme theme;

  public DialogBase(IShellProvider parentShell, Theme theme) {
    super(parentShell);
    this.theme = theme;
    setReturnCode(Window.CANCEL); // Switch the default return code to cancel.
  }

  public DialogBase(Shell shell, Theme theme) {
    super(shell);
    this.theme = theme;
    setReturnCode(Window.CANCEL); // Switch the default return code to cancel.
  }

  @Override
  protected boolean isResizable() {
    return true;
  }

  @Override
  protected void configureShell(Shell newShell) {
    super.configureShell(newShell);
    newShell.setText(getTitle());
    newShell.setImages(theme.windowLogo());
  }

  public abstract String getTitle();

  @Override
  protected Point getInitialSize() {
    Point size = super.getInitialSize();
    return new Point(Math.max(convertHorizontalDLUsToPixels(MIN_DIALOG_WIDTH), size.x),
        Math.max(convertVerticalDLUsToPixels(MIN_DIALOG_HEIGHT), size.y));
  }
}
