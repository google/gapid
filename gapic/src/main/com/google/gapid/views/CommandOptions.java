/*
 * Copyright (C) 2021 Google Inc.
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

import static com.google.gapid.util.Paths.lastCommand;

import com.google.gapid.models.CommandStream;
import com.google.gapid.models.Models;
import com.google.gapid.util.Experimental;
import com.google.gapid.views.CommandTree.Tree;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.MenuItem;

public class CommandOptions {
  private CommandOptions() {
  }

  public static void CreateCommandOptionsMenu(Control parent, Widgets widgets, Tree tree, Models models) {
    final Menu optionsMenu = new Menu(parent);
    final Menu experimentsMenu = new Menu(optionsMenu);

    MenuItem editMenuItem = Widgets.createMenuItem(optionsMenu , "&Edit", SWT.MOD1 + 'E', e -> {
      CommandStream.Node node = tree.getSelection();
      if (node != null && node.getData() != null && node.getCommand() != null) {
        widgets.editor.showEditPopup(optionsMenu.getShell(), lastCommand(node.getData().getCommands()),
            node.getCommand(), node.device);
      }
    });

    if (Experimental.enableProfileExperiments(models.settings)) {
      MenuItem experimentMenuItem = Widgets.createMenuItem(optionsMenu, "Ex&periments", SWT.MOD1 + 'P', e -> {
        CommandStream.Node node = tree.getSelection();
        if (node != null && node.getData() != null) {
          widgets.experiments.showExperimentsPopup(experimentsMenu.getShell());
        }
      });
    }

    tree.setPopupMenu(optionsMenu, node -> {
      if (node.getData() == null) {
        return false;
      }
      editMenuItem.setEnabled(false);
      if (node.getCommand() != null && CommandEditor.shouldShowEditPopup(node.getCommand())) {
        editMenuItem.setEnabled(true);
      }
      return true;
    });
  }
}
