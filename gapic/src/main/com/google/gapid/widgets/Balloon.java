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

import static com.google.gapid.widgets.Widgets.withMargin;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Shell;

import java.util.function.Consumer;

/**
 * A popup window without a trim similar to tooltips, but with intractable components.
 */
public class Balloon {
  private final Shell shell;
  private final Listener listener;

  private Balloon(Shell shell) {
    this.shell = shell;
    this.listener = e -> {
      if (!(e.widget instanceof Control )|| ((Control)e.widget).getShell() != shell) {
        // Not our event, close the balloon.
        close();
      }
    };
    shell.getDisplay().addFilter(SWT.MouseDown, listener);
  }

  public static Balloon createAndShow(
      Control parent, Consumer<Shell> createContents, Point offset) {
    return createAndShow(parent, createContents, offset, SWT.DEFAULT, SWT.DEFAULT);
  }

  public static Balloon createAndShow(
      Control parent, Consumer<Shell> createContents, Point offset, int wHint, int hHint) {
    Shell parentShell = parent.getShell();
    Shell shell = new Shell(parentShell, SWT.ON_TOP | SWT.NO_TRIM | SWT.NO_FOCUS);
    shell.setLayout(withMargin(new FillLayout(), 5, 5));
    createContents.accept(shell);
    shell.setLocation(parent.toDisplay(offset));
    shell.setSize(shell.computeSize(wHint, hHint));
    shell.setVisible(true);
    return new Balloon(shell);
  }

  public void close() {
    if (!shell.isDisposed()) {
      shell.getDisplay().removeFilter(SWT.MouseDown, listener);
      shell.close();
      shell.dispose();
    }
  }
}
