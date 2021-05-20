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
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.util.Experimental;
import com.google.gapid.views.CommandTree.Tree;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.MenuItem;

import java.util.List;
import java.util.stream.Collectors;

public class CommandOptions {
  private CommandOptions() {
  }

  public static void CreateCommandOptionsMenu(Control parent, Widgets widgets, Tree tree, Models models) {
    final Menu optionsMenu = new Menu(parent);

    MenuItem editMenuItem = Widgets.createMenuItem(optionsMenu , "&Edit", SWT.MOD1 + 'E', e -> {
      CommandStream.Node node = tree.getSelection();
      if (node != null && node.getData() != null && node.getCommand() != null) {
        widgets.editor.showEditPopup(optionsMenu.getShell(), lastCommand(node.getData().getCommands()),
            node.getCommand(), node.device);
      }
    });

    MenuItem disableMenuItem;
    MenuItem enableMenuItem;
    MenuItem isolateMenuItem;
    if (Experimental.enableProfileExperiments(models.settings)) {
      disableMenuItem = Widgets.createMenuItem(optionsMenu, "Disable Drawcall", SWT.MOD1 + 'D', e -> {
        CommandStream.Node node = tree.getSelection();
        if (node != null && node.getData() != null) {
          widgets.experiments.disableCommands(node.getData().getExperimentalCommandsList());
        }
        tree.updateTree(tree.getSelectionItem());
      });

      enableMenuItem = Widgets.createMenuItem(optionsMenu, "Enable Drawcall", SWT.MOD1 + 'E', e -> {
        CommandStream.Node node = tree.getSelection();
        if (node != null && node.getData() != null) {
          widgets.experiments.enableCommands(node.getData().getExperimentalCommandsList());
        }
        tree.updateTree(tree.getSelectionItem());
      });

      isolateMenuItem = Widgets.createMenuItem(optionsMenu, "Disable Other Drawcalls", SWT.MOD1 + 'I', e -> {
        CommandStream.Node node = tree.getSelection();
        if (node != null && node.getData() != null) {
          widgets.experiments.disableCommands(getSiblings(node));
        }
        tree.updateTree(tree.getSelectionItem());
      });
    } else {
      disableMenuItem = null;
      enableMenuItem = null;
      isolateMenuItem = null;
    }

    tree.setPopupMenu(optionsMenu, node -> {
      if (node.getData() == null) {
        return false;
      }

      editMenuItem.setEnabled(false);
      if (node.getCommand() != null && CommandEditor.shouldShowEditPopup(node.getCommand())) {
        editMenuItem.setEnabled(true);
      }

      boolean canBeDisabled = node.getData().getExperimentalCommandsCount() > 0;
      boolean canBeIsolated = node.getParent().getData().getExperimentalCommandsCount() > 1;

      if (disableMenuItem != null) {
        boolean disabled = widgets.experiments.areAllCommandsDisabled(
            node.getData().getExperimentalCommandsList());
        disableMenuItem.setEnabled(canBeDisabled && !disabled);
      }

      if (enableMenuItem != null) {
        boolean hasDisabledChildren = widgets.experiments.isAnyCommandDisabled(
            node.getData().getExperimentalCommandsList());
        enableMenuItem.setEnabled(canBeDisabled && hasDisabledChildren);
      }

      if (isolateMenuItem != null) {
        isolateMenuItem.setEnabled(canBeDisabled && canBeIsolated);
      }
      return true;
    });
  }

  private static List<Path.Command> getSiblings(CommandStream.Node node) {
    List<Path.Command> experimentalCommands = node.getData().getExperimentalCommandsList();
    return node.getParent().getData().getExperimentalCommandsList()
      .stream()
      .filter(cmd -> !experimentalCommands.contains(cmd))
      .collect(Collectors.toList());
  }
}
