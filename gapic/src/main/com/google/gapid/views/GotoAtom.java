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

import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSpinner;

import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.proto.service.path.Path;
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
import org.eclipse.swt.widgets.Spinner;

/**
 * Dialog for the goto command (atom) action.
 */
public class GotoAtom {
  private GotoAtom() {
  }

  public static void showGotoAtomDialog(Shell shell, Path.Capture capture, AtomStream atoms) {
    GotoDialog dialog = new GotoDialog(shell, atoms);
    if (dialog.open() == Window.OK) {
      atoms.selectAtoms(AtomIndex.forCommand(Path.Command.newBuilder()
          .addIndices(dialog.value)
          .setCapture(capture)
          .build()), true);
    }
  }

  /**
   * Dialog asking the user for the ID of the command to jump to.
   */
  private static class GotoDialog extends MessageDialog {
    private final AtomStream atoms;
    private Spinner spinner;
    public int value;

    public GotoDialog(Shell shell, AtomStream atoms) {
      super(shell, Messages.GOTO, null, Messages.GOTO_ATOM, MessageDialog.CONFIRM, 0,
          IDialogConstants.OK_LABEL, IDialogConstants.CANCEL_LABEL);
      this.atoms = atoms;
    }

    @Override
    protected boolean isResizable() {
      return true;
    }

    @Override
    protected Control createCustomArea(Composite parent) {
      // Although the atom ID is a long, we currently only actually support the int range, as
      // the atoms are stored in an array. So, using an int spinner here is fine.
      //TODO limit to max atoms
      int max = Integer.MAX_VALUE;
      AtomIndex selection = atoms.getSelectedAtoms();
      int current = (selection == null) ? 0 : 0 /*TODO(int)last(selection)*/;

      Composite container = createComposite(parent, new GridLayout(2, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
      createLabel(container, Messages.ATOM_ID + ":");
      spinner = createSpinner(container, current, 0, max);
      spinner.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

      return container;
    }

    @Override
    protected void buttonPressed(int buttonId) {
      // The spinner gets disposed after this.
      value = spinner.getSelection();
      super.buttonPressed(buttonId);
    }
  }
}
