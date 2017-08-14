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

import static com.google.gapid.util.Paths.memoryAfter;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createTextbox;

import com.google.gapid.models.Models;
import com.google.gapid.util.Messages;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.dialogs.MessageDialog;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Text;

/**
 * Dialog for the goto memory location action.
 */
public class GotoMemory {
  private GotoMemory() {
  }

  public static void showGotoMemoryDialog(Shell shell, Models models) {
    GotoDialog dialog = new GotoDialog(shell);
    if (dialog.open() == Window.OK) {
      long address;
      int pool;
      try {
        if (dialog.addressValue.startsWith("0x")) {
          address = Long.parseLong(dialog.addressValue.substring(2), 16);
        } else {
          address = Long.parseLong(dialog.addressValue, 10);
        }

        pool = Integer.parseInt(dialog.poolValue, 10);
      } catch (NumberFormatException e) {
        // TODO
        return;
      }
      models.follower.gotoMemory(memoryAfter(
          models.atoms.getSelectedAtoms().getCommand(), pool, address, 0).getMemory());
    }
  }

  /**
   * Dialog asking the user for an address and memory pool to jump to.
   */
  private static class GotoDialog extends MessageDialog {
    private Text addressText, poolText;
    public String addressValue, poolValue;

    public GotoDialog(Shell shell) {
      super(shell, Messages.GOTO, null, Messages.GOTO_MEMORY, MessageDialog.CONFIRM, 0,
          IDialogConstants.OK_LABEL, IDialogConstants.CANCEL_LABEL);
    }

    @Override
    protected boolean isResizable() {
      return true;
    }

    @Override
    protected Control createCustomArea(Composite parent) {
      Composite container = createComposite(parent, new GridLayout(2, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      createLabel(container, Messages.MEMORY_ADDRESS + ":");
      addressText = createTextbox(container, ""); // TODO: default to current
      addressText.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

      createLabel(container, Messages.MEMORY_POOL + ":");
      poolText = createTextbox(container, "0");  // TODO: default to current
      poolText.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

      return container;
    }

    @Override
    protected void buttonPressed(int buttonId) {
      // The textboxes get disposed after this.
      addressValue = addressText.getText();
      poolValue = poolText.getText();
      super.buttonPressed(buttonId);
    }
  }
}
