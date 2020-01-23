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
import static com.google.gapid.widgets.Widgets.createTextbox;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.ClientAction;
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
import org.eclipse.swt.widgets.Text;
import java.util.ArrayList;
import java.util.List;

/**
 * Dialog for the goto command action.
 */
public class GotoCommand {
  private GotoCommand() {
  }

  public static void showGotoCommandDialog(Shell shell, Models models) {
    models.analytics.postInteraction(View.GotoCommand, ClientAction.Show);
    GotoDialog dialog = new GotoDialog(shell, models.commands);
    if (dialog.open() == Window.OK) {
      models.commands.selectCommands(CommandIndex.forCommand(Path.Command.newBuilder()
          .addAllIndices(dialog.value)
          .setCapture(models.capture.getData().path)
          .build()), true);
    }
  }

  /**
   * Dialog asking the user for the ID of the command to jump to.
   */
  private static class GotoDialog extends MessageDialog {
    private final CommandStream commands;
    private Text text;
    public List<Long> value;

    public GotoDialog(Shell shell, CommandStream commands) {
      super(shell, Messages.GOTO, null, Messages.GOTO_COMMAND, MessageDialog.CONFIRM, 0,
          IDialogConstants.OK_LABEL, IDialogConstants.CANCEL_LABEL);
      this.commands = commands;
    }

    @Override
    protected boolean isResizable() {
      return true;
    }

    @Override
    protected Control createCustomArea(Composite parent) {
      Composite container = createComposite(parent, new GridLayout(2, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
      createLabel(container, Messages.COMMAND_ID + ":");
      text = createTextbox(container, "");
      text.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

      return container;
    }

    @Override
    protected void buttonPressed(int buttonId) {
      String[] strings = text.getText().split("\\.");
      value = new ArrayList<Long>();
      for (String s : strings) {
        value.add(Long.parseLong(s));
      }
      super.buttonPressed(buttonId);
    }
  }
}
