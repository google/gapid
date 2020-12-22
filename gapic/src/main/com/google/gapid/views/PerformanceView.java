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
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;

import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.models.Profile.PerfNode;
import com.google.gapid.perfetto.Unit;
import com.google.gapid.perfetto.models.CounterInfo;
import com.google.gapid.proto.service.Service;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;
import java.util.List;
import java.util.function.Predicate;
import java.util.logging.Logger;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.TreeColumn;
import org.eclipse.swt.widgets.TreeItem;

public class PerformanceView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Profile.Listener {
  protected static final Logger LOG = Logger.getLogger(PerformanceView.class.getName());

  private final Models models;
  private final LoadablePanel<Composite> loading;
  private final Button button;
  private final TreeViewer tree;
  private boolean showEstimate = true;

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
    button.addListener(SWT.Selection, e -> toggleEstimateOrRange());

    tree = createTreeViewer(composite, SWT.NONE);

    tree.getTree().setHeaderVisible(true);
    tree.setLabelProvider(new LabelProvider());
    tree.setContentProvider(new ITreeContentProvider() {
      @SuppressWarnings("unchecked")
      @Override
      public Object[] getElements(Object inputElement) {
        return ((List<Profile.PerfNode>)inputElement).toArray();
      }

      @Override
      public boolean hasChildren(Object element) {
        return element instanceof Profile.PerfNode && ((Profile.PerfNode)element).hasChildren();
      }

      @Override
      public Object getParent(Object element) {
        return null;
      }

      @Override
      public Object[] getChildren(Object element) {
        if (element instanceof Profile.PerfNode) {
          return ((Profile.PerfNode)element).getChildren().toArray();
        } else {
          return new Object[0];
        }
      }
    });

    tree.getTree().addListener(SWT.Selection, e -> {
      TreeItem[] items = tree.getTree().getSelection();
      if (items.length > 0) {
        PerfNode selection = (PerfNode)items[0].getData();
        models.profile.linkGpuGroupToCommand(selection.getGroup());
        models.profile.selectGroup(selection.getGroup());
      }
    });

    tree.getTree().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
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
  public void onGroupSelected(Service.ProfilingData.GpuSlices.Group group) {
    TreeItem item = findItem(tree.getTree().getItems(), n -> group.getId() == n.getGroup().getId());
    if (item != null) {
      tree.getTree().setSelection(item);
    }
  }

  @Override
  public void onProfileLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(error);
      return;
    }
    updateTree(false);
  }

  private void updateTree(boolean assumeLoading) {
    if (assumeLoading || !models.profile.isLoaded()) {
      loading.startLoading();
      tree.setInput(null);
      return;
    }
    loading.stopLoading();

    List<Profile.PerfNode> perf = models.profile.getData().getPerfNodes();
    for (TreeColumn col : tree.getTree().getColumns()) {
      col.dispose();
    }
    createTreeColumn(tree, "Name", e -> ((Profile.PerfNode)e).getGroup().getName());
    for (Service.ProfilingData.GpuCounters.Metric metric :
        models.profile.getData().getGpuPerformance().getMetricsList()) {
      addColumnForMetric(metric);
    }
    tree.setInput(perf);
    tree.expandAll(true);
    packColumns(tree.getTree());
    tree.refresh();
  }

  private void addColumnForMetric(Service.ProfilingData.GpuCounters.Metric metric) {
    Unit unit = CounterInfo.unitFromString(metric.getUnit());
    TreeViewerColumn column = createTreeColumn(tree, metric.getName() + "(" + unit.name + ")", e -> {
      Profile.PerfNode node = (Profile.PerfNode)e;
      if (node == null || node.getPerfs() == null) {
        return "";
      } else if (!models.profile.isLoaded()) {
        return "Profiling...";
      } else {
        Service.ProfilingData.GpuCounters.Perf perf = node.getPerfs().get(metric.getId());
        if (showEstimate) {
          return perf.getEstimate() < 0 ? "" : unit.format(perf.getEstimate());
        } else {
          String minStr = perf.getMin() < 0 ? "?" : unit.format(perf.getMin());
          String maxStr = perf.getMax() < 0 ? "?" : unit.format(perf.getMax());
          return minStr + " ~ " + maxStr;
        }
      }
    });
    column.getColumn().setAlignment(SWT.RIGHT);
  }

  private void toggleEstimateOrRange() {
    showEstimate = !showEstimate;
    packColumns(tree.getTree());
    tree.refresh();
  }

  // For a tree-structured TreeItem, do a preorder traversal, to find the PerfNode that meets the
  // requirement.
  private TreeItem findItem(TreeItem root, Predicate<PerfNode> requirement) {
    if (root == null || !(root.getData() instanceof PerfNode)) {
      return null;
    }
    PerfNode rootNode = (PerfNode)root.getData();
    if (requirement.test(rootNode)) {
      return root;
    }
    return findItem(root.getItems(), requirement);
  }

  // For multiple tree-structured TreeItems, do a preorder traversal to each of them, to find the
  // PerfNode that meets the requirement.
  private TreeItem findItem(TreeItem[] roots, Predicate<PerfNode> requirement) {
    for (TreeItem root : roots) {
      TreeItem result = findItem(root, requirement);
      if (result != null) {
        return result;
      }
    }
    return null;
  }
}
