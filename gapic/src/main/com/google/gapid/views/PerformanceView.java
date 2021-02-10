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

import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.models.Profile.PerfNode;
import com.google.gapid.models.Settings;
import com.google.gapid.perfetto.Unit;
import com.google.gapid.perfetto.models.CounterInfo;
import com.google.gapid.perfetto.views.TraceConfigDialog.GpuCountersDialog;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.SettingsProto.UI.PerformancePreset;
import com.google.gapid.proto.device.GpuProfiling;
import com.google.gapid.proto.device.GpuProfiling.GpuCounterDescriptor.GpuCounterSpec;
import com.google.gapid.proto.service.Service;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;
import java.util.Comparator;
import java.util.List;
import java.util.Set;
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
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
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
  private boolean showEstimate = true;
  private Set<Integer> visibleMetrics = Sets.newHashSet();  // identified by metric.id

  public PerformanceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.buttonColor = getDisplay().getSystemColor(SWT.COLOR_LIST_BACKGROUND);

    setLayout(new GridLayout(1, false));
    loading = LoadablePanel.create(this, widgets, p -> new Composite(p, SWT.NONE));
    Composite composite = loading.getContents();
    composite.setLayout(new GridLayout(1, false));
    composite.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    Composite buttonsComposite = createComposite(composite, new GridLayout(3, false));
    buttonsComposite.setLayoutData(new GridData(SWT.LEFT, SWT.TOP, true, false));

    Button toggleButton = createButton(buttonsComposite, SWT.FLAT, "Estimate / Confidence Range",
        buttonColor, e -> toggleEstimateOrRange());
    toggleButton.setImage(widgets.theme.swap());
    toggleButton.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));

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
    presetsBar.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, true, false));

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
    presetsBar.refresh();
    updateTree(false);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    presetsBar.refresh();
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
    presetsBar.refresh();
    visibleMetrics = getCounterSpecs().stream()
        .mapToInt(GpuCounterSpec::getCounterId).boxed().collect(Collectors.toSet());
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
      if (visibleMetrics.contains(metric.getId())){
        addColumnForMetric(metric);
      }
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
    private List<Button> buttons = Lists.newArrayList();

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
      for (Button button : buttons) {
        button.dispose();
      }
      buttons.clear();
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
      buttons.add(addButton);

      for (PerformancePreset preset : settings.ui().getPerformancePresetsList()) {
        if (!preset.getDeviceName().equals(models.devices.getSelectedReplayDevice().getName())) {
          continue;
        }
        buttons.add(createButton(this, SWT.FLAT, preset.getPresetName(), buttonColor, e -> {
          visibleMetrics = Sets.newHashSet(preset.getCounterIdsList());
          updateTree(false);
        }));
      }
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
}
