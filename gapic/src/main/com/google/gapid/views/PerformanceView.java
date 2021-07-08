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

import static com.google.gapid.perfetto.views.StyleConstants.threadStateSleeping;
import static com.google.gapid.perfetto.views.StyleConstants.mainGradient;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.widgets.Widgets.createButton;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createTableColumn;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withLayoutData;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.models.Profile.PerfNode;
import com.google.gapid.models.Settings;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.Unit;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.PanelCanvas;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.CounterInfo;
import com.google.gapid.perfetto.views.TraceConfigDialog.GpuCountersDialog;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.SettingsProto.UI.PerformancePreset;
import com.google.gapid.proto.device.GpuProfiling;
import com.google.gapid.proto.device.GpuProfiling.GpuCounterDescriptor.GpuCounterGroup;
import com.google.gapid.proto.device.GpuProfiling.GpuCounterDescriptor.GpuCounterSpec;

import com.google.gapid.proto.service.Service;
import com.google.gapid.util.Experimental;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.TreeSet;
import java.util.function.Predicate;
import java.util.logging.Logger;
import java.util.stream.Collectors;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.IStructuredSelection;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.jface.viewers.ViewerCell;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowData;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.TreeColumn;
import org.eclipse.swt.widgets.TreeItem;

public class PerformanceView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Profile.Listener {
  protected static final Logger LOG = Logger.getLogger(PerformanceView.class.getName());

  private final Models models;
  private final Color buttonColor;
  private final LoadablePanel<Composite> loading;
  private final PresetsBar presetsBar;
  private final TreeViewer tree;
  private final Map<Integer, Service.ProfilingData.GpuCounters.Metric> columnToMetric = Maps.newHashMap();;
  private final CounterDetailUi counterDetailUi;
  private boolean showEstimate = true;
  private Set<Integer> visibleMetrics = Sets.newHashSet();  // identified by metric.id
  private ViewerCell selectedCell;

  public PerformanceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.buttonColor = getDisplay().getSystemColor(SWT.COLOR_LIST_BACKGROUND);

    setLayout(new FillLayout(SWT.VERTICAL));
    loading = LoadablePanel.create(this, widgets, p -> createComposite(p, new FillLayout(SWT.VERTICAL)));
    Composite composite = loading.getContents();
    SashForm splitter = new SashForm(composite, SWT.VERTICAL);
    Composite top = withLayoutData(createComposite(splitter, new GridLayout(1, false)),
        new GridData(SWT.FILL, SWT.FILL, true, true));
    Composite bottom = withLayoutData(createComposite(splitter, new FillLayout()),
        new GridData(SWT.FILL, SWT.BOTTOM, true, true));
    splitter.setWeights(new int[] { 70, 30 });

    int numberOfButtonsPerRow = 2;
    if (Experimental.enableProfileExperiments(models.settings)) {
      numberOfButtonsPerRow = 3;
    }

    Composite buttonsComposite = createComposite(top, new GridLayout(numberOfButtonsPerRow, false));
    buttonsComposite.setLayoutData(new GridData(SWT.LEFT, SWT.TOP, true, false));

    Button toggleButton = createButton(buttonsComposite, SWT.FLAT, "Estimate / Confidence Range",
        buttonColor, e -> toggleEstimateOrRange());
    toggleButton.setImage(widgets.theme.swap());
    toggleButton.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));

    if (Experimental.enableProfileExperiments(models.settings)) {
      Button experimentsButton =createButton(buttonsComposite, SWT.FLAT, "Experiments",buttonColor,
          e -> widgets.experiments.showExperimentsPopup(getShell()));
      experimentsButton.setImage(widgets.theme.science());
      experimentsButton.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));
    }

    Button filterButton = createButton(buttonsComposite, SWT.FLAT, "Filter Counters", buttonColor, e -> {
      GpuCountersDialog dialog = new GpuCountersDialog(
          getShell(), widgets.theme, getCounterSpecs(), Lists.newArrayList(visibleMetrics));
      if (dialog.open() == Window.OK) {
        visibleMetrics = Sets.newHashSet(dialog.getSelectedIds());
        updateTree(false);
      }
    });
    filterButton.setImage(widgets.theme.more());
    filterButton.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));

    presetsBar = new PresetsBar(buttonsComposite, models.settings, widgets.theme);
    presetsBar.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, true, false, numberOfButtonsPerRow, 1));

    tree = createTreeViewer(top, SWT.NONE);

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

    tree.getTree().addMouseListener(new MouseAdapter() {
      @Override
      public void mouseUp(MouseEvent e) {
        if (selectedCell != null && !selectedCell.getItem().isDisposed()) {
          selectedCell.setFont(widgets.theme.defaultFont());
        }
        ViewerCell newSelectedCell = tree.getCell(new Point(e.x, e.y));
        if (newSelectedCell != null) {
          newSelectedCell.setFont(widgets.theme.selectedTabTitleFont());
        }
        selectedCell = newSelectedCell;

        TreeItem[] items = tree.getTree().getSelection();
        if (items.length > 0) {
          PerfNode selection = (PerfNode)items[0].getData();
          models.profile.linkGpuGroupToCommand(selection.getGroup());
          models.profile.selectGroup(selection.getGroup());
          if (selectedCell != null) {
            counterDetailUi.updateSelectedCell(selection.getEntry(),
                columnToMetric.get(selectedCell.getColumnIndex()));
          }
        }
      }
    });

    tree.getTree().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    counterDetailUi = new CounterDetailUi(bottom, widgets.theme);

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
    presetsBar.refresh();
    updateTree(false);
    counterDetailUi.updateCounterData(models.profile);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    presetsBar.refresh();
    updateTree(true);
    counterDetailUi.updateCounterData(models.profile);
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
    presetsBar.refresh();
    visibleMetrics = getCounterSpecs().stream()
        .mapToInt(GpuCounterSpec::getCounterId).boxed().collect(Collectors.toSet());
    updateTree(false);
    counterDetailUi.updateCounterData(models.profile);
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
    columnToMetric.clear();

    createTreeColumn(tree, "Name", e -> ((Profile.PerfNode)e).getGroup().getName());
    int columnIndex = 1;
    for (Service.ProfilingData.GpuCounters.Metric metric :
        models.profile.getData().getGpuPerformance().getMetricsList()) {
      if (visibleMetrics.contains(metric.getId())){
        addColumnForMetric(metric);
        columnToMetric.put(columnIndex++, metric);
      }
    }
    tree.setInput(perf);
    tree.expandAll(true);
    packColumns(tree.getTree());
    tree.refresh();
  }

  private void addColumnForMetric(Service.ProfilingData.GpuCounters.Metric metric) {
    Unit unit = CounterInfo.unitFromString(metric.getUnit()).withFixedScale(metric.getAverage());
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

  private List<GpuProfiling.GpuCounterDescriptor.GpuCounterSpec> getCounterSpecs() {
    List<GpuProfiling.GpuCounterDescriptor.GpuCounterSpec> specs = Lists.newArrayList();
    if (!models.profile.isLoaded()) {
      return specs;
    }
    // To reuse the existing GpuCountersDialog class for displaying, forge GpuCounterSpec instances
    // with the minimum data requirement that is needed and referenced in GpuCountersDialog.
    for (Service.ProfilingData.GpuCounters.Metric metric : models.profile.getData()
        .getGpuPerformance().getMetricsList()) {
      specs.add(GpuProfiling.GpuCounterDescriptor.GpuCounterSpec.newBuilder()
          .setName(metric.getName())
          .setDescription(metric.getDescription())
          .setCounterId(metric.getId())
          .setSelectByDefault(metric.getSelectByDefault())
          .build());
    }
    return specs;
  }

  private class PresetsBar extends Composite {
    private final Settings settings;
    private final Theme theme;

    public PresetsBar(Composite parent, Settings settings, Theme theme) {
      super(parent, SWT.NONE);
      this.settings = settings;
      this.theme = theme;

      RowLayout stripLayout = withMargin(new RowLayout(SWT.HORIZONTAL), 0, 0);
      stripLayout.fill = true;
      stripLayout.wrap = true;
      stripLayout.spacing = 5;
      setLayout(stripLayout);
    }

    public void refresh() {
      for (Control children : this.getChildren()) {
        children.dispose();
      }
      createPresetButtons();
      redraw();
      requestLayout();
    }

    private void createPresetButtons() {
      if (models.devices.getSelectedReplayDevice() == null) {
        return;
      }

      Button addButton = createButton(this, SWT.FLAT, "Add New Preset", buttonColor, e -> {
        AddPresetDialog dialog = new AddPresetDialog(
            getShell(), theme, getCounterSpecs(), Lists.newArrayList());
        if (dialog.open() == Window.OK) {
          Set<Integer> selectedIds = Sets.newHashSet(dialog.getSelectedIds());
          visibleMetrics = selectedIds;
          models.settings.writeUi().addPerformancePresets(SettingsProto.UI.PerformancePreset.newBuilder()
              .setPresetName(dialog.getFinalPresetName())
              .setDeviceName(models.devices.getSelectedReplayDevice().getName())
              .addAllCounterIds(selectedIds)
              .build());
          refresh();
          updateTree(false);
        }
      });
      addButton.setImage(theme.add());
      withLayoutData(new Label(this, SWT.VERTICAL | SWT.SEPARATOR), new RowData(SWT.DEFAULT, 1));

      boolean customPresetButtonCreated = false;
      for (PerformancePreset preset : settings.ui().getPerformancePresetsList()) {
        if (!preset.getDeviceName().equals(models.devices.getSelectedReplayDevice().getName())) {
          continue;
        }
        createButton(this, SWT.FLAT, preset.getPresetName(), buttonColor, e -> {
          visibleMetrics = Sets.newHashSet(preset.getCounterIdsList());
          updateTree(false);
        });
        customPresetButtonCreated = true;
      }
      if (customPresetButtonCreated) {
        withLayoutData(new Label(this, SWT.VERTICAL | SWT.SEPARATOR), new RowData(SWT.DEFAULT, 1));
      }


      for (PerformancePreset preset : getRecommendedPresets()) {
        createButton(this, SWT.FLAT, preset.getPresetName(), buttonColor, e -> {
          visibleMetrics = Sets.newHashSet(preset.getCounterIdsList());
          updateTree(false);
        });
      }
    }

    // Create and return a list of presets based on vendor provided GPU counter grouping metadata.
    private List<SettingsProto.UI.PerformancePreset> getRecommendedPresets() {
      List<SettingsProto.UI.PerformancePreset> presets = Lists.newArrayList();
      if (!models.profile.isLoaded()) {
        return presets;
      }
      Map<GpuCounterGroup, List<Integer>> groupToMetrics = Maps.newHashMap();
      // Pre-create the map entries so they go with the default order in enum definition.
      for (GpuCounterGroup group : GpuCounterGroup.values()) {
        groupToMetrics.put(group, Lists.newArrayList());
      }
      for (Service.ProfilingData.GpuCounters.Metric metric: models.profile.getData().
          getGpuPerformance().getMetricsList()) {
        for (GpuCounterGroup group : metric.getCounterGroupsList()) {
          groupToMetrics.get(group).add(metric.getId());
        }
      }
      for (GpuCounterGroup group : groupToMetrics.keySet()) {
        if (group != GpuCounterGroup.UNCLASSIFIED && groupToMetrics.get(group).size() > 0) {
          presets.add(SettingsProto.UI.PerformancePreset.newBuilder()
              .setPresetName(group.name())
              .setDeviceName(models.devices.getSelectedReplayDevice().getName())
              .addAllCounterIds(groupToMetrics.get(group))
              .build());
        }
      }
      return presets;
    }

    private class AddPresetDialog extends GpuCountersDialog {
      private Text presetNameInput;
      private String finalPresetName;
      private Label warningLabel;

      public AddPresetDialog(Shell shell, Theme theme, List<GpuCounterSpec> specs,
          List<Integer> currentIds) {
        super(shell, theme, specs, currentIds);
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        Button button = createButton(parent, IDialogConstants.OPEN_ID, "Manage Presets", false);
        button.addListener(SWT.Selection,
            e-> new ManagePresetsDialog(getShell(), theme, getCounterSpecs(), Lists.newArrayList()).open());
        super.createButtonsForButtonBar(parent);
      }

      @Override
      protected Control createContents(Composite parent) {
        Control control = super.createContents(parent);
        Button okButton = getButton(IDialogConstants.OK_ID);
        okButton.setText("Add");
        okButton.setEnabled(false);
        return control;
      }

      @Override
      public String getTitle() {
        return "Create New Preset";
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = createComposite(parent, withMargin(new GridLayout(2, false),
            IDialogConstants.HORIZONTAL_MARGIN, IDialogConstants.VERTICAL_MARGIN));
        area.setLayoutData(new GridData(GridData.FILL_BOTH));

        String currentDevice = models.devices.getSelectedReplayDevice().getName();
        createLabel(area, "Current Device: ").setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));
        createLabel(area, currentDevice).setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, true, false));

        createLabel(area, "Preset Name: ").setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));
        presetNameInput = createTextbox(area, "");
        presetNameInput.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, true, false));
        Set<String> usedNames = Sets.newHashSet();
        settings.ui().getPerformancePresetsList().stream()
            .filter(p -> p.getDeviceName().equals(currentDevice))
            .forEach(p -> usedNames.add(p.getPresetName()));
        presetNameInput.addModifyListener(e -> {
          String input = presetNameInput.getText();
          Button okButton = getButton(IDialogConstants.OK_ID);
          if (input.isEmpty() || usedNames.contains(input)) {
            okButton.setEnabled(false);
            warningLabel.setVisible(true);
          } else {
            okButton.setEnabled(true);
            warningLabel.setVisible(false);
          }
        });

        Composite tableArea = createComposite(area, new GridLayout());
        tableArea.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true, 2, 1));
        createGpuCounterTable(tableArea);

        warningLabel = createLabel(area, "Preset name empty or already exist.");
        warningLabel.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_RED));

        return area;
      }

      @Override
      protected void okPressed() {
        finalPresetName = presetNameInput.getText();
        super.okPressed();
      }

      public String getFinalPresetName() {
        return finalPresetName;
      }
    }

    private class ManagePresetsDialog extends GpuCountersDialog {
      private final List<PerformancePreset> removalWaitlist = Lists.newArrayList();

      public ManagePresetsDialog(Shell shell, Theme theme, List<GpuCounterSpec> specs,
          List<Integer> currentIds) {
        super(shell, theme, specs, currentIds);
      }

      @Override
      public String getTitle() {
        return "Manage Presets List";
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        super.createButtonsForButtonBar(parent);
        getButton(IDialogConstants.OK_ID).setText("Save");
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = createComposite(parent, withMargin(new GridLayout(2, false),
            IDialogConstants.HORIZONTAL_MARGIN, IDialogConstants.VERTICAL_MARGIN));
        area.setLayoutData(new GridData(GridData.FILL_BOTH));

        // Create the presets listing table.
        TableViewer viewer = createTableViewer(area, SWT.NONE);
        viewer.getTable().setLayoutData(new GridData(SWT.LEFT, SWT.FILL, false, true));
        viewer.setContentProvider(new ArrayContentProvider());
        viewer.setLabelProvider(new LabelProvider());
        createTableColumn(viewer, "Device", p -> ((PerformancePreset)p).getDeviceName());
        createTableColumn(viewer, "Preset Name", p -> ((PerformancePreset)p).getPresetName());
        viewer.addSelectionChangedListener(e -> {
          IStructuredSelection selection = e.getStructuredSelection();
          // Handle an edge case after deletion.
          if (selection == null || selection.getFirstElement() == null) {
            table.setAllChecked(false);
            return;
          }
          PerformancePreset selectedPreset = (PerformancePreset)selection.getFirstElement();
          Set<Integer> counterIds = Sets.newHashSet(selectedPreset.getCounterIdsList());
          table.setCheckedElements(getSpecs().stream()
              .filter(s -> counterIds.contains(s.getCounterId()))
              .toArray(GpuCounterSpec[]::new));
        });
        List<PerformancePreset> presets = Lists.newArrayList(settings.ui().getPerformancePresetsList());
        viewer.setInput(presets);
        packColumns(viewer.getTable());

        // Create the GPU counter table, which will reflect the selected preset's containing counters.
        Composite tableArea = createComposite(area, new GridLayout());
        tableArea.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
        createGpuCounterTable(tableArea);

        // Create the delete button.
        Widgets.createButton(area, SWT.FLAT, "Delete", buttonColor, e -> {
          IStructuredSelection selection = viewer.getStructuredSelection();
          if (selection == null || selection.getFirstElement() == null) {
            return;
          }
          PerformancePreset selectedPreset = (PerformancePreset)selection.getFirstElement();
          removalWaitlist.add(selectedPreset);
          presets.remove(selectedPreset);
          viewer.refresh();
        });

        return area;
      }

      @Override
      protected void okPressed() {
        SettingsProto.UI.Builder uiBuilder = models.settings.writeUi();
        // Reverse iteration, so as to avoid getting affected by index change at removal.
        for (int i = uiBuilder.getPerformancePresetsCount() - 1; i >= 0 ; i--) {
          if (removalWaitlist.contains(uiBuilder.getPerformancePresets(i))) {
            uiBuilder.removePerformancePresets(i);
          }
        }
        presetsBar.refresh();
        super.okPressed();
      }
    }
  }

  private static class CounterDetailUi extends Composite {
    private static final String ENTRY_LABEL_PREFIX = "Entry: ";
    private static final String METRIC_LABEL_PREFIX = "Metric: ";

    private final Label coordinateLabel;
    private final CounterDetailPanel counterDetailPanel;
    private final PanelCanvas canvas;
    private final Map<Integer, Service.ProfilingData.Counter> counterDataLookup = Maps.newHashMap(); // metric id -> counter data.
    private Service.ProfilingData.GpuCounters.Entry selectedEntry;      // Row.
    private Service.ProfilingData.GpuCounters.Metric selectedMetric;    // Column.

    public CounterDetailUi(Composite parent, Theme theme) {
      super(parent, SWT.NONE);

      setLayout(withMargin(new GridLayout(2, false), 0, 5));
      Label titleLabel = withLayoutData(createLabel(this, "GPU Counter Detail Graph: "),
          new GridData(SWT.LEFT, SWT.FILL, true, false, 2, 1));
      titleLabel.setFont(theme.selectedTabTitleFont());
      coordinateLabel = withLayoutData(createLabel(this, ENTRY_LABEL_PREFIX + "; " + METRIC_LABEL_PREFIX),
          new GridData(SWT.LEFT, SWT.FILL, true, false));
      withLayoutData(createLabel(this, "* (counter sample weight). ** Only for GPU counters."),
          new GridData(SWT.RIGHT, SWT.FILL, false, false));
      counterDetailPanel = new CounterDetailPanel();
      this.canvas = withLayoutData(new PanelCanvas(this, SWT.NONE, theme, counterDetailPanel),
          new GridData(SWT.FILL, SWT.FILL, true, true, 2, 1));
    }

    public void updateCounterData(Profile profile) {
      counterDataLookup.clear();
      if (profile == null || !profile.isLoaded()) {
        updateSelectedCell(null, null);
        return;
      }
      List<Service.ProfilingData.Counter> counters = profile.getData().getCounters();
      Service.ProfilingData.GpuCounters perfs = profile.getData().getGpuPerformance();
      List<Service.ProfilingData.GpuCounters.Metric> metrics = perfs.getMetricsList();
      for (Service.ProfilingData.Counter counter: counters) {
        for (Service.ProfilingData.GpuCounters.Metric metric : metrics) {
          if (counter.getId() == metric.getCounterId()) {
            counterDataLookup.put(metric.getId(), counter);
          }
        }
      }
      counterDetailPanel.updateTimeRange(profile.getData().getSlicesTimeSpan());
    }

    public void updateSelectedCell(Service.ProfilingData.GpuCounters.Entry entry, Service.ProfilingData.GpuCounters.Metric metric) {
      this.selectedEntry = entry;
      this.selectedMetric = metric;
      String coordinateStr = ENTRY_LABEL_PREFIX
          + (entry != null ? entry.getGroup().getName() : "")
          + "; " + METRIC_LABEL_PREFIX
          + (metric != null ? metric.getName() : "");
      coordinateLabel.setText(coordinateStr);
      coordinateLabel.requestLayout();
      canvas.redraw();
    }

    private class CounterDetailPanel extends Panel.Base {
      public static final double TRACK_HEIGHT = 200;
      public static final double TRACK_HEIGHT_MARGIN = 30;
      public static final double TRACK_WIDTH_MARGIN = 20;
      public static final double TRACK_X_AXIS_HEIGHT = 10;
      public static final double TRACK_Y_AXIS_WIDTH = 100;
      public static final double AXIS_ARROW_LENGTH = 10;
      public static final double AXIS_ARROW_SIZE = 5;
      public static final double AXIS_Y_CHUNK_NUM = 4;

      private long traceStart;

      public CounterDetailPanel() {
        super();
      }

      // Update the time range of the trace.
      public void updateTimeRange(TimeSpan ts) {
        traceStart = ts.start;
      }

      @Override
      public double getPreferredHeight() {
        return TRACK_HEIGHT + 2 * TRACK_HEIGHT_MARGIN + TRACK_X_AXIS_HEIGHT + AXIS_ARROW_SIZE;
      }

      @Override
      public void render(RenderContext ctx, Repainter repainter) {
        ctx.trace("Counter", () -> {
          int selectedMetricId = selectedMetric == null ? -1 : selectedMetric.getId();
          if (selectedEntry == null
              || !selectedEntry.getMetricToValueMap().containsKey(selectedMetricId)
              || !counterDataLookup.containsKey(selectedMetricId)) {
            return;
          }
          Service.ProfilingData.Counter counter = counterDataLookup.get(selectedMetricId);
          Service.ProfilingData.GpuCounters.Perf perf = selectedEntry.getMetricToValueMap().get(selectedMetricId);
          // Use TreeSet so that the indexes are sorted during iteration.
          TreeSet<Integer> indexes = Sets.newTreeSet(perf.getEstimateSamplesMap().keySet());
          if (indexes.size() == 0) {
            return;
          }
          double max = indexes.stream().mapToDouble(counter::getValues).max().getAsDouble();

          double h = height - 2 * TRACK_HEIGHT_MARGIN - TRACK_X_AXIS_HEIGHT;
          double w = width - 2 * TRACK_WIDTH_MARGIN - TRACK_Y_AXIS_WIDTH;
          double top = TRACK_HEIGHT_MARGIN, bottom = height - TRACK_X_AXIS_HEIGHT - TRACK_HEIGHT_MARGIN;
          double left = TRACK_Y_AXIS_WIDTH + TRACK_WIDTH_MARGIN, right = width - TRACK_WIDTH_MARGIN;
          double stepX = w / (indexes.size());
          double stepY = h / AXIS_Y_CHUNK_NUM;

          // Draw graph.
          mainGradient().applyBaseAndBorder(ctx);
          ctx.path(path -> {
            double lastX = left;
            path.moveTo(left, bottom);
            for (int i : indexes) {
              double nextX = lastX + stepX;
              double nextY = top + h * (1 - counter.getValues(i) / max);
              path.lineTo(lastX, nextY);  // Go up.
              path.lineTo(nextX, nextY);  // Go right.
              ctx.drawTextCenteredRightTruncate(Fonts.Style.Normal,
                  "(" + perf.getEstimateSamplesMap().get(i) + ")",
                  lastX, nextY - 20, stepX, 20);
              lastX = nextX;
            }
            path.lineTo(lastX, bottom);
            ctx.fillPath(path);
            ctx.drawPath(path);
          });

          // Draw x-axis.
          threadStateSleeping().applyBaseAndBorder(ctx);
          ctx.drawLine(left, bottom, right + AXIS_ARROW_LENGTH, bottom);
          ctx.drawLine(right + AXIS_ARROW_LENGTH, bottom, right + AXIS_ARROW_LENGTH - AXIS_ARROW_SIZE, bottom + AXIS_ARROW_SIZE);
          ctx.drawLine(right + AXIS_ARROW_LENGTH, bottom, right + AXIS_ARROW_LENGTH - AXIS_ARROW_SIZE, bottom - AXIS_ARROW_SIZE);
          double x = left;
          for (int i : indexes) {
            // A counter sample at time {tn} is seen as holding the value {vn} during the implicit
            // time range between {tn-1} ~ {tn}.
            if (i > 0) {
              ctx.drawText(Fonts.Style.Normal, Unit.NANO_SECOND.format(
                  counter.getTimestamps(i - 1) - traceStart), x, bottom + AXIS_ARROW_SIZE);
            }
            ctx.drawLine(x, bottom, x, bottom - AXIS_ARROW_SIZE);
            x += stepX;
          }
          ctx.drawTextRightJustified(Fonts.Style.Normal, Unit.NANO_SECOND.format(
              counter.getTimestamps(indexes.last()) - traceStart), x, bottom + AXIS_ARROW_SIZE);

          // Draw y-axis.
          ctx.drawLine(left, bottom, left, top - AXIS_ARROW_LENGTH);
          ctx.drawLine(left, top - AXIS_ARROW_LENGTH, left - AXIS_ARROW_SIZE, top - AXIS_ARROW_LENGTH + AXIS_ARROW_SIZE);
          ctx.drawLine(left, top - AXIS_ARROW_LENGTH, left + AXIS_ARROW_SIZE, top - AXIS_ARROW_LENGTH + AXIS_ARROW_SIZE);
          for (int i = 1; i <= AXIS_Y_CHUNK_NUM; i++) {
            ctx.drawText(Fonts.Style.Normal, CounterInfo.unitFromString(
                selectedMetric.getUnit()).format(max / AXIS_Y_CHUNK_NUM * i),
                TRACK_WIDTH_MARGIN, bottom - stepY * i);
            ctx.drawLine(left, bottom - stepY * i, left + AXIS_ARROW_SIZE, bottom - stepY * i);
          }
        });
      }
    }
  }
}
