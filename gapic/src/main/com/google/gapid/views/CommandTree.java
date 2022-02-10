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

import static com.google.common.base.Preconditions.checkState;
import static com.google.gapid.image.Images.noAlpha;
import static com.google.gapid.models.ImagesModel.THUMB_SIZE;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createButton;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createTableForViewer;
import static com.google.gapid.widgets.Widgets.createToggleButton;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMarginOnly;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.CommandStream.Node;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.models.Settings;
import com.google.gapid.perfetto.Unit;
import com.google.gapid.perfetto.models.CounterInfo;
import com.google.gapid.perfetto.views.TraceConfigDialog.GpuCountersDialog;
import com.google.gapid.proto.device.GpuProfiling;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Events;
import com.google.gapid.util.Experimental;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Loadable.Message;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.Style;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.LinkifiedTreeWithImages;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadableImageWidget;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.SearchBox;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Layout;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.SwtUtil;
import org.eclipse.swt.widgets.Table;
import org.eclipse.swt.widgets.TableColumn;
import org.eclipse.swt.widgets.TableItem;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * API command view displaying the commands with their hierarchy grouping in a tree.
 */
public class CommandTree extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Profile.Listener {
  protected static final Logger LOG = Logger.getLogger(CommandTree.class.getName());
  private static final String COMMAND_INDEX_HOVER = "Double click to copy index. Use Ctrl+G to jump to a given command index.";
  private static final String COMMAND_INDEX_DSCRP = "Command index: ";
  private static final int NUM_PRE_CREATED_COLUMNS = 300;

  private final Models models;
  private final Widgets widgets;
  private final Paths.CommandFilter filter;
  private final LoadablePanel<SashForm> loading;
  protected final Tree tree;
  private final LoadablePanel<Table> profileTable;
  private final NodeLookup nodeLookup = new NodeLookup();
  protected final Label commandIdx;
  private final SelectionHandler<Control> selectionHandler;
  private final SingleInFlight searchController = new SingleInFlight();
  private boolean showEstimate = true;
  private final Set<Integer> visibleMetrics = Sets.newHashSet();  // identified by metric.id
  private final ToggleButtonBar counterGroupButtons;

  public CommandTree(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;
    this.filter = models.commands.getFilter();

    setLayout(new GridLayout(1, false));

    loading = withLayoutData(
        LoadablePanel.create(this, widgets, p -> new SashForm(p, SWT.HORIZONTAL)),
        new GridData(SWT.FILL, SWT.FILL, true, true));
    SashForm splitter = loading.getContents();

    Composite left = createComposite(splitter, null);
    Composite topLeft = createComposite(left, withMarginOnly(new GridLayout(2, false), 0, 0));
    SearchBox search = withLayoutData(new SearchBox(topLeft, "Search commands...", false),
        new GridData(SWT.FILL, SWT.CENTER, true, false));
    ToolBar bar = withLayoutData(
        new ToolBar(topLeft, SWT.FLAT), new GridData(SWT.RIGHT, SWT.CENTER, false, false));
    createBaloonToolItem(bar, widgets.theme.filter(), bubble -> {
      bubble.setLayout(new GridLayout(1, false));
      createCheckbox(bubble, "Show Host Commands", filter.showHostCommands,
          e -> filter.showHostCommands = ((Button)e.widget).getSelection());
      createCheckbox(bubble, "Show Submit Info Nodes", filter.showSubmitInfoNodes,
          e -> filter.showSubmitInfoNodes = ((Button)e.widget).getSelection());
      createCheckbox(bubble, "Show Event/Sync Commands", filter.showSyncCommands,
          e -> filter.showSyncCommands = ((Button)e.widget).getSelection());
      createCheckbox(bubble, "Show Begin/End Commands", filter.showBeginEndCommands,
          e -> filter.showBeginEndCommands = ((Button)e.widget).getSelection());
      withLayoutData(createButton(bubble, "Apply", e -> bubble.close()),
          new GridData(SWT.RIGHT, SWT.TOP, false, false));

      bubble.addListener(SWT.Close, e -> models.commands.setFilter(filter));
    }, "Filter");
    tree = new Tree(left, models, widgets);

    Composite right = createComposite(splitter, null);
    int topRightCount = 2 /* toggle, toolbar */;
    if (Experimental.enableProfileExperiments(models.settings)) {
      topRightCount++; /* experiments button */
    }
    Composite topRight = createComposite(
        right, withMarginOnly(new GridLayout(topRightCount, false), 0, 0));
    Button toggleButton = withLayoutData(
        createButton(topRight, "Estimate / Confidence Range", e -> toggleEstimateOrRange()),
        new GridData(SWT.LEFT, SWT.CENTER, false, true));
    toggleButton.setImage(widgets.theme.swap());

    if (Experimental.enableProfileExperiments(models.settings)) {
      Button experimentsButton = withLayoutData(createButton(topRight, "Experiments", e ->
            widgets.experiments.showExperimentsPopup(getShell())),
          new GridData(SWT.LEFT, SWT.CENTER, false, true));
      experimentsButton.setImage(widgets.theme.science());
    }

    counterGroupButtons = withLayoutData(new ToggleButtonBar(topRight),
        new GridData(SWT.LEFT, SWT.CENTER, false, true));

    profileTable = LoadablePanel.create(right, widgets, p ->
        createTableForViewer(p, SWT.H_SCROLL | SWT.V_SCROLL | SWT.SINGLE | SWT.VIRTUAL | SWT.FULL_SELECTION));
    syncTreeWithTable();
    MatchingRowsLayout.setupFor(left, right);

    commandIdx = withLayoutData(createLabel(this, COMMAND_INDEX_DSCRP),
        withIndents(new GridData(SWT.FILL, SWT.BOTTOM, true, false), 3, 0));
    commandIdx.setToolTipText(COMMAND_INDEX_HOVER);
    commandIdx.addListener(SWT.MouseDoubleClick, e -> {
      if (commandIdx.getText().length() > COMMAND_INDEX_DSCRP.length()) {
        widgets.copypaste.setContents(commandIdx.getText().substring(COMMAND_INDEX_DSCRP.length()));
      }
    });

    splitter.setWeights(models.settings.getSplitterWeights(Settings.SplitterWeights.Commands));

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.profile.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.settings.setSplitterWeights(Settings.SplitterWeights.Commands, splitter.getWeights());

      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.profile.removeListener(this);
    });

    search.addListener(Events.Search, e -> search(e.text, (e.detail & Events.REGEX) != 0));

    selectionHandler = new SelectionHandler<Control>(LOG, tree.getControl()) {
      @Override
      protected void updateModel(Event e) {
        models.analytics.postInteraction(View.Commands, ClientAction.Select);
        CommandStream.Node node = tree.getSelection();
        if (node != null) {
          CommandIndex index = node.getIndex();
          if (index == null) {
            models.commands.load(node, () -> models.commands.selectCommands(node.getIndex(), false));
          } else {
            commandIdx.setText(COMMAND_INDEX_DSCRP + node.getIndexString());
            models.commands.selectCommands(index, false);
          }
          models.profile.linkCommandToGpuGroup(node.getCommandStart());
        }
      }
    };

    CommandOptions.CreateCommandOptionsMenu(tree.getControl(), widgets, tree, this.models);

    tree.registerAsCopySource(widgets.copypaste, node -> {
      models.analytics.postInteraction(View.Commands, ClientAction.Copy);
      Service.CommandTreeNode data = node.getData();
      if (data == null) {
        // Copy before loaded. Not ideal, but this is unlikely.
        return new String[] { "Loading..." };
      }

      StringBuilder result = new StringBuilder();
      if (data.getGroup().isEmpty() && data.hasCommands()) {
        result.append(data.getCommands().getTo(0)).append(": ");
        API.Command cmd = node.getCommand();
        if (cmd == null) {
          // Copy before loaded. Not ideal, but this is unlikely.
          result.append("Loading...");
        } else {
          result.append(Formatter.toString(cmd, models.constants::getConstants));
        }
      } else {
        result.append(data.getCommands().getFrom(0)).append(": ").append(data.getGroup());
      }
      return new String[] { result.toString() };
    }, true);
  }

  private void syncTreeWithTable() {
    Table table = profileTable.getContents();
    for (int i = 0; i < NUM_PRE_CREATED_COLUMNS; i++) {
      new TableColumn(table, SWT.RIGHT);
    }

    table.addListener(SWT.SetData, event -> {
      updateTableRow((TableItem)event.item, event.index, false);
    });

    tree.getTree().addListener(SWT.Expand, event -> {
      CommandStream.Node node = (CommandStream.Node)event.item.getData();
      if (node != null) {
        nodeLookup.expand(node, table);
      }
    });
    tree.getTree().addListener(SWT.Collapse, event -> {
      CommandStream.Node node = (CommandStream.Node)event.item.getData();
      if (node != null) {
        nodeLookup.collapse(node, table);
      }
    });

    tree.getTree().addListener(SWT.Selection, event -> {
      TreeItem[] selection = tree.getTree().getSelection();
      if (selection.length == 0) {
        table.setSelection(-1);
      } else {
        table.setSelection(nodeLookup.getIndex((CommandStream.Node)selection[0].getData()));
      }
    });
    table.addListener(SWT.Selection, event -> {
      int index = table.getSelectionIndex();
      if (index < 0) {
        tree.setSelection(null);
      } else {
        CommandStream.Node node = nodeLookup.getNode(index);
        tree.setSelection(node.getTreePath());
        models.commands.selectCommands(node.getIndex(), false);
      }
    });

    SwtUtil.syncTreeAndTableScroll(tree.getTree(), table);
  }

  private void updateTableRow(TableItem item, int index, boolean redraw) {
    if (index < 0) {
      index = profileTable.getContents().indexOf(item);
      if (index < 0) {
        return;
      }
    }

    CommandStream.Node node = nodeLookup.getNode(index);
    Service.CommandTreeNode data = node.getData();
    if (data == null) {
      // Node is still loading, make sure the table gets updated when it's done.
      models.commands.load(node, () -> updateTableRow(item, -1, true));
    } else if (models.profile.isLoaded()) {
      Profile.PerfNode perf = models.profile.getData().getPerfNode(data);
      if (perf == null) {
        return;
      }
      Service.ProfilingData.GpuCounters counters = models.profile.getData().getGpuPerformance();
      for (int metricIdx = 0, itemIdx = 0; metricIdx < counters.getMetricsCount(); metricIdx++) {
        Service.ProfilingData.GpuCounters.Metric metric = counters.getMetrics(metricIdx);
        if (!visibleMetrics.contains(metric.getId())) {
          continue;
        }

        Profile.PerfNode.Value value = perf.getPerf(metric);
        if (value != null) {
          Unit unit = CounterInfo.unitFromString(metric.getUnit());
          item.setText(itemIdx, value.format(unit, showEstimate));
        }

        itemIdx++;
      }
      if (redraw) {
        item.getParent().redraw();
      }
    }
  }

  private void toggleEstimateOrRange() {
    showEstimate = !showEstimate;
    Table table = profileTable.getContents();
    table.clearAll();
  }

  private void search(String text, boolean regex) {
    models.analytics.postInteraction(View.Commands, ClientAction.Search);
    CommandStream.Node parent = models.commands.getData();
    if (parent != null && !text.isEmpty()) {
      CommandStream.Node selection = tree.getSelection();
      if (selection != null) {
        parent = selection;
      }
      searchController.start().listen(
          MoreFutures.transformAsync(models.commands.search(parent, text, regex),
              r -> models.commands.getTreePath(models.commands.getData(), Lists.newArrayList(),
                  r.getCommandTreeNode().getIndicesList().iterator())),
          new UiCallback<TreePath, TreePath>(tree, LOG) {
            @Override
            protected TreePath onRpcThread(Rpc.Result<TreePath> result)
                throws RpcException, ExecutionException {
              return result.get();
            }

            @Override
            protected void onUiThread(TreePath result) {
              select(result);
            }
          });
    }
  }

  protected void select(TreePath path) {
    models.commands.selectCommands(((CommandStream.Node)path.getLastSegment()).getIndex(), true);
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
  public void onCommandsLoadingStart() {
    updateTree(false);
  }

  @Override
  public void onCommandsLoaded() {
    updateTree(false);
  }

  @Override
  public void onCommandsSelected(CommandIndex index) {
    selectionHandler.updateSelectionFromModel(() -> models.commands.getTreePath(index.getNode()).get(), tree::setSelection);
  }

  @Override
  public void onProfileLoadingStart() {
    profileTable.showMessage(Message.loading(Messages.LOADING_PROFILE));
  }

  @Override
  public void onProfileLoaded(Loadable.Message error) {
    if (error != null) {
      profileTable.showMessage(error);
      return;
    }

    profileTable.stopLoading();

    // Update the group buttons.
    counterGroupButtons.clear();
    counterGroupButtons.addLabel("Counters:");
    List<Service.ProfilingData.CounterGroup> groups = models.profile.getData().getCounterGroups();
    for (Service.ProfilingData.CounterGroup group : groups) {
      counterGroupButtons.addButton(group.getLabel(), e -> {
        selectCounterGroup(group);
        updateTable();
      });
    }
    counterGroupButtons.addButton("All", e -> {
      selectAllCounters();
      updateTable();
    });
    counterGroupButtons.addButton("Custom", e -> {
      GpuCountersDialog dialog = new GpuCountersDialog(
          getShell(), widgets.theme, getCounterSpecs(), Lists.newArrayList(visibleMetrics));
      if (dialog.open() == Window.OK) {
        visibleMetrics.clear();
        visibleMetrics.addAll(dialog.getSelectedIds());
        updateTable();
      }
    });
    counterGroupButtons.selectButton(0);
    counterGroupButtons.requestLayout();

    if (groups.isEmpty()) {
      selectAllCounters();
    } else {
      selectCounterGroup(groups.get(0));
    }

    // Update the table UI.
    updateTable();
  }

  private void selectCounterGroup(Service.ProfilingData.CounterGroup counterGroup) {
    visibleMetrics.clear();
    models.profile.getData().getGpuPerformance().getMetricsList().stream()
        .filter(m -> isStaticAnalysisCounter(m) || m.getCounterGroupIdsList().contains(counterGroup.getId()))
        .mapToInt(Service.ProfilingData.GpuCounters.Metric::getId)
        .boxed()
        .forEach(visibleMetrics::add);
  }

  private void selectAllCounters() {
    visibleMetrics.clear();
    models.profile.getData().getGpuPerformance().getMetricsList().stream()
        .mapToInt(Service.ProfilingData.GpuCounters.Metric::getId)
        .boxed()
        .forEach(visibleMetrics::add);
  }

  private void updateTree(boolean assumeLoading) {
    if (assumeLoading || !models.commands.isLoaded()) {
      loading.startLoading();
      profileTable.showMessage(Message.loading(Messages.LOADING_CAPTURE));

      tree.setInput(null);
      profileTable.getContents().setItemCount(0);
      nodeLookup.reset();
      commandIdx.setText(COMMAND_INDEX_DSCRP);
      return;
    }

    loading.stopLoading();
    nodeLookup.setRoot(models.commands.getData());
    tree.setInput(models.commands.getData());
    profileTable.getContents().setItemCount(nodeLookup.size());

    if (models.commands.getSelectedCommands() != null) {
      onCommandsSelected(models.commands.getSelectedCommands());
    }
    if (models.profile.isLoaded()) {
      onProfileLoaded(null);
    }
  }

  private void updateTable() {
    Table table = profileTable.getContents();
    Service.ProfilingData.GpuCounters counterz = models.profile.getData().getGpuPerformance();
    List<Service.ProfilingData.GpuCounters.Metric> metrics = counterz.getMetricsList().stream()
        .filter(metric -> visibleMetrics.contains(metric.getId()))
        .collect(toList());
    int done = 0;
    for (; done < table.getColumnCount() && done < metrics.size(); done++) {
      table.getColumn(done).setText(metrics.get(done).getName());
    }
    for (; done < metrics.size(); done++) {
      TableColumn column = new TableColumn(table, SWT.RIGHT);
      column.setText(metrics.get(done).getName());
    }
    for (int i = table.getColumnCount() - 1; i >= done; i--) {
      table.getColumn(i).dispose();
    }

    int[] order = new int[metrics.size()];
    done = 0;
    for (int i = 0; i < order.length; i++) {
      if (isStaticAnalysisCounter(metrics.get(i))) {
        order[done++] = i;
      }
    }
    for (int i = 0; i < order.length; i++) {
      if (!isStaticAnalysisCounter(metrics.get(i))) {
        order[done++] = i;
      }
    }
    table.setColumnOrder(order);

    packColumns(table);
    table.clearAll();
  }

  private List<GpuProfiling.GpuCounterDescriptor.GpuCounterSpec> getCounterSpecs() {
    if (!models.profile.isLoaded()) {
      return Collections.emptyList();
    }
    // To reuse the existing GpuCountersDialog class for displaying, forge GpuCounterSpec instances
    // with the minimum data requirement that is needed and referenced in GpuCountersDialog.
    return models.profile.getData().getGpuPerformance().getMetricsList().stream()
        .sorted((m1, m2) -> {
          if (isStaticAnalysisCounter(m1) != isStaticAnalysisCounter(m2)) {
            return isStaticAnalysisCounter(m1) ? -1 : 1;
          }
          return Integer.compare(m1.getId(), m2.getId());
        })
        .map(metric -> GpuProfiling.GpuCounterDescriptor.GpuCounterSpec.newBuilder()
            .setName(metric.getName())
            .setDescription(metric.getDescription())
            .setCounterId(metric.getId())
            .setSelectByDefault(metric.getSelectByDefault() || isStaticAnalysisCounter(metric))
            .build())
        .collect(toList());
  }

  private static boolean isStaticAnalysisCounter(Service.ProfilingData.GpuCounters.Metric metric) {
    return metric.getType() != Service.ProfilingData.GpuCounters.Metric.Type.Hardware;
  }

  protected static class Tree extends LinkifiedTreeWithImages<CommandStream.Node, String> {
    protected final Models models;
    private final Widgets widgets;
    private final Map<Long, Color> threadBackgroundColors = Maps.newHashMap();

    public Tree(Composite parent, Models models, Widgets widgets) {
      super(parent, SWT.H_SCROLL | SWT.V_SCROLL | SWT.SINGLE, widgets);
      this.models = models;
      this.widgets = widgets;

      // To match up with the table, show an empty header here.
      getTree().setHeaderVisible(true);
    }

    public void updateTree(TreeItem item) {
      labelProvider.updateHierarchy(item);
    }

    @Override
    protected ContentProvider<Node> createContentProvider() {
      return new ContentProvider<CommandStream.Node>() {
        @Override
        protected boolean hasChildNodes(CommandStream.Node element) {
          return element.getChildCount() > 0;
        }

        @Override
        protected CommandStream.Node[] getChildNodes(CommandStream.Node node) {
          return node.getChildren();
        }

        @Override
        protected CommandStream.Node getParentNode(CommandStream.Node child) {
          return child.getParent();
        }

        @Override
        protected boolean isLoaded(CommandStream.Node element) {
          return element.getData() != null;
        }

        @Override
        protected boolean isDefaultExpanded(CommandStream.Node element) {
          Service.CommandTreeNode data = element.getData();
          return data != null && data.getExpandByDefault();
        }

        @Override
        protected void load(CommandStream.Node node, Runnable callback) {
          models.commands.load(node, callback);
        }
      };
    }

    private Style getCommandStyle(Service.CommandTreeNode node, StylingString string) {
      if (node.getExperimentalCommandsCount() == 0) {
        return string.labelStyle();
      }

      final List<Path.Command> experimentCommands = node.getExperimentalCommandsList();
      if (widgets.experiments.areAllCommandsDisabled(experimentCommands)) {
        return string.disabledLabelStyle();
      }

      if (widgets.experiments.isAnyCommandDisabled(experimentCommands)) {
        return string.semiDisabledLabelStyle();
      }

      return string.labelStyle();
    }

    @Override
    protected <S extends StylingString> S format(
        CommandStream.Node element, S string, Follower.Prefetcher<String> follower) {
      Service.CommandTreeNode data = element.getData();
      if (data == null) {
        string.append("Loading...", string.structureStyle());
      } else {
        if (data.getGroup().isEmpty() && data.hasCommands()) {
          API.Command cmd = element.getCommand();
          if (cmd == null) {
            string.append("Loading...", string.structureStyle());
          } else {
            Formatter.format(cmd, models.constants::getConstants, follower::canFollow,
                string, getCommandStyle(data, string), string.identifierStyle());
          }
        } else {
          string.append(data.getGroup(), getCommandStyle(data, string));
          long count = data.getNumCommands();
          string.append(
              " (" + count + " command" + (count != 1 ? "s" : "") + ")", string.structureStyle());
        }
      }
      return string;
    }

    @Override
    protected boolean shouldShowImage(CommandStream.Node node) {
      return models.images.isReady() &&
          node.getData() != null && !node.getData().getGroup().isEmpty();
    }

    @Override
    protected ListenableFuture<ImageData> loadImage(CommandStream.Node node, int size) {
      return noAlpha(models.images.getThumbnail(
          node.getPath(Path.CommandTreeNode.newBuilder()).build(), size, i -> { /*noop*/ }));
    }

    @Override
    protected void createImagePopupContents(Shell shell, CommandStream.Node node) {
      LoadableImageWidget.forImage(
          shell, LoadableImage.newBuilder(widgets.loading)
              .forImageData(loadImage(node, THUMB_SIZE))
              .onErrorShowErrorIcon(widgets.theme))
      .withImageEventListener(new LoadableImage.Listener() {
        @Override
        public void onLoaded(boolean success) {
          if (success) {
            Widgets.ifNotDisposed(shell,() -> {
              Point oldSize = shell.getSize();
              Point newSize = shell.computeSize(SWT.DEFAULT, SWT.DEFAULT);
              shell.setSize(newSize);
              if (oldSize.y != newSize.y) {
                Point location = shell.getLocation();
                location.y += (oldSize.y - newSize.y) / 2;
                shell.setLocation(location);
              }
            });
          }
        }
      });
    }

    @Override
    protected Follower.Prefetcher<String> prepareFollower(CommandStream.Node node, Runnable cb) {
      return models.follower.prepare(node, cb);
    }

    @Override
    protected void follow(Path.Any path) {
      models.follower.onFollow(path);
    }

    @Override
    public void reset() {
      super.reset();
      for (Color color : threadBackgroundColors.values()) {
        color.dispose();
      }
      threadBackgroundColors.clear();
    }
  }

  private static class NodeLookup {
    private final List<CommandStream.Node> nodes = Lists.newArrayList();
    private final Set<CommandStream.Node> expanded = Sets.newHashSet();
    private final Map<CommandStream.Node, Integer> rows = Maps.newIdentityHashMap();

    public NodeLookup() {
    }

    public void reset() {
      nodes.clear();
      expanded.clear();
      rows.clear();
    }

    public void setRoot(CommandStream.Node root) {
      reset();
      for (CommandStream.Node node : root.getChildren()) {
        rows.put(node, nodes.size());
        nodes.add(node);
      }
    }

    public void expand(CommandStream.Node node, Table table) {
      int row = rows.get(node);
      nodes.addAll(row + 1, Arrays.asList(node.getChildren()));
      for (int i = row + 1; i < nodes.size(); i++) {
        rows.put(nodes.get(i), i);
      }
      for (int i = 0; i < node.getChildren().length; i++) {
        new TableItem(table, SWT.NONE, row + 1 + i);
      }
      expanded.add(node);
    }

    public void collapse(CommandStream.Node node, Table table) {
      for (CommandStream.Node child : node.getChildren()) {
        if (expanded.contains(child)) {
          // TODO(pmuetschard): we could possibly improve performance by computing the last element
          // and removing all rows at once, rather than collapsing expanded descendent node.
          collapse(child, table);
        }
      }

      int row = rows.get(node);
      nodes.subList(row + 1, row + 1 + node.getChildren().length).clear();
      for (int i = row + 1; i < nodes.size(); i++) {
        rows.put(nodes.get(i), i);
      }
      table.remove(row + 1, row /*inclusive*/ + node.getChildren().length);
      expanded.remove(node);
    }

    public CommandStream.Node getNode(int row) {
      return nodes.get(row);
    }

    public int getIndex(CommandStream.Node node) {
      Integer index = rows.get(node);
      if (index == null) {
        LOG.log(WARNING, "Asked for index of non-existing row: " + node);
        // This is currently only used by the selection logic. Returning -1 here will simply clear
        // the selection, which is not ideal, but also not really an issue.
        return -1;
      }
      return index;
    }

    public int size() {
      return nodes.size();
    }
  }

  /**
   * A {@link Layout} that lays out corresponding elements within two composites, so that they have
   * the same height. Currently, the last element on each side gets all the remaining space.
   */
  private static class MatchingRowsLayout extends Layout {
    private final Composite left;
    private final Composite right;

    private MatchingRowsLayout(Composite left, Composite right) {
      this.left = left;
      this.right = right;
    }

    public static void setupFor(Composite left, Composite right) {
      MatchingRowsLayout layout = new MatchingRowsLayout(left, right);
      left.setLayout(layout);
      right.setLayout(layout);
    }

    @Override
    protected Point computeSize(Composite composite, int wHint, int hHint, boolean flushCache) {
      checkState(composite == left || composite == right, "Asked to layout an unknown composite");
      boolean forLeft = composite == left;
      Control[] lControls = left.getChildren();
      Control[] rControls = right.getChildren();
      checkState(lControls.length == rControls.length,
          "Unbalanced composites: %s != %s", lControls.length, rControls.length);

      int w = 0, h = 0;
      for (int i = 0; i < lControls.length; i++) {
        Point lSize = lControls[i].computeSize(wHint, hHint, flushCache && forLeft);
        Point rSize = rControls[i].computeSize(wHint, hHint, flushCache && !forLeft);
        w = Math.max(w, forLeft ? lSize.x : rSize.x);
        h += Math.max(lSize.y, rSize.y);
      }
      return new Point((wHint == SWT.DEFAULT) ? w : wHint, (hHint == SWT.DEFAULT) ? h : hHint);
    }

    @Override
    protected void layout(Composite composite, boolean flushCache) {
      checkState(composite == left || composite == right, "Asked to layout an unknown composite");
      boolean forLeft = composite == left;
      Control[] lControls = left.getChildren();
      Control[] rControls = right.getChildren();
      checkState(lControls.length == rControls.length,
          "Unbalanced composites: %s != %s", lControls.length, rControls.length);

      Rectangle size = composite.getClientArea();
      for (int i = 0; i < lControls.length - 1; i++) {
        Point lSize = lControls[i].computeSize(size.width, SWT.DEFAULT, flushCache && forLeft);
        Point rSize = rControls[i].computeSize(size.width, SWT.DEFAULT, flushCache && !forLeft);
        int h = Math.max(lSize.y, rSize.y);
        (forLeft ? lControls[i] : rControls[i]).setBounds(size.x, size.y, size.width, h);
        size.y += h;
        size.height -= h;
      }
      Control last = forLeft ? lControls[lControls.length - 1] : rControls[rControls.length - 1];
      last.setBounds(size);
    }
  }

  private static class ToggleButtonBar extends Composite {
    private final List<Button> buttons = Lists.newArrayList();
    private final Listener selectionListener = e -> {
      for (Button button : buttons) {
        button.setSelection(e.widget == button);
      }
    };

    public ToggleButtonBar(Composite parent) {
      super(parent, SWT.NONE);
      setLayout(withMarginOnly(new GridLayout(0, false), 0, 0));
    }

    public void clear() {
      for (Control child : getChildren()) {
        child.dispose();
      }
      buttons.clear();
      ((GridLayout)getLayout()).numColumns = 0;
    }

    public void addLabel(String text) {
      ((GridLayout)getLayout()).numColumns++;
      withLayoutData(createLabel(this, text), new GridData(SWT.LEFT, SWT.CENTER, false, true));
    }

    public void addButton(String text, Listener listener) {
      ((GridLayout)getLayout()).numColumns++;
      Button button = withLayoutData(createToggleButton(this, text, listener),
          new GridData(SWT.LEFT, SWT.CENTER, false, true));
      buttons.add(button);
      button.addListener(SWT.Selection, selectionListener);
    }

    public void selectButton(int idx) {
      for (int i = 0; i < buttons.size(); i++) {
        buttons.get(i).setSelection(i == idx);
      }
    }
  }
}
