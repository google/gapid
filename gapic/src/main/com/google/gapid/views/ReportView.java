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

import static com.google.gapid.util.Loadable.Message.info;
import static com.google.gapid.util.Loadable.Message.smile;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createTreeViewer;

import com.google.common.base.Throwables;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import com.google.common.collect.Maps;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.models.Reports;
import com.google.gapid.models.Settings;
import com.google.gapid.models.Strings;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.stringtable.Stringtable;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.MeasuringViewLabelProvider;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ILazyTreeContentProvider;
import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.viewers.TreeSelection;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.TreeItem;
import org.eclipse.swt.widgets.Widget;

import java.util.Map;
import java.util.concurrent.ExecutionException;

/**
 * View that shows the capture report items in a tree.
 */
public class ReportView extends Composite implements Tab, Capture.Listener, Reports.Listener {
  private final Models models;
  private final MessageProvider messages = new MessageProvider();
  private final LoadablePanel<SashForm> loading;
  private final TreeViewer viewer;
  private final Composite detailsGroup;
  private Text reportDetails;

  // Need this flag to prevent a weird quirk where when opening a second
  // trace, the report of the previous trace will show up once everything
  // is fully loaded.
  private boolean ranReport = false;

  public ReportView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new GridLayout(1, false));

    Composite buttons = Widgets.withLayoutData(Widgets.createComposite(this, new FillLayout(SWT.VERTICAL)),
      new GridData(SWT.LEFT, SWT.TOP, false, false));
    Widgets.createButton(buttons, "Generate Report", e-> {
      models.reports.reload();
      ranReport = true;
    });


    Composite top = Widgets.withLayoutData(Widgets.createComposite(this, new FillLayout(SWT.VERTICAL)),
      new GridData(SWT.FILL, SWT.FILL, true, true));
    loading = LoadablePanel.create(top, widgets, panel -> new SashForm(panel, SWT.VERTICAL));
    SashForm splitter = loading.getContents();

    viewer = createTreeViewer(splitter, SWT.H_SCROLL | SWT.V_SCROLL | SWT.VIRTUAL);
    viewer.setContentProvider(new ReportContentProvider(viewer, messages));
    ViewLabelProvider labelProvider = new ViewLabelProvider(viewer, widgets.theme, messages);
    viewer.setLabelProvider(labelProvider);

    detailsGroup = Widgets.createGroup(splitter, "Details");

    splitter.setWeights(models.settings.getSplitterWeights(Settings.SplitterWeights.Report));

    models.capture.addListener(this);
    models.reports.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.reports.removeListener(this);
    });

    viewer.getTree().addListener(SWT.MouseMove, e -> {
      Object follow = labelProvider.getFollow(new Point(e.x, e.y));
      setCursor((follow == null) ? null : e.display.getSystemCursor(SWT.CURSOR_HAND));
    });
    viewer.getTree().addListener(SWT.MouseDown, e -> {
      Path.Command command = (Path.Command)labelProvider.getFollow(new Point(e.x, e.y));
      if (command != null) {
        models.commands.selectCommands(CommandIndex.forCommand(command), true);
      }
    });
    viewer.getTree().addListener(SWT.Selection, e -> {
      if (viewer.getTree().getSelectionCount() > 0) {
        TreeItem item = viewer.getTree().getSelection()[0];
        while (item != null && !(item.getData() instanceof Group)) {
          item = item.getParentItem();
        }
        if (item != null) {
          getDetails().setText(((Group)item.getData()).name);
          getDetails().requestLayout();
        }
      }
    });
    addListener(SWT.Dispose, e ->
      models.settings.setSplitterWeights(Settings.SplitterWeights.Report, splitter.getWeights()));
  }

  private Text getDetails() {
    // Lazy init'ed due to https://github.com/google/gapid/issues/2624
    if (reportDetails == null) {
      reportDetails = new Text(detailsGroup, SWT.MULTI | SWT.READ_ONLY | SWT.WRAP | SWT.V_SCROLL);
    }
    return reportDetails;
  }

  @Override
  public Control getControl() {
    return this;
  }

  private void clear() {
    loading.showMessage(info("Report not generated. Press \"Generate Report\" button."));
    ranReport = false;
  }

  @Override
  public void reinitialize() {
    onCaptureLoadingStart(false);
    onReportLoaded();
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      clear();
    }
  }

  @Override
  public void onReportLoadingStart() {
    loading.startLoading();
  }

  @Override
  public void onReportLoaded() {
    if (ranReport) {
      messages.clear();
      getDetails().setText("");
      if (models.reports.isLoaded()) {
        updateReport();
      } else {
        loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
      }
    } else {
      clear();
    }
  }

  private void updateReport() {
    loading.stopLoading();
    Service.Report report = models.reports.getData().report;
    if (report.getGroupsCount() == 0) {
      loading.showMessage(smile("Rock on! No issues found in this trace."));
    } else {
      viewer.setInput(report);
      viewer.setSelection(
          new TreeSelection(new TreePath(new Object[] { viewer.getInput() })), true);
    }
  }

  /**
   * A node in the tree representing a report item group with children.
   */
  private static class Group {
    public final Service.ReportGroup group;
    public final String name;

    public Group(MessageProvider messages, Service.Report report, Service.ReportGroup group) {
      this.group = group;
      this.name = messages.get(report, group.getName());
    }
  }

  /**
   * A report item leaf in the tree.
   */
  private static class Item {
    public final Service.Report report;
    public final Service.ReportItem item;

    public Item(Service.Report report, int index) {
      this.report = report;
      this.item = report.getItems(index);
    }
  }

  /**
   * Content provider for the report tree.
   */
  private static class ReportContentProvider implements ILazyTreeContentProvider {
    private final TreeViewer viewer;
    private final MessageProvider messages;
    private Service.Report report;

    public ReportContentProvider(TreeViewer viewer, MessageProvider messages) {
      this.viewer = viewer;
      this.messages = messages;
    }

    @Override
    public void inputChanged(Viewer v, Object oldInput, Object newInput) {
      report = (Service.Report)newInput;
    }

    @Override
    public void updateChildCount(Object element, int currentChildCount) {
      if (element instanceof Service.Report) {
        viewer.setChildCount(element, ((Service.Report)element).getGroupsCount());
      } else if (element instanceof Group) {
        viewer.setChildCount(element, ((Group)element).group.getItemsCount());
      } else {
        viewer.setChildCount(element, 0);
      }
    }

    @Override
    public void updateElement(Object parent, int index) {
      if (parent instanceof Service.Report) {
        Group group = new Group(messages, report, report.getGroups(index));
        viewer.replace(parent, index, group);
        viewer.setChildCount(group, group.group.getItemsCount());
      } else if (parent instanceof Group) {
        Item item = new Item(report, ((Group)parent).group.getItems(index));
        viewer.replace(parent, index, item);
        viewer.setChildCount(item, 0);
      }
    }

    @Override
    public Object getParent(Object element) {
      return null;
    }
  }

  /**
   * Label provider for the report tree.
   */
  private static class ViewLabelProvider extends MeasuringViewLabelProvider {
    private static final int TAG_STR_LENGTH = 40;

    private final MessageProvider messages;

    public ViewLabelProvider(TreeViewer viewer, Theme theme, MessageProvider messages) {
      super(viewer, theme);
      this.messages = messages;
    }

    @Override
    protected <S extends StylingString> S format(Widget widget, Object element, S string) {
      if (element instanceof Group) {
        Group group = (Group)element;
        string.append(trimGroupString(group.name), string.defaultStyle());
        string.append(" " + group.group.getItemsCount(), string.structureStyle());
      } else if (element instanceof Item) {
        Item item = (Item)element;
        string.startLink(item.item.getCommand());
        string.append(Formatter.commandIndex(item.item.getCommand()), string.linkStyle());
        string.endLink();
        string.append(": ", string.structureStyle());
        switch (item.item.getSeverity()) {
          case FatalLevel:
          case ErrorLevel:
            string.append(trimSeverity(item.item), string.errorStyle());
            break;
          case WarningLevel:
            string.append(trimSeverity(item.item), string.errorStyle());
            break;
          default:
            string.append(trimSeverity(item.item), string.defaultStyle());
        }

        String sep = " ";
        for (Service.MsgRef tag : item.item.getTagsList()) {
          string.append(sep, string.structureStyle());
          string.append(trimTagString(messages.get(item.report, tag)), string.defaultStyle());
          sep = ", ";
        }
      }
      return string;
    }

    @Override
    protected boolean isFollowable(Object element) {
      return element instanceof Item;
    }

    private static String trimGroupString(String str) {
      str = str.trim();
      int p = str.indexOf('\n');
      if (p > 0) {
        str = str.substring(0, p) + '…';
      }
      return str;
    }

    private static String trimTagString(String str) {
      String result = str;
      if (result.charAt(result.length() - 1) == '.') {
        result = result.substring(0, result.length() - 1);
      }
      if (result.length() > TAG_STR_LENGTH) {
        result = result.substring(0, TAG_STR_LENGTH - 1) + '…';
      }
      return result;
    }

    private static String trimSeverity(Service.ReportItem item) {
      String result = item.getSeverity().name();
      if (result.endsWith("Level")) {
        result = result.substring(0, result.length() - 5);
      }
      return result;
    }
  }

  /**
   * Formats the {@link com.google.gapid.proto.service.Service.MsgRef messages} in the report tree.
   */
  private static class MessageProvider {
    private final Cache<Service.MsgRef, String> cache =
        CacheBuilder.newBuilder().softValues().build();

    public MessageProvider() {
    }

    public void clear() {
      cache.invalidateAll();
    }

    public String get(Service.Report report, Service.MsgRef ref) {
      try {
        return cache.get(ref, () -> {
          Map<String, Stringtable.Value> arguments = Maps.newHashMap();
          for (Service.MsgRefArgument a : ref.getArgumentsList()) {
            arguments.put(report.getStrings(a.getKey()), report.getValues(a.getValue()));
          }
          return Strings.getMessage(report.getStrings(ref.getIdentifier()), arguments);
        });
      } catch (ExecutionException e) {
        Throwables.throwIfUnchecked(e.getCause());
        throw new RuntimeException(e);
      }
    }
  }
}
