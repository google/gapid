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

import static com.google.gapid.image.Images.createNonScaledImage;
import static com.google.gapid.image.Images.noAlpha;
import static com.google.gapid.models.Follower.nullPrefetcher;
import static com.google.gapid.models.ImagesModel.THUMB_SIZE;
import static com.google.gapid.models.ImagesModel.scaleImage;
import static com.google.gapid.perfetto.canvas.Tooltip.LocationComputer.horizontallyCenteredAndConstrained;
import static com.google.gapid.perfetto.canvas.Tooltip.LocationComputer.standardTooltip;
import static com.google.gapid.perfetto.canvas.Tooltip.LocationComputer.verticallyCenteredAndConstrained;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createButton;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createToggleButton;
import static com.google.gapid.widgets.Widgets.createVerticalSash;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMarginOnly;
import static java.util.function.Function.identity;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Follower;
import com.google.gapid.models.ImagesModel;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Fonts.Style;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.canvas.Size;
import com.google.gapid.perfetto.canvas.Tooltip;
import com.google.gapid.perfetto.views.TraceConfigDialog.GpuCountersDialog;
import com.google.gapid.proto.device.GpuProfiling;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Experimental;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Scheduler;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.LinkableStyledString;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.SearchBox;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StyleRange;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.TextLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Layout;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Sash;
import org.eclipse.swt.widgets.Slider;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.Widget;

import java.util.Arrays;
import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;
import java.util.logging.Level;
import java.util.logging.Logger;
import java.util.stream.Collectors;

public class CommandTree extends Canvas
    implements Tab, Capture.Listener, CommandStream.Listener, Profile.Listener {
  protected static final Logger LOG = Logger.getLogger(CommandTree.class.getName());
  private static final String COMMAND_INDEX_HOVER = "Double click to copy index. Use Ctrl+G to jump to a given command index.";
  private static final String COMMAND_INDEX_DSCRP = "Command index: ";

  private static final int SASH_WIDTH = 3;
  private static final int MIN_CHILD_SIZE = 150;
  private static final double Y_PADDING = 5;
  private static final double HEADER_MARGIN = 10;
  private static final double COLUMN_SPACING = 8;
  private static final double ROW_SPACING = 4;
  private static final double TREE_INDENT = 10;
  private static final double CARRET_SIZE_LONG = 8;
  private static final double CARRET_SIZE_SHORT = 4;
  private static final double CARRET_STROKE = 1.75;
  private static final double CARRET_Y_PADDING = 4;
  private static final double CARRET_X_SIZE = CARRET_SIZE_LONG + 2;
  private static final int IMAGE_SIZE = 18;
  private static final double IMAGE_PADDING = 6;
  private static final int PREVIEW_HOVER_DELAY_MS = 500;
  private static final double TOOLTIP_OFFSET = 5;

  private final Models models;
  private final Widgets widgets;
  private final Paths.CommandFilter filter;

  private final RenderContext.Global context;
  private final SizeData size;
  private final TreeState tree;
  private final ImageProvider images;

  private final MatchingRowsLayout layout;
  private final Slider treeHBar;
  private final Slider tableHBar;
  private final Slider commonVBar;
  private final ToggleButtonBar counterGroupButtons;
  private final Label commandIdx;
  private final SelectionHandler<Control> selectionHandler;
  private final SingleInFlight searchController = new SingleInFlight();

  private Loadable.Message tableMessage = Loadable.Message.loading(Messages.LOADING_CAPTURE);
  private int hoveredRow = -1;
  private TreeState.Row rowMarkedAsHovered = null;
  private Follower.Prefetcher<String> lastPrefetcher = nullPrefetcher();
  private int selectedRow = -1;
  private Tooltip columnTooltip = null;
  private Tooltip valueTooltip = null;
  private CommandStream.Node hoveredNode = null;
  private Future<?> lastScheduledFuture = Futures.immediateFuture(null);
  private boolean showImagePreview = false;

  private final LoadingIndicator.Repaintable loadingRepainter;
  private final LoadingIndicator.Repaintable previewRepainter;

  public CommandTree(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NO_BACKGROUND | SWT.DOUBLE_BUFFERED);
    this.models = models;
    this.widgets = widgets;
    this.filter = models.commands.getFilter();

    this.context = new RenderContext.Global(widgets.theme, this);
    this.size = new SizeData();
    this.tree = new TreeState();
    this.images = new ImageProvider(this, models.images);

    this.layout = new MatchingRowsLayout(size, models.settings.ui().getCommandSplitterRatio());
    setLayout(layout);

    Composite topLeft = createComposite(this, withMarginOnly(new GridLayout(2, false), 0, 0));
    SearchBox search = withLayoutData(new SearchBox(topLeft, "Search commands...", false),
        new GridData(SWT.FILL, SWT.CENTER, true, true));
    ToolBar bar = withLayoutData(
        new ToolBar(topLeft, SWT.FLAT), new GridData(SWT.RIGHT, SWT.CENTER, false, true));
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

    createVerticalSash(this, this::onSashMoved);

    int topRightCount = 1 /* toolbar */;
    if (Experimental.enableProfileExperiments(models.settings)) {
      topRightCount++; /* experiments button */
    }
    Composite topRight = createComposite(
        this, withMarginOnly(new GridLayout(topRightCount, false), 0, 0));

    if (Experimental.enableProfileExperiments(models.settings)) {
      Button experimentsButton = withLayoutData(createButton(topRight, "Experiments", e ->
            widgets.experiments.showExperimentsPopup(getShell())),
          new GridData(SWT.LEFT, SWT.CENTER, false, true));
      experimentsButton.setImage(widgets.theme.science());
    }

    counterGroupButtons = withLayoutData(new ToggleButtonBar(topRight),
        new GridData(SWT.LEFT, SWT.CENTER, false, true));

    treeHBar = new Slider(this, SWT.HORIZONTAL);
    tableHBar = new Slider(this, SWT.HORIZONTAL);
    commonVBar = new Slider(this, SWT.VERTICAL);

    Composite footer = createComposite(this, withMarginOnly(new GridLayout(1, false), 2, 2));
    commandIdx = createLabel(footer, COMMAND_INDEX_DSCRP);
    commandIdx.setToolTipText(COMMAND_INDEX_HOVER);
    commandIdx.addListener(SWT.MouseDoubleClick, e -> {
      if (commandIdx.getText().length() > COMMAND_INDEX_DSCRP.length()) {
        widgets.copypaste.setContents(commandIdx.getText().substring(COMMAND_INDEX_DSCRP.length()));
      }
    });

    addListener(SWT.Paint, this::onPaint);
    addListener(SWT.Resize, e -> layout.onResize(getClientArea().width));
    addListener(SWT.MouseDown, this::onMouseDown);
    addListener(SWT.MouseUp, this::onMouseUp);
    addListener(SWT.MouseMove, e -> updateHover(e.x, e.y));
    addListener(SWT.MouseExit, e -> updateHover(0, 0));
    addListener(SWT.MouseWheel, this::onMouseWheel);
    addListener(SWT.MouseHorizontalWheel, this::onMouseHorizontalWheel);
    addListener(SWT.KeyDown, this::onKeyDown);

    treeHBar.addListener(SWT.Selection, e -> redraw(size.tree));
    tableHBar.addListener(SWT.Selection, e -> redraw(size.table));
    commonVBar.addListener(SWT.Selection, e -> redraw(Area.FULL));

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.profile.addListener(this);
    addListener(SWT.Dispose, e -> {
      context.dispose();
      tree.dispose();
      images.dispose();
      models.settings.writeUi().setCommandSplitterRatio(layout.getRatio());

      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.profile.removeListener(this);
    });

    search.addListener(Events.Search, e -> search(e.text, (e.detail & Events.REGEX) != 0));

    selectionHandler = new SelectionHandler<Control>(LOG, this) {
      @Override
      protected void updateModel(Event e) {
        models.analytics.postInteraction(View.Commands, ClientAction.Select);
        updateSelection(true);
      }
    };

    loadingRepainter = () -> {
      if (!isDisposed()) {
        updateSize(true, 0);
        redraw(size.table);
      }
    };
    previewRepainter = () -> {
      if (!isDisposed()) {
        redraw(Area.FULL); // TODO
      }
    };

    updateSize(false, 0);
  }

  private void onSashMoved(Event e) {
    Rectangle area = getClientArea();
    if (area.width < 2 * MIN_CHILD_SIZE + SASH_WIDTH) {
      e.doit = false;
      return;
    }

    e.x = Math.max(MIN_CHILD_SIZE, Math.min(area.width - SASH_WIDTH - MIN_CHILD_SIZE, e.x));
    if (e.x != ((Sash)e.widget).getBounds().x) {
      layout.onSashMoved(e.x, area.width);
      layout();
      redraw();
    }
  }

  private void onPaint(Event e) {
    long start = System.nanoTime();
    Rectangle clip = e.gc.getClipping();
    e.gc.fillRectangle(clip);
    try (RenderContext ctx = context.newContext(e.gc)) {
      ctx.withClipAndTranslation(size.tree, () -> {
        drawTree(ctx);
      });
      ctx.withClipAndTranslation(size.table, () -> {
        drawTable(ctx);
      });
      if (showImagePreview) {
        ctx.trace("ImagePreview", () -> drawImagePreview(ctx));
      }
      if (columnTooltip != null) {
        ctx.trace("ColumnTooltip", () -> columnTooltip.render(ctx));
      }
      if (valueTooltip != null) {
        ctx.trace("ValueTooltip", () -> valueTooltip.render(ctx));
      }
    }
    long end = System.nanoTime();
    LOG.log(Level.FINE, clip + " (" + (end - start) / 1000000.0 + ")");
  }

  private void onMouseDown(Event e) {
    if (e.widget instanceof Control) {
      ((Control)e.widget).setFocus();
    }
  }

  private void onMouseUp(Event e) {
    double y = e.y - size.tree.y + commonVBar.getSelection();
    if (y >= size.headerHeight) {
      int rowIdx = size.getRow(y - size.headerHeight);
      if (rowIdx < tree.getNumRows()) {
        if (e.count == 2) {
          if (tree.toggle(rowIdx, this::initRows)) {
            updateSize(false, rowIdx + 1);
            updateHover(e.x, e.y);
            redraw(Area.FULL);
          }
        } else {
          selectRow(rowIdx);
          if (size.tree.contains(e.x, e.y)) {
            TreeState.Row row = tree.getRow(rowIdx);
            double x = e.x - size.tree.x + treeHBar.getSelection();
            double cx = x  - row.getCarretX();
            if (cx >= -IMAGE_PADDING / 2 && cx < CARRET_X_SIZE + IMAGE_PADDING / 2 &&
                row.node.getChildCount() > 0) {
              if (tree.toggle(rowIdx, this::initRows)) {
                updateHover(e.x, e.y);
                updateSize(false, rowIdx + 1);
              }
            } else {
              Path.Any follow = getFollow(x);
              if (follow != null) {
                models.follower.onFollow(follow);
              }
            }
          }
          redraw(Area.FULL);
        }
      }
    }
  }

  private void onMouseWheel(Event e) {
    int height = (int)(size.height + Y_PADDING);
    int areaHeight = (int)(size.tree.h - size.headerHeight);
    if (areaHeight < height) {
      int current = commonVBar.getSelection();
      int selection = Math.max(0, Math.min(current - e.count * 10, height - areaHeight));
      if (current != selection) {
        commonVBar.setSelection(selection);
        redraw(Area.FULL);
      }
    }

    updateHover(e.x, e.y);
  }

  private void onMouseHorizontalWheel(Event e) {
    double areaWidth, width;
    Slider slider;
    if (size.tree.contains(e.x, e.y)) {
      areaWidth = size.tree.w;
      width = size.treeWidth;
      slider = treeHBar;
    } else {
      areaWidth = size.table.w;
      width = size.tableWidth;
      slider = tableHBar;
    }
    if (areaWidth < width) {
      int current = slider.getSelection();
      int selection = Math.max(0, Math.min(current - e.count * 10, (int)(width - areaWidth)));
      if (current != selection) {
        slider.setSelection(selection);
        redraw(Area.FULL);
      }
    }

    updateHover(e.x, e.y);
  }

  private void onKeyDown(Event e) {
    switch (e.keyCode) {
      case SWT.ARROW_DOWN:
        if (selectedRow < 0) {
          selectRow(0);
        } else if (selectedRow < tree.getNumRows() - 1) {
          selectRow(selectedRow + 1);
        }
        redraw(Area.FULL);
        break;
      case SWT.ARROW_UP:
        if (selectedRow < 0) {
          selectRow(tree.getNumRows() - 1);
        } else if (selectedRow > 0) {
          selectRow(selectedRow - 1);
        }
        redraw(Area.FULL);
        break;
      case SWT.ARROW_RIGHT:
        if (selectedRow >= 0 && selectedRow < tree.getNumRows()) {
          TreeState.Row row = tree.getRow(selectedRow);
          if (row.node.getChildCount() > 0 && !tree.isExpanded(row)) {
            if (tree.toggle(selectedRow, this::initRows)) {
              updateSize(false, selectedRow + 1);
              updateHover();
              redraw(Area.FULL);
            }
          }
        }
        break;
      case SWT.ARROW_LEFT:
        if (selectedRow >= 0 && selectedRow < tree.getNumRows()) {
          TreeState.Row row = tree.getRow(selectedRow);
          if (tree.isExpanded(row)) {
            if (tree.toggle(selectedRow, this::initRows)) {
              updateSize(false, selectedRow + 1);
              updateHover();
              redraw(Area.FULL);
            }
          }
        }
        break;
      case '\r':
      case ' ':
        if (selectedRow >= 0 && selectedRow < tree.getNumRows()) {
          if (tree.toggle(selectedRow, this::initRows)) {
            updateSize(false, selectedRow + 1);
            updateHover();
            redraw(Area.FULL);
          }
        }
      }
  }

  private void search(String text, boolean regex) {
    models.analytics.postInteraction(View.Commands, ClientAction.Search);
    CommandStream.Node parent = models.commands.getData();
    if (parent != null && !text.isEmpty()) {
      if (selectedRow >= 0 && selectedRow < tree.getNumRows()) {
        parent = tree.getRow(selectedRow).node;
      }
      searchController.start().listen(
          MoreFutures.transformAsync(models.commands.search(parent, text, regex),
              r -> models.commands.getTreePath(models.commands.getData(), Lists.newArrayList(),
                  r.getCommandTreeNode().getIndicesList().iterator())),
          new UiCallback<TreePath, TreePath>(this, LOG) {
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
    if (!models.capture.isLoaded()) {
      onCaptureLoadingStart(false);
    } else {
      if (!models.profile.isLoaded()) {
        onProfileLoadingStart();
      }
      if (models.commands.isLoaded()) {
        onCommandsLoaded();
      }
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    tableMessage = Loadable.Message.loading(Messages.LOADING_CAPTURE);
    updateSize(true, 0);
    redraw(Area.FULL);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      tableMessage = error;
      updateSize(true, 0);
      redraw(Area.FULL);
    }
  }

  @Override
  public void onCommandsLoaded() {
    tree.reset(models.commands.getData(), this::initRows);
    updateSize(false, 0);
    redraw(Area.FULL);
  }

  @Override
  public void onCommandsSelected(CommandIndex index) {
    selectionHandler.updateSelectionFromModel(
        () -> models.commands.getTreePath(index.getNode()).get(),
        path -> {
          TreeState.Row current = tree.getRootRow();
          for (int i = 1; i < path.getSegmentCount(); i++) {
            if (!tree.isExpanded(current)) {
              tree.expand(current.index, current, this::initRows);
            }
            int childIdx = ((CommandStream.Node)path.getSegment(i)).getChildIndex();
            if (childIdx < 0) {
              LOG.log(WARNING, "Couldn't find requested selection child " + path.getSegment(i) +
                  " in " + current.node);
              break;
            }
            current = tree.getChildRow(current, childIdx);
          }

          selectedRow = current.index;
          updateSelection(false);
          updateSize(false, 0);
          redraw(Area.FULL);
        });
  }

  @Override
  public void onProfileLoadingStart() {
    tableMessage = Loadable.Message.loading(Messages.LOADING_PROFILE);
    updateSize(true, 0);
    redraw(Area.FULL);
  }

  @Override
  public void onProfileLoaded(Loadable.Message error) {
    if (error != null) {
      tableMessage = error;
    } else {
      // Update the group buttons.
      counterGroupButtons.clear();
      counterGroupButtons.addLabel("Counters:");
      List<Service.ProfilingData.CounterGroup> groups = models.profile.getData().getCounterGroups();
      for (Service.ProfilingData.CounterGroup group : groups) {
        counterGroupButtons.addButton(group.getLabel(), e -> {
          selectCounterGroup(group);
          updateSize(false, 0);
          redraw(Area.FULL);
        });
      }
      counterGroupButtons.addButton("All", e -> {
        selectAllCounters();
        updateSize(false, 0);
        redraw(Area.FULL);
      });
      counterGroupButtons.addButton("Custom", e -> {
        GpuCountersDialog dialog = new GpuCountersDialog(
            getShell(), widgets.theme, getCounterSpecs(), tree.getEnabledMetricIds());
        if (dialog.open() == Window.OK) {
          tree.clearColumns();
          selectCounters(dialog.getSelectedIds());
          updateSize(false, 0);
          redraw(Area.FULL);
        }
      });
      counterGroupButtons.selectButton(0);
      counterGroupButtons.requestLayout();

      if (groups.isEmpty()) {
        selectAllCounters();
      } else {
        selectCounterGroup(groups.get(0));
      }
    }

    updateSize(false, 0);
    redraw(Area.FULL);
  }

  private void selectCounterGroup(Service.ProfilingData.CounterGroup counterGroup) {
    tree.clearColumns();
    models.profile.getData().getGpuPerformance().getMetricsList().stream()
        .filter(m -> isStaticAnalysisCounter(m) || m.getCounterGroupIdsList().contains(counterGroup.getId()))
        .sorted((m1, m2) -> {
          if (isStaticAnalysisCounter(m1) != isStaticAnalysisCounter(m2)) {
            return isStaticAnalysisCounter(m1) ? -1 : 1;
          }
          return Integer.compare(m1.getId(), m2.getId());
        })
        .map(TreeState.Column::new)
        .forEach(tree::addColumn);
  }

  private void selectAllCounters() {
    tree.clearColumns();
    models.profile.getData().getGpuPerformance().getMetricsList().stream()
        .sorted((m1, m2) -> {
          if (isStaticAnalysisCounter(m1) != isStaticAnalysisCounter(m2)) {
            return isStaticAnalysisCounter(m1) ? -1 : 1;
          }
          return Integer.compare(m1.getId(), m2.getId());
        })
        .map(TreeState.Column::new)
        .forEach(tree::addColumn);
  }

  private void selectCounters(Collection<Integer> counterIds) {
    Map<Integer, Service.ProfilingData.GpuCounters.Metric> metrics =
        models.profile.getData().getGpuPerformance().getMetricsList().stream()
          .collect(Collectors.toMap(m -> m.getId(), identity()));
    counterIds.stream()
        .map(metrics::get)
        .filter(Objects::nonNull)
        .map(TreeState.Column::new)
        .forEach(tree::addColumn);
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

  protected void updateSize(boolean onlyLoadingMessage, int startRow) {
    double height = 0;
    if (!models.profile.isLoaded()) {
      size.message = widgets.loading.measure(context, tableMessage);
      size.tableWidth = Math.ceil(size.message.w);
      height = size.message.h;
      if (onlyLoadingMessage) {
        size.height = Math.max(size.height, height);
        updateScrollbars();
        return;
      }

      // Default header height based on a random/typical string.
      size.headerHeight = context.measure(Style.Bold, "Agi").h + HEADER_MARGIN;
    } else if (onlyLoadingMessage) {
      updateScrollbars();
      return;
    } else {
      if (size.columnX.length < tree.getNumColumns() + 1) {
        size.columnX = new double[tree.getNumColumns() + 1];
      }
      double width = 0, headerHeight = 0;
      for (int i = 0; i < tree.getNumColumns(); i++) {
        size.columnX[i] = width;
        Size column = context.measure(Style.Bold, tree.getColumn(i).metric.getName());
        width += Math.ceil(Math.max(100, column.w) + COLUMN_SPACING);
        headerHeight = Math.max(headerHeight, column.h);
      }
      size.columnX[tree.getNumColumns()] = width;
      size.tableWidth = width;
      size.headerHeight = Math.ceil(headerHeight + HEADER_MARGIN);
    }

    double y = startRow == 0 ? 0 : size.rowY[startRow], width = 0;
    if (size.rowY.length < tree.getNumRows() + 1) {
      size.rowY = Arrays.copyOf(size.rowY, tree.getNumRows() + 1);
    }
    Size loadingSize = context.measure(Fonts.Style.Normal, "Loading...");
    for (int i = startRow; i < tree.getNumRows(); i++) {
      TreeState.Row row = tree.getRow(i);
      size.rowY[i] = y;
      Size text = (row.size != null) ? row.size : loadingSize;
      width = Math.max(width, row.depth * TREE_INDENT + CARRET_X_SIZE + IMAGE_SIZE + IMAGE_PADDING +
          text.w + COLUMN_SPACING);
      y += Math.ceil(Math.max(text.h, IMAGE_SIZE) + ROW_SPACING);
    }
    size.rowY[tree.getNumRows()] = y++;

    height = Math.max(height, y);
    size.treeWidth = Math.ceil(width + COLUMN_SPACING);
    size.height = height;

    updateScrollbars();
  }

  private void updateScrollbars() {
    int treeWidth = (int)size.treeWidth;
    int tableWidth = (int)size.tableWidth;
    if (size.tree.w >= treeWidth) {
      treeHBar.setValues(0, 0, treeWidth, treeWidth, 1, 10);
    } else {
      int selection = Math.min(treeHBar.getSelection(), (int)(treeWidth - size.tree.w));
      treeHBar.setValues(selection, 0, treeWidth, (int)size.tree.w, 1, 10);
    }
    if (size.table.w >= tableWidth) {
      tableHBar.setValues(0, 0, tableWidth, tableWidth, 1, 10);
    } else {
      int selection = Math.min(tableHBar.getSelection(), (int)(tableWidth - size.table.w));
      tableHBar.setValues(selection, 0, tableWidth, (int)size.table.w, 1, 10);
    }

    int height = (int)(size.height + Y_PADDING);
    int areaHeight = (int)(size.tree.h - size.headerHeight);
    if (areaHeight >= height) {
      commonVBar.setValues(0, 0, height, height, 1, 10);
    } else {
      int selection = Math.min(commonVBar.getSelection(), height - areaHeight);
      commonVBar.setValues(selection, 0, height, areaHeight, 1, 10);
    }
  }

  private void redraw(Area area) {
    if (!area.isEmpty()) {
      Rectangle r = getClientArea();
      if (area != Area.FULL) {
        r.x = Math.max(0, Math.min(r.width - 1, (int)Math.floor(area.x)));
        r.y = Math.max(0, Math.min(r.height - 1, (int)Math.floor(area.y)));
        r.width = Math.max(0, Math.min(r.width - r.x, (int)Math.ceil(area.w + (area.x - r.x))));
        r.height = Math.max(0, Math.min(r.height - r.y, (int)Math.ceil(area.h + (area.y - r.y))));
      }
      redraw(r.x, r.y, r.width, r.height, false);
    }
  }

  private void redrawTree(Area area) {
    area = area.translate(
        size.tree.x - treeHBar.getSelection(), size.tree.y - commonVBar.getSelection());
    redraw(area.intersect(size.tree));
  }

  private void redrawRow(int row) {
    if (row >= 0) {
      redraw(Area.FULL); // TODO
    }
  }

  private void drawTree(RenderContext ctx) {
    ctx.setBackgroundColor(SWT.COLOR_LIST_BACKGROUND);
    ctx.fillRect(0, 0, size.tree.w, size.tree.h);

    drawTreeHeader(ctx);
    ctx.withClipAndTranslation(0, size.headerHeight, size.tree.w, size.height,
        -treeHBar.getSelection(), size.headerHeight - commonVBar.getSelection(), () -> {
      drawTreeGrid(ctx);
      drawTreeRows(ctx);
    });
  }

  private void drawTreeHeader(RenderContext ctx) {
    ctx.setForegroundColor(SWT.COLOR_WIDGET_BACKGROUND);
    ctx.drawLine(0, size.headerHeight - 1, size.tree.w, size.headerHeight - 1);
  }

  private void drawTreeGrid(RenderContext ctx) {
    ctx.setForegroundColor(SWT.COLOR_WIDGET_BACKGROUND);

    double endX = Math.max(size.treeWidth, size.tree.w);
    Area clip = ctx.getClip();
    double endY = clip.y + clip.h;
    for (int i = size.getRow(clip.y); i <= tree.getNumRows() && size.rowY[i] < endY; i++) {
      if (i == selectedRow) {
        ctx.setBackgroundColor(SWT.COLOR_LIST_SELECTION);
        ctx.fillRect(0, size.rowY[i] + 1, endX, size.rowY[i + 1] - size.rowY[i] - 1);
      } else if (i == hoveredRow) {
        ctx.setBackgroundColor(SWT.COLOR_WIDGET_LIGHT_SHADOW);
        ctx.fillRect(0, size.rowY[i] + 1, endX, size.rowY[i + 1] - size.rowY[i] - 1);
      }
      ctx.drawLine(0, size.rowY[i], endX, size.rowY[i]);
    }
  }

  private void drawTreeRows(RenderContext ctx) {
    ctx.setForegroundColor(SWT.COLOR_WIDGET_FOREGROUND);
    ctx.setBackgroundColor(SWT.COLOR_WIDGET_FOREGROUND);

    Area clip = ctx.getClip();
    double endY = clip.y + clip.h;
    for (int i = size.getRow(clip.y); i < tree.getNumRows() && size.rowY[i] < endY; i++) {
      TreeState.Row row = tree.getRow(i);
      double x = row.getCarretX(), y = size.rowY[i] + ROW_SPACING / 2;

      if (row.node.getChildCount() > 0 && x + CARRET_X_SIZE > clip.x && x < clip.x + clip.w) {
        if (tree.isExpanded(row)) {
          double cx = x, cy = y + CARRET_Y_PADDING + (CARRET_SIZE_LONG - CARRET_SIZE_SHORT) / 2;
          ctx.path(path -> {
            path.moveTo(cx, cy);
            path.lineTo(cx + CARRET_SIZE_LONG / 2, cy + CARRET_SIZE_SHORT);
            path.lineTo(cx + CARRET_SIZE_LONG, cy);
            ctx.drawPath(path, CARRET_STROKE);
          });
        } else {
          double cx = x + (CARRET_SIZE_LONG - CARRET_SIZE_SHORT) / 2, cy = y + CARRET_Y_PADDING;
          ctx.path(path -> {
            path.moveTo(cx, cy);
            path.lineTo(cx + CARRET_SIZE_SHORT, cy + CARRET_SIZE_LONG / 2);
            path.lineTo(cx, cy + CARRET_SIZE_LONG);
            ctx.drawPath(path, CARRET_STROKE);
          });
        }
      }
      x += CARRET_X_SIZE;

      double imageX = x + IMAGE_PADDING / 2;
      if (shouldShowImage(row.node) && imageX + IMAGE_SIZE > clip.x && imageX < clip.x + clip.w) {
        ImageProvider.ImageInfo info = images.getImage(row.node);
        if (info.hasImage()) {
          Image image = info.isLoading() ? widgets.loading.getCurrentSmallFrame() : info.imageSmall;
          Rectangle bounds = image.getBounds();
          ctx.drawImage(image,
              imageX + (IMAGE_SIZE - bounds.width) / 2, y + (IMAGE_SIZE - bounds.height) / 2);
          if (info.isLoading()) {
            widgets.loading.scheduleForRedraw(() -> {
              if (tree.validate(row)) {
                redrawTree(new Area(imageX, size.headerHeight + y, IMAGE_SIZE, IMAGE_SIZE));
              }
            });
          }
        }
      }
      x += IMAGE_SIZE + IMAGE_PADDING;

      if (x < clip.x + clip.w) {
        if (row.hasLabel()) {
          if (i == selectedRow) {
            ctx.setForegroundColor(SWT.COLOR_LIST_SELECTION_TEXT);
            row.drawLabel(ctx, x, y, true);
            ctx.setForegroundColor(SWT.COLOR_WIDGET_FOREGROUND);
          } else {
            row.drawLabel(ctx, x, y, false);
          }
        } else {
          if (i == selectedRow) {
            ctx.setForegroundColor(SWT.COLOR_LIST_SELECTION_TEXT);
            ctx.drawText(Fonts.Style.Normal, "Loading...", x, y);
            ctx.setForegroundColor(SWT.COLOR_WIDGET_FOREGROUND);
          } else {
            ctx.drawText(Fonts.Style.Normal, "Loading...", x, y);
          }
          row.load(models.commands, () -> {
            if (tree.validate(row)) {
              updateRowLabel(row);
              Service.CommandTreeNode data = row.node.getData();
              if (data.getNumChildren() > 0 && data.getExpandByDefault() && !tree.isExpanded(row)) {
                tree.expand(row.index, row, this::initRows);
              }
              updateSize(false, row.index);
              updateHover();
              redraw(Area.FULL);
            }
          });
        }
      }
    }
  }

  private void drawTable(RenderContext ctx) {
    if (!models.profile.isLoaded()) {
      ctx.setBackgroundColor(SWT.COLOR_WIDGET_BACKGROUND);
      ctx.fillRect(0, 0, size.table.w, size.table.h);

      ctx.withTranslation(-tableHBar.getSelection(),
          (size.message.h < size.table.h) ? 0 : -commonVBar.getSelection(), () -> {
        ctx.setForegroundColor(SWT.COLOR_WIDGET_FOREGROUND);
        widgets.loading.paint(ctx, 0, 0,
            Math.max(size.table.w, size.message.w), Math.max(size.table.h, size.message.h),
            tableMessage);
        if (tableMessage.type == Loadable.MessageType.Loading) {
          widgets.loading.scheduleForRedraw(loadingRepainter);
        }
      });
      return;
    }

    ctx.setBackgroundColor(SWT.COLOR_LIST_BACKGROUND);
    ctx.fillRect(0, 0, size.table.w, size.table.h);
    ctx.withClipAndTranslation(0, 0, size.table.w, size.headerHeight, -tableHBar.getSelection(), 0, () -> {
      drawTableHeader(ctx);
    });
    ctx.withClipAndTranslation(0, size.headerHeight, size.table.w, size.height,
        -tableHBar.getSelection(), size.headerHeight - commonVBar.getSelection(), () -> {
      drawTableGrid(ctx);
      drawTableRows(ctx);
    });
  }

  private void drawTableHeader(RenderContext ctx) {
    ctx.setForegroundColor(SWT.COLOR_LIST_FOREGROUND);

    Area clip = ctx.getClip();
    double end = clip.x + clip.w;
    for (int i = size.getColumn(clip.x); i < tree.getNumColumns() && size.columnX[i] < end; i++) {
      ctx.setForegroundColor(SWT.COLOR_LIST_FOREGROUND);
      ctx.drawText(Fonts.Style.Normal, tree.getColumn(i).metric.getName(),
          size.columnX[i] + COLUMN_SPACING / 2, HEADER_MARGIN / 2);
      if (i > 0) {
        ctx.setForegroundColor(SWT.COLOR_WIDGET_BACKGROUND);
        ctx.drawLine(size.columnX[i], 0, size.columnX[i], size.headerHeight);
      }
    }
    ctx.setForegroundColor(SWT.COLOR_WIDGET_BACKGROUND);
    ctx.drawLine(0, size.headerHeight - 1, size.tableWidth, size.headerHeight - 1);
  }

  private void drawTableGrid(RenderContext ctx) {
    ctx.setForegroundColor(SWT.COLOR_WIDGET_BACKGROUND);

    ctx.trace("y-grid", () -> {
      Area clip = ctx.getClip();
      double endY = clip.y + clip.h;
      for (int i = size.getRow(clip.y); i <= tree.getNumRows() && size.rowY[i] < endY; i++) {
        if (i == selectedRow) {
          ctx.setBackgroundColor(SWT.COLOR_LIST_SELECTION);
          ctx.fillRect(0, size.rowY[i] + 1, size.tableWidth, size.rowY[i + 1] - size.rowY[i] - 1);
        } else if (i == hoveredRow) {
          ctx.setBackgroundColor(SWT.COLOR_WIDGET_LIGHT_SHADOW);
          ctx.fillRect(0, size.rowY[i] + 1, size.tableWidth, size.rowY[i + 1] - size.rowY[i] - 1);
        }
        ctx.drawLine(0, size.rowY[i], size.tableWidth, size.rowY[i]);
      }
    });

    ctx.trace("x-grid", () -> {
      Area clip = ctx.getClip();
      double endX = clip.x + clip.w;
      for (int i = Math.max(1, size.getColumn(clip.x));
          i <= tree.getNumColumns() && size.columnX[i] < endX; i++) {
        ctx.drawLine(size.columnX[i], 0, size.columnX[i], size.headerHeight + size.height);
      }
    });
  }

  private void drawTableRows(RenderContext ctx) {
    ctx.setForegroundColor(SWT.COLOR_WIDGET_FOREGROUND);
    Area clip = ctx.getClip();
    double endX = clip.x + clip.w, endY = clip.y + clip.h;
    int startColumn = size.getColumn(clip.x);
    for (int i = size.getRow(clip.y); i < tree.getNumRows() && size.rowY[i] < endY; i++) {
      TreeState.Row row = tree.getRow(i);
      if (row.node.getData() != null) {
        if (i == selectedRow) {
          ctx.setForegroundColor(SWT.COLOR_LIST_SELECTION_TEXT);
        }
        for (int j = startColumn; j < tree.getNumColumns() && size.columnX[j] < endX; j++) {
          double x = size.columnX[j], w = size.columnX[j + 1] - x;
          String text = getValue(row, j).format(true);
          Size textSize = ctx.measure(Fonts.Style.Normal, text);
          ctx.drawText(Fonts.Style.Normal, text,
              x + (w - textSize.w - COLUMN_SPACING / 2), size.rowY[i] + ROW_SPACING / 2);
        }
        if (i == selectedRow) {
          ctx.setForegroundColor(SWT.COLOR_WIDGET_FOREGROUND);
        }
      }
    }
  }

  private Profile.PerfNode.Value getValue(TreeState.Row row, int column) {
    Profile.PerfNode perf = models.profile.getData().getPerfNode(row.node.getData());
    if (perf == null) {
      return Profile.PerfNode.Value.NULL;
    }
    return perf.getPerf(tree.getColumn(column).metric);
  }

  private void drawImagePreview(RenderContext ctx) {
    ImageProvider.ImageInfo info = images.getImage(hoveredNode);
    if (!info.isLoading() && !info.hasImage()) {
      showImagePreview = false;
      return;
    }
    Image image = info.isLoading() ?  widgets.loading.getCurrentFrame() : info.imageLarge;
    TreeState.Row row = tree.getRow(hoveredRow);
    double y = size.tree.y + size.headerHeight +
        (size.rowY[hoveredRow] + size.rowY[hoveredRow + 1]) / 2 - commonVBar.getSelection();
    Tooltip.forImage(image, verticallyCenteredAndConstrained(
        row.getImageX() + IMAGE_SIZE + IMAGE_PADDING, y, size.tree.y, size.tree.h)).render(ctx);

    if (info.isLoading()) {
      widgets.loading.scheduleForRedraw(previewRepainter);
    }
  }

  private void initRows(List<TreeState.Row> rows) {
    for (TreeState.Row row : rows) {
      updateRowLabel(row);
    }
  }

  private void updateRowLabel(TreeState.Row row) {
    if (row.isLoaded()) {
      Service.CommandTreeNode data = row.node.getData();
      boolean hovered = row.index == hoveredRow;
      LinkableStyledString label = hovered ? LinkableStyledString.create(widgets.theme) :
        LinkableStyledString.ignoring(widgets.theme);
      Follower.Prefetcher<String> prefetcher = hovered ? lastPrefetcher : nullPrefetcher();

      if (data.getGroup().isEmpty() && data.hasCommands()) {
        Formatter.format(row.node.getCommand(), models.constants::getConstants,
            prefetcher::canFollow, label, getCommandStyle(data, label), label.identifierStyle());
      } else {
        label.append(data.getGroup(), getCommandStyle(data, label));
        long count = data.getNumCommands();
        label.append(
            " (" + count + " command" + (count != 1 ? "s" : "") + ")", label.structureStyle());
      }

      label.endLink(); // make sure links are closed.
      row.updateLabel(context, label);
    }
  }

  private Formatter.Style getCommandStyle(Service.CommandTreeNode node, StylingString string) {
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

  private void selectRow(int row) {
    this.selectedRow = row;
    notifyListeners(SWT.Selection, new Event());
  }

  protected void updateSelection(boolean updateModel) {
    if (selectedRow >= 0 && selectedRow < tree.getNumRows()) {
      TreeState.Row row = tree.getRow(selectedRow);
      CommandIndex index = row.node.getIndex();
      if (updateModel) {
        if (index == null) {
          models.commands.load(row.node, () -> updateSelection(true));
        } else {
          models.commands.selectCommands(index, false);
          commandIdx.setText(COMMAND_INDEX_DSCRP + row.node.getIndexString());
          commandIdx.requestLayout();
        }
      }
      models.profile.linkCommandToGpuGroup(row.node.getCommandStart());
    } else {
     commandIdx.setText(COMMAND_INDEX_DSCRP);
     commandIdx.requestLayout();
    }
  }

  private void updateHover() {
    Display disp = getDisplay();
    Point cursor = disp.map(null, this, disp.getCursorLocation());
    updateHover(cursor.x, cursor.y);
  }

  private void updateHover(int ex, int ey) {
    int newHover = -1;
    Tooltip newColumnTooltip = null;
    CommandStream.Node newHoveredNode = null;
    Tooltip newValueTooltip = null;

    double y = ey - size.tree.y;
    if (y < 0) {
      // Do nothing.
    } else if (y < size.headerHeight) {
      double x = ex - size.table.x + tableHBar.getSelection();
      int column = (x < 0) ? -1 : size.getColumn(x);
      if (column >= 0 && column < tree.getNumColumns()) {
        newColumnTooltip = Tooltip.forText(context, tree.getColumn(column).getTooltip(),
            horizontallyCenteredAndConstrained(
                ex, size.table.y + size.headerHeight, size.table.x, size.table.w));
      }
    } else {
      y += commonVBar.getSelection() - size.headerHeight;
      int rowIdx = size.getRow(y);
      if (rowIdx >= 0 && rowIdx < tree.getNumRows()) {
        newHover = rowIdx;
        TreeState.Row row = tree.getRow(rowIdx);
        if (ex >= size.tree.x && ex < size.tree.x + size.tree.w) {
          double x = ex - size.tree.x + treeHBar.getSelection() - row.getImageX();
          if (x >= 0 && x < IMAGE_SIZE && shouldShowImage(row.node)) {
            newHoveredNode = row.node;
          }
        } else if (ex >= size.table.x && ex < size.table.x + size.table.w) {
          double x = ex - size.table.x + tableHBar.getSelection();
          int column = size.getColumn(x);
          if (column >= 0 && column < tree.getNumColumns()) {
            Profile.PerfNode.Value value = getValue(row, column);
            if (value.hasInterval()) {
              newValueTooltip = Tooltip.forFormattedText(context, value.format(false),
                  standardTooltip(ex + TOOLTIP_OFFSET, ey + TOOLTIP_OFFSET, size.table));
            }
          }
        }
      }
    }

    if (newHover != hoveredRow) {
      hoveredRow = newHover;
      checkHoveredRow();
      redraw(Area.FULL);
    }
    if (newColumnTooltip != columnTooltip) {
      columnTooltip = newColumnTooltip;
      redraw(Area.FULL);
    }
    if (newValueTooltip != valueTooltip) {
      valueTooltip = newValueTooltip;
      redraw(Area.FULL);
    }
    if (newHoveredNode != hoveredNode) {
      lastScheduledFuture.cancel(true);
      showImagePreview = false;

      hoveredNode = newHoveredNode;
      if (hoveredNode != null) {
        CommandStream.Node scheduledNode = hoveredNode;
        lastScheduledFuture = Scheduler.EXECUTOR.schedule(() -> scheduleIfNotDisposed(this, () -> {
          if (hoveredNode == scheduledNode) {
            showImagePreview = true;
            redraw(Area.FULL);
          }
        }), PREVIEW_HOVER_DELAY_MS, TimeUnit.MILLISECONDS);
      } else {
        redraw(Area.FULL);
      }
    }

    updateCursor(ex - size.tree.x + treeHBar.getSelection());
  }

  private void checkHoveredRow() {
    if (rowMarkedAsHovered != null) {
      if (!tree.validate(rowMarkedAsHovered)) {
        rowMarkedAsHovered = null;
        lastPrefetcher.cancel();
        lastPrefetcher = nullPrefetcher();
      } else if (rowMarkedAsHovered.index != hoveredRow) {
        updateRowLabel(rowMarkedAsHovered);
        redrawRow(rowMarkedAsHovered.index);
        rowMarkedAsHovered = null;
        lastPrefetcher.cancel();
        lastPrefetcher = nullPrefetcher();
      }
    }

    if (hoveredRow >= 0 && rowMarkedAsHovered == null) {
      TreeState.Row newHoveredRow = tree.getRow(hoveredRow);
      rowMarkedAsHovered = newHoveredRow;
      lastPrefetcher = models.follower.prepare(rowMarkedAsHovered.node, () -> {
        scheduleIfNotDisposed(this, () -> {
          updateRowLabel(newHoveredRow);
          redrawRow(newHoveredRow.index);
        });
      });
      updateRowLabel(rowMarkedAsHovered);
      redrawRow(hoveredRow);
    }
  }

  private Path.Any getFollow(double x) {
    Path.Any result = null;
    if (hoveredRow >= 0) {
      TreeState.Row row = tree.getRow(hoveredRow);
      if (row.size != null) {
        result = row.getFollow(context, x);
      }
    }
    return result;
  }

  private boolean shouldShowImage(CommandStream.Node node) {
    return models.images.isReady() &&
        node.getData() != null && !node.getData().getGroup().isEmpty();
  }

  private void updateCursor(double x) {
    setCursor(getFollow(x) != null ? getDisplay().getSystemCursor(SWT.CURSOR_HAND) : null);
  }

  @Override
  public boolean setFocus() {
    return forceFocus();
  }

  private static class SizeData {
    public Area tree = Area.NONE;
    public Area table = Area.NONE;

    public Size message = Size.ZERO;
    public double treeWidth = 0, tableWidth = 0;
    public double headerHeight = 0;
    public double height = 0;

    public double[] columnX = new double[0];
    public double[] rowY = new double[0];

    public SizeData() {
    }

    public int getColumn(double x) {
      int first = Arrays.binarySearch(columnX, x);
      return (first < 0) ? -first - 2 : first;
    }

    public int getRow(double y) {
      int first = Arrays.binarySearch(rowY, y);
      return (first < 0) ? -first - 2 : first;
    }
  }

  private static class TreeState {
    private final List<Row> rows = Lists.newArrayList();
    private final Set<Row> expanded = Sets.newIdentityHashSet();
    private final List<Column> columns = Lists.newArrayList();
    private Row rootRow;

    public TreeState() {
    }

    public void reset(CommandStream.Node root, Consumer<List<Row>> onNewRows) {
      for (Row row : rows) {
        row.dispose();
      }
      rows.clear();
      expanded.clear();

      rootRow = new Row(root);
      expanded.add(rootRow);
      expandChildren(rootRow, rows);
      onNewRows.accept(rows);
    }

    public void dispose() {
      for (Row row : rows) {
        row.dispose();
      }
      rows.clear();
      expanded.clear();
      columns.clear();
    }

    public int getNumRows() {
      return rows.size();
    }

    public Row getRow(int idx) {
      return rows.get(idx);
    }

    public Row getChildRow(Row parent, int idx) {
      Row last = parent;
      for (int i = 0; i < idx; i++) {
        last = rows.get(last.index + 1);
        while (isExpanded(last)) {
          last = getChildRow(last, last.node.getChildCount() - 1);
        }
      }
      return rows.get(last.index + 1);
    }

    public Row getRootRow() {
      return rootRow;
    }

    public boolean isExpanded(Row row) {
      return expanded.contains(row);
    }

    public boolean toggle(int idx, Consumer<List<Row>> onNewRows) {
      Row row = rows.get(idx);
      if (expanded.contains(row)) {
        collapse(idx, row);
        return true;
      } else if (row.node.getChildCount() > 0) {
        expand(idx, row, onNewRows);
        return true;
      }
      return false;
    }

    protected void expand(int idx, Row parent, Consumer<List<Row>> onNewRows) {
      expanded.add(parent);
      List<Row> newRows = expandChildren(parent, Lists.newArrayList());
      rows.addAll(idx + 1, newRows);
      for (int i = idx + 1 + newRows.size(); i < rows.size(); i++) {
        rows.get(i).index = i;
      }
      onNewRows.accept(newRows);
    }

    private List<Row> expandChildren(Row parent, List<Row> newRows) {
      int idx = 1;
      for (CommandStream.Node node : parent.node.getChildren()) {
        Row child = new Row(node, parent.depth + 1, parent.index + idx);
        newRows.add(child);
        Service.CommandTreeNode data = node.getData();
        if (data != null && data.getNumChildren() > 0 && data.getExpandByDefault()) {
          expanded.add(child);
          int oldSize = newRows.size();
          expandChildren(child, newRows);
          idx += newRows.size() - oldSize + 1;
        } else {
          idx++;
        }
      }
      return newRows;
    }

    private void collapse(int idx, Row parent) {
      expanded.remove(parent);
      Row last = collapseChildren(parent);
      List<Row> remove = rows.subList(idx + 1, last.index + 1);
      for (Row row : remove) {
        row.dispose();
      }
      remove.clear();
      for (int i = idx + 1; i < rows.size(); i++) {
        rows.get(i).index = i;
      }
    }

    // Removes any expanded children from the expanded set, and returns the last child.
    private Row collapseChildren(Row parent) {
      CommandStream.Node[] children = parent.node.getChildren();
      Row last = parent;
      for (int i = 0; i < children.length; i++) {
        Row child = rows.get(last.index + 1);
        if (expanded.contains(child)) {
          expanded.remove(child);
          last = collapseChildren(child);
        } else {
          last = child;
        }
      }
      return last;
    }

    public boolean validate(Row row) {
      return row.index < rows.size() && rows.get(row.index) == row;
    }

    public int getNumColumns() {
      return columns.size();
    }

    public Column getColumn(int idx) {
      return columns.get(idx);
    }

    public void clearColumns() {
      columns.clear();
    }

    public void addColumn(Column column) {
      columns.add(column);
    }

    public List<Integer> getEnabledMetricIds() {
      return columns.stream()
          .mapToInt(c -> c.metric.getId())
          .boxed()
          .collect(toList());
    }

    public static class Row {
      public final CommandStream.Node node;
      public final int depth;
      public int index;
      private boolean loading;
      private LinkableStyledString label;
      private TextLayout layout;
      public Size size;

      public Row(CommandStream.Node root) {
        this(root, -1, -1);
      }

      public Row(CommandStream.Node node, int depth, int index) {
        this.node = node;
        this.depth = depth;
        this.index = index;
        this.loading = false;
      }

      public boolean isLoaded() {
        Service.CommandTreeNode data = node.getData();
        if (data == null) {
          return false;
        }
        return !data.getGroup().isEmpty() || !data.hasCommands() || node.getCommand() != null;
      }

      public void load(CommandStream model, Runnable callback) {
        if (!loading) {
          loading = true;
          model.load(node, callback);
        }
      }

      public void updateLabel(RenderContext.Global ctx, LinkableStyledString newLabel) {
        this.label = newLabel;
        updateLayout(ctx, false);
        size = ctx.measure(layout);
      }

      private void updateLayout(Fonts.FontContext ctx, boolean ignoreColors) {
        if (layout == null) {
          layout = ctx.newTextLayout();
        }
        layout.setText(label.getString().getString());
        for (StyleRange range : label.getString().getStyleRanges()) {
          if (ignoreColors) {
            range.foreground = range.background = null;
          }
          ctx.applyStyle(layout, range);
        }
      }

      public boolean hasLabel() {
        return label != null;
      }

      public void drawLabel(RenderContext ctx, double x, double y, boolean ignoreColors) {
        updateLayout(ctx, ignoreColors);
        ctx.drawText(layout, x, y);
      }

      public Path.Any getFollow(Fonts.TextMeasurer tm, double x) {
        x -= getTextX();
        if (x < 0 || x >= size.w) {
          return null;
        }
        int offset = tm.getOffset(layout, x, 5);
        return (Path.Any)label.getLinkTarget(offset);
      }

      public double getCarretX() {
        return COLUMN_SPACING / 2 + depth * TREE_INDENT;
      }

      public double getImageX() {
        return getCarretX() + CARRET_X_SIZE + IMAGE_PADDING / 2;
      }

      public double getTextX() {
        return getCarretX() + CARRET_X_SIZE + IMAGE_SIZE + IMAGE_PADDING;
      }

      public void dispose() {
        if (layout != null) {
          layout.dispose();
        }
      }
    }

    public static class Column {
      public final Service.ProfilingData.GpuCounters.Metric metric;

      public Column(Service.ProfilingData.GpuCounters.Metric metric) {
        this.metric = metric;
      }

      public String getTooltip() {
        StringBuilder sb = new StringBuilder().append("\\b").append(metric.getName());
        if (!metric.getDescription().isEmpty()) {
          sb.append("\n").append(metric.getDescription());
        }
        return sb.toString();
      }
    }
  }

  private static class ImageProvider {
    protected final Widget owner;
    private final ImagesModel model;
    protected final Map<CommandStream.Node, ImageInfo> images = Maps.newIdentityHashMap();

    public ImageProvider(Widget owner, ImagesModel model) {
      this.owner = owner;
      this.model = model;
    }

    private ListenableFuture<ImageData> loadImage(CommandStream.Node node) {
      return noAlpha(model.getThumbnail(
          node.getPath(Path.CommandTreeNode.newBuilder()).build(), THUMB_SIZE, i -> { /*noop*/ }));
    }

    public ImageInfo getImage(CommandStream.Node node) {
      ImageInfo image = images.get(node);
      if (image == null) {
        image = ImageInfo.LOADING;
        images.put(node, image);

        Rpc.listen(loadImage(node), new UiCallback<ImageData, ImageData>(owner, LOG) {
          @Override
          protected ImageData onRpcThread(Result<ImageData> result) {
            try {
              return result.get();
            } catch (DataUnavailableException e) {
              return null;
            } catch (RpcException | ExecutionException e) {
              if (!owner.isDisposed()) {
                throttleLogRpcError(LOG, "Failed to load image", e);
              }
              return null;
            }
          }

          @Override
          protected void onUiThread(ImageData result) {
            ImageInfo info = ImageInfo.NONE;
            if (result != null) {
              Image large = createNonScaledImage(owner.getDisplay(), result);
              Image small = createNonScaledImage(owner.getDisplay(), scaleImage(result, IMAGE_SIZE));
              info = new ImageInfo(large, small);
            }

            images.replace(node, info);
          }
        });
      }
      return image;
    }

    public void dispose() {
      for (ImageInfo info : images.values()) {
        info.dispose();
      }
      images.clear();
    }

    private static class ImageInfo {
      public static final ImageInfo LOADING = new ImageInfo(null, null);
      public static final ImageInfo NONE = new ImageInfo(null, null);

      public final Image imageLarge;
      public final Image imageSmall;

      public ImageInfo(Image imageLarge, Image imageSmall) {
        this.imageLarge = imageLarge;
        this.imageSmall = imageSmall;
      }

      public boolean hasImage() {
        return this != NONE;
      }

      public boolean isLoading() {
        return this == LOADING;
      }

      public void dispose() {
        if (imageLarge != null) {
          imageLarge.dispose();
        }
        if (imageSmall != null) {
          imageSmall.dispose();
        }
      }
    }
  }

  private static class MatchingRowsLayout extends Layout {
    private static final int LEFT = 0;
    private static final int SASH = 1;
    private static final int RIGHT = 2;
    private static final int LEFT_H_SLIDER = 3;
    private static final int RIGHT_H_SLIDER = 4;
    private static final int COMMON_V_SLIDER = 5;
    private static final int FOOTER = 6;

    private SizeData size;
    private int sliderWidth = 0, sliderHeight = 0;
    private int footerHeight = 0;
    private double ratio;

    public MatchingRowsLayout(SizeData size, double ratio) {
      this.size = size;
      this.ratio = ratio;
    }

    @Override
    protected boolean flushCache(Control control) {
      sliderWidth = sliderHeight = 0;
      footerHeight = 0;
      return true;
    }

    @Override
    protected Point computeSize(Composite composite, int wHint, int hHint, boolean flushCache) {
      Control[] children = composite.getChildren();
      Point left = children[LEFT].computeSize(SWT.DEFAULT, hHint, flushCache);
      Point right = children[RIGHT].computeSize(SWT.DEFAULT, hHint, flushCache);
      Point sliderH = children[LEFT_H_SLIDER].computeSize(SWT.DEFAULT, SWT.DEFAULT);
      Point sliderV = children[COMMON_V_SLIDER].computeSize(SWT.DEFAULT, SWT.DEFAULT);
      Point footer = children[FOOTER].computeSize(wHint, hHint, flushCache);
      sliderWidth = sliderV.x;
      sliderHeight = sliderH.y;
      footerHeight = footer.y;

      int width = wHint;
      if (wHint == SWT.DEFAULT) {
        width = Math.max(footer.x, left.x + SASH_WIDTH + right.x + sliderV.x);
      }
      int height = hHint;
      if (hHint == SWT.DEFAULT) {
        height = Math.max(left.y, right.y) + sliderH.y + footer.y;
      }
      return new Point(width, height);
    }

    @Override
    protected void layout(Composite composite, boolean flushCache) {
      Rectangle area = composite.getClientArea();
      Control[] children = composite.getChildren();
      Point sliderH = children[LEFT_H_SLIDER].computeSize(SWT.DEFAULT, SWT.DEFAULT);
      Point sliderV = children[COMMON_V_SLIDER].computeSize(SWT.DEFAULT, SWT.DEFAULT);
      Point footer = children[FOOTER].computeSize(area.width, SWT.DEFAULT);
      sliderWidth = sliderV.x;
      sliderHeight = sliderH.y;
      footerHeight = footer.y;

      int width = area.width - sliderWidth - SASH_WIDTH;
      int leftWidth = Math.max(0, (int)(ratio * width));
      int rightWidth = Math.max(0, width - leftWidth);
      Point left = children[0].computeSize(leftWidth, SWT.DEFAULT, flushCache);
      Point right = children[2].computeSize(rightWidth + sliderWidth, SWT.DEFAULT, flushCache);
      int height = area.height - sliderHeight - footerHeight;
      int sashHeight = Math.max(0, area.height - footerHeight);
      int topHeight = Math.max(left.y, right.y);
      int vBarY = topHeight + (int)size.headerHeight;

      size.tree = new Area(0, topHeight, leftWidth, Math.max(0, height - topHeight));
      size.table =
          new Area(leftWidth + SASH_WIDTH, topHeight, rightWidth, Math.max(0, height - topHeight));

      if (width <= 0) {
        int x = Math.max(0, area.width - sliderWidth);
        int w = Math.min(area.width, sliderWidth);
        children[LEFT].setBounds(0, 0, 0, 0);
        children[SASH].setBounds(0, 0, 0, 0);
        children[RIGHT].setBounds(x, 0, w, Math.min(topHeight, Math.max(0, height)));
        children[LEFT_H_SLIDER].setBounds(0, 0, 0, 0);
        children[RIGHT_H_SLIDER].setBounds(0, 0, 0, 0);
        children[COMMON_V_SLIDER].setBounds(x, vBarY, w, Math.max(0, height - vBarY));
      } else if (height <= 0) {
        children[LEFT].setBounds(0, 0, leftWidth, 0);
        children[SASH].setBounds(leftWidth, 0, SASH_WIDTH, sashHeight);
        children[RIGHT].setBounds(leftWidth + SASH_WIDTH, 0, rightWidth + sliderWidth, 0);
        children[LEFT_H_SLIDER].setBounds(0, 0, leftWidth, sashHeight);
        children[RIGHT_H_SLIDER].setBounds(leftWidth + SASH_WIDTH, 0, rightWidth, sashHeight);
        children[COMMON_V_SLIDER].setBounds(0, 0, 0, 0);
      } else {
        int h = Math.min(topHeight, height);
        children[LEFT].setBounds(0, 0, leftWidth, h);
        children[SASH].setBounds(leftWidth, 0, SASH_WIDTH, sashHeight);
        children[RIGHT].setBounds(leftWidth + SASH_WIDTH, 0, rightWidth + sliderWidth, h);
        children[LEFT_H_SLIDER].setBounds(0, height, leftWidth, sliderHeight);
        children[RIGHT_H_SLIDER].setBounds(leftWidth + SASH_WIDTH, height, rightWidth, sliderHeight);
        children[COMMON_V_SLIDER].setBounds(
            area.width - sliderWidth, vBarY, sliderWidth, Math.max(0, height - vBarY));
      }
      children[FOOTER].setBounds(0, Math.max(0, area.height - footerHeight),
          area.width, Math.min(footerHeight, area.height));

      ((CommandTree)composite).updateSize(false, 0);
    }

    public double getRatio() {
      return ratio;
    }

    public void onSashMoved(int x, int width) {
      width -= SASH_WIDTH + sliderWidth;
      if (width < 2 * MIN_CHILD_SIZE) {
        ratio = 0.5;
      } else if (x < MIN_CHILD_SIZE) {
        ratio = (double)MIN_CHILD_SIZE / width;
      } else if (x >= width - MIN_CHILD_SIZE) {
        ratio = (double)(width - MIN_CHILD_SIZE) / width;
      } else {
        ratio = (double)x / width;
      }
    }

    public void onResize(int width) {
      width -= SASH_WIDTH + sliderWidth;
      if (width <= 2 * MIN_CHILD_SIZE) {
        ratio = 0.5;
      } else if (ratio * width < MIN_CHILD_SIZE) {
        ratio = (double)MIN_CHILD_SIZE / width;
      } else if (width - ratio * width < MIN_CHILD_SIZE) {
        ratio = (double)(width - MIN_CHILD_SIZE) / width;
      }
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
