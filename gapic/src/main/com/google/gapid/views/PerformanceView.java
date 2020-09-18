/*
 * Copyright (C) 2020 Google Inc.
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

import static com.google.gapid.util.Loadable.MessageType.Error;

import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.proto.service.Service;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;
import java.util.logging.Logger;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Menu;

public class PerformanceView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Profile.Listener {
  protected static final Logger LOG = Logger.getLogger(PerformanceView.class.getName());

  private final Models models;
  private final LoadablePanel<PerfTree> loading;
  protected final PerfTree tree;

  public PerformanceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new GridLayout(1, false));
    loading = LoadablePanel.create(this, widgets, p -> new PerfTree(p, models, widgets));
    tree = loading.getContents();

    Menu popup = new Menu(tree.getControl());
    Widgets.createMenuItem(popup, "Select in Command Tab", e -> {
      CommandStream.Node node = tree.getSelection();
      if (node != null && node.getIndex() != null && models.resources.isLoaded()) {
        models.commands.selectCommands(node.getIndex(), true);
      }
    });
    tree.setPopupMenu(popup, node -> node != null && node.getIndex() != null && models.resources.isLoaded());

    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.profile.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.profile.removeListener(this);
    });
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    updateTree(false);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    updateTree(true);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onCommandsLoaded() {
    updateTree(false);
  }

  @Override
  public void onProfileLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(error);
      return;
    }
    // Create columns for counters after profile get loaded, because we need to know counter numbers.
    for (Service.ProfilingData.Counter counter : models.profile.getData().getCounters()) {
      tree.addColumnForCounter(counter);
    }
    tree.packColumn();
    tree.refresh();
  }

  private void updateTree(boolean assumeLoading) {
    if (assumeLoading || !models.commands.isLoaded()) {
      loading.startLoading();
      tree.setInput(null);
      return;
    }

    loading.stopLoading();
    tree.setInput(models.commands.getData());
  }

  private static class PerfTree extends CommandTree.Tree {
    private static final int DURATION_WIDTH = 95;

    public PerfTree(Composite parent, Models models, Widgets widgets) {
      super(parent, models, widgets);
    }

    @Override
    protected boolean shouldShowImage(CommandStream.Node node) {
      return false;
    }

    private void addColumnForCounter(Service.ProfilingData.Counter counter) {
      TreeViewerColumn column = addColumn(counter.getName(), node -> {
        Service.CommandTreeNode data = node.getData();
        if (data == null) {
          return "";
        } else if (!models.profile.isLoaded()) {
          return "Profiling...";
        } else {
          Double aggregation = models.profile.getData().getCounterAggregation(data.getCommands(), counter);
          return aggregation.isNaN() ? "" : String.format("%.3f", aggregation);
        }
      }, DURATION_WIDTH);
      column.getColumn().setAlignment(SWT.RIGHT);
    }

    public void refresh() {
      refresher.refresh();
    }
  }
}
