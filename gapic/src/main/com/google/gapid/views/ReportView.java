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

import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createTreeViewer;

import com.google.common.base.Throwables;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import com.google.common.collect.Maps;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.ApiContext;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Reports;
import com.google.gapid.models.Strings;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.MsgRef;
import com.google.gapid.proto.service.Service.Report;
import com.google.gapid.proto.service.Service.ReportGroup;
import com.google.gapid.proto.service.Service.ReportItem;
import com.google.gapid.proto.stringtable.Stringtable;
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
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Map;
import java.util.concurrent.ExecutionException;

/**
 * View that shows the capture report items in a tree.
 */
public class ReportView extends Composite
    implements Tab, Capture.Listener, ApiContext.Listener, Reports.Listener {
  private final Models models;
  private final MessageProvider messages = new MessageProvider();
  private final LoadablePanel<SashForm> loading;
  private final TreeViewer viewer;
  private final Text reportDetails;

  public ReportView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));
    loading = LoadablePanel.create(this, widgets, panel -> new SashForm(panel, SWT.VERTICAL));
    SashForm splitter = loading.getContents();

    viewer = createTreeViewer(splitter, SWT.H_SCROLL | SWT.V_SCROLL | SWT.VIRTUAL);
    viewer.setContentProvider(new ReportContentProvider(viewer, messages));
    ViewLabelProvider labelProvider = new ViewLabelProvider(viewer, widgets.theme, messages);
    viewer.setLabelProvider(labelProvider);

    Composite detailsGroup = Widgets.createGroup(splitter, "Details");
    reportDetails = new Text(detailsGroup, SWT.MULTI | SWT.READ_ONLY | SWT.H_SCROLL | SWT.V_SCROLL);

    splitter.setWeights(models.settings.reportSplitterWeights);

    models.capture.addListener(this);
    models.contexts.addListener(this);
    models.reports.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.contexts.removeListener(this);
      models.reports.removeListener(this);
    });

    viewer.getTree().addListener(SWT.MouseMove, e -> {
      Object follow = labelProvider.getFollow(new Point(e.x, e.y));
      setCursor((follow == null) ? null : e.display.getSystemCursor(SWT.CURSOR_HAND));
    });
    viewer.getTree().addListener(SWT.MouseDown, e -> {
      Long atomId = (Long)labelProvider.getFollow(new Point(e.x, e.y));
      if (atomId != null) {
        models.atoms.selectAtoms(atomId, 1, true);
      }
    });
    viewer.getTree().addListener(SWT.Selection, e -> {
      if (viewer.getTree().getSelectionCount() > 0) {
        TreeItem item = viewer.getTree().getSelection()[0];
        while (item != null && !(item.getData() instanceof Group)) {
          item = item.getParentItem();
        }
        if (item != null) {
          reportDetails.setText(((Group)item.getData()).name);
          reportDetails.requestLayout();
        }
      }
    });
    addListener(SWT.Dispose, e -> models.settings.reportSplitterWeights = splitter.getWeights());
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    onCaptureLoadingStart(false);
    updateReport();
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(GapisInitException error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onContextsLoaded() {
    updateReport();
  }

  @Override
  public void onContextSelected(FilteringContext context) {
    updateReport();
  }

  @Override
  public void onReportLoaded() {
    messages.clear();
    if (models.reports.isLoaded()) {
      updateReport();
    } else {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  private void updateReport() {
    if (models.reports.isLoaded()) {
      loading.stopLoading();
      viewer.setInput(filter(models.reports.getData()));
      viewer.setSelection(new TreeSelection(new TreePath(new Object[] { viewer.getInput() })), true);
    }
  }

  private Service.Report filter(Service.Report report) {
    FilteringContext context = models.contexts.getSelectedContext();
    if (context == FilteringContext.ALL) {
      return report;
    }

    Service.Report.Builder result = report.toBuilder();
    for (int i = report.getGroupsCount() - 1; i >= 0; i--) {
      for (int item : report.getGroups(i).getItemsList()) {
        if (!context.contains(report.getItems(item).getCommand())) {
          result.removeGroups(i);
          break;
        }
      }
    }
    return result.build();
  }

  /**
   * A node in the tree representing a report item group with children.
   */
  private static class Group {
    public final ReportGroup group;
    public final String name;

    public Group(MessageProvider messages, Report report, ReportGroup group) {
      this.group = group;
      this.name = messages.get(report, group.getName());
    }
  }

  /**
   * A report item leaf in the tree.
   */
  private static class Item {
    public final Report report;
    public final ReportItem item;

    public Item(Report report, int index) {
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
    private Report report;

    public ReportContentProvider(TreeViewer viewer, MessageProvider messages) {
      this.viewer = viewer;
      this.messages = messages;
    }

    @Override
    public void inputChanged(Viewer v, Object oldInput, Object newInput) {
      report = (Report)newInput;
    }

    @Override
    public void updateChildCount(Object element, int currentChildCount) {
      if (element instanceof Report) {
        viewer.setChildCount(element, ((Report)element).getGroupsCount());
      } else if (element instanceof Group) {
        viewer.setChildCount(element, ((Group)element).group.getItemsCount());
      } else {
        viewer.setChildCount(element, 0);
      }
    }

    @Override
    public void updateElement(Object parent, int index) {
      if (parent instanceof Report) {
        Group group = new Group(messages, (Report)parent, ((Report)parent).getGroups(index));
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
    protected <S extends StylingString> S format(Object element, S string) {
      if (element instanceof Group) {
        Group group = (Group)element;
        string.append(trimGroupString(group.name), string.defaultStyle());
        string.append(" " + group.group.getItemsCount(), string.structureStyle());
      } else if (element instanceof Item) {
        Item item = (Item)element;
        string.startLink(item.item.getCommand());
        string.append(String.valueOf(item.item.getCommand()), string.linkStyle());
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
        for (MsgRef tag : item.item.getTagsList()) {
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
        str = str.substring(0, p) + "\u2026";
      }
      return str;
    }

    private static String trimTagString(String str) {
      String result = str;
      if (result.charAt(result.length() - 1) == '.') {
        result = result.substring(0, result.length() - 1);
      }
      if (result.length() > TAG_STR_LENGTH) {
        result = result.substring(0, TAG_STR_LENGTH - 1) + "\u2026";
      }
      return result;
    }

    private static String trimSeverity(ReportItem item) {
      String result = item.getSeverity().name();
      if (result.endsWith("Level")) {
        result = result.substring(0, result.length() - 5);
      }
      return result;
    }
  }

  /**
   * Formats the various {@link MsgRef messages} in the report tree.
   */
  private static class MessageProvider {
    private final Cache<MsgRef, String> cache = CacheBuilder.newBuilder().softValues().build();

    public MessageProvider() {
    }

    public void clear() {
      cache.invalidateAll();
    }

    public String get(Report report, MsgRef ref) {
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
