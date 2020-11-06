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

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.perfetto.Unit;
import com.google.gapid.perfetto.models.CounterInfo;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;
import java.util.logging.Logger;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Menu;

public class PerformanceView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Profile.Listener {
  protected static final Logger LOG = Logger.getLogger(PerformanceView.class.getName());

  private final Models models;
  private final LoadablePanel<Composite> loading;
  private final Button button;
  protected final PerfTree tree;
  private final SelectionHandler<Control> selectionHandler;

  public PerformanceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new GridLayout(1, false));
    loading = LoadablePanel.create(this, widgets, p -> new Composite(p, SWT.NONE));
    Composite composite = loading.getContents();
    composite.setLayout(new GridLayout(1, false));
    composite.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    button = new Button(composite, SWT.PUSH);
    button.setText("Estimate / Confidence Range");
    button.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));

    tree = new PerfTree(composite, models, widgets);
    tree.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    button.addListener(SWT.Selection, e -> tree.toggleEstimateOrRange());

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

    selectionHandler = new SelectionHandler<Control>(LOG, tree.getControl()) {
      @Override
      protected void updateModel(Event e) {
        models.analytics.postInteraction(View.Performance, ClientAction.Select);
        CommandStream.Node node = tree.getSelection();
        if (node != null) {
          CommandIndex index = node.getIndex();
          if (index == null) {
            models.commands.load(node, () -> models.commands.selectCommands(node.getIndex(), false));
          } else {
            models.commands.selectCommands(index, false);
          }
        }
      }
    };
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
  public void onCommandsSelected(CommandIndex index) {
    selectionHandler.updateSelectionFromModel(() -> models.commands.getTreePath(index).get(), tree::setSelection);
  }

  @Override
  public void onProfileLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(error);
      return;
    }
    // Create columns for all the performance metrics.
    for (Service.ProfilingData.GpuCounters.Metric metric :
        models.profile.getData().getGpuPerformance().getMetricsList()) {
      tree.addColumnForMetric(metric);
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
    private boolean showEstimate = true;

    public PerfTree(Composite parent, Models models, Widgets widgets) {
      super(parent, models, widgets);
    }

    @Override
    protected void addGpuPerformanceColumn() {
      // The performance tab's GPU performances are calculated from server's side.
      // Don't create columns at initialization, the columns will be created after profile is loaded.
      setUpStateForColumnAdding();
    }

    @Override
    protected boolean shouldShowImage(CommandStream.Node node) {
      return false;
    }

    public void toggleEstimateOrRange() {
      showEstimate = !showEstimate;
      refresh();
    }

    private void addColumnForMetric(Service.ProfilingData.GpuCounters.Metric metric) {
      Unit unit = CounterInfo.unitFromString(metric.getUnit());
      TreeViewerColumn column = addColumn(metric.getName() + "(" + unit.name + ")", node -> {
        Service.CommandTreeNode data = node.getData();
        if (data == null) {
          return "";
        } else if (!models.profile.isLoaded()) {
          return "Profiling...";
        } else {
          Service.ProfilingData.GpuCounters.Perf perf = models.profile.getData().getGpuPerformance(data.getCommands().getFromList(), metric.getId());
          if (perf == null) {
            return "";
          }
          if (showEstimate) {
            return perf.getEstimate() < 0 ? "" : unit.format(perf.getEstimate());
          } else {
            String minStr = perf.getMin() < 0 ? "?" : unit.format(perf.getMin());
            String maxStr = perf.getMax() < 0 ? "?" : unit.format(perf.getMax());
            return minStr + " ~ " + maxStr;
          }
        }
      }, DURATION_WIDTH);
      column.getColumn().setAlignment(SWT.RIGHT);
    }

    public void refresh() {
      refresher.refresh();
    }
  }
}
