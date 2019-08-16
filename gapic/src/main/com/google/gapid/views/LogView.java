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

import static com.google.gapid.widgets.Widgets.createTree;

import com.google.gapid.proto.log.Log;
import com.google.gapid.util.Logging;
import com.google.gapid.util.Pods;
import com.google.gapid.widgets.CopySources;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;
import com.google.protobuf.Timestamp;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeColumn;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Calendar;
import java.util.Date;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * A view that shows log messages.
 */
public class LogView extends Composite implements Tab {
  private final Theme theme;
  private final Tree tree;
  private final AtomicBoolean dirty = new AtomicBoolean(false);
  private final int MAX_ITEMS = 10000;
  private final int MAX_NEW_ITEMS_PER_UPDATE = 100;
  private Logging.MessageIterator messageIterator;

  private enum Column {
    SEVERITY(0, 35, "Severity"),
    TIME(1, 100, "Time"),
    PROCESS(2, 60, "Process"),
    TEXT(3, 300, "Text");

    final int index;
    final int width;
    final String name;

    Column(int index, int width, String name) {
      this.index = index;
      this.width = width;
      this.name = name;
    }
  }

  public LogView(Composite parent, Widgets widgets) {
    super(parent, SWT.NONE);
    theme = widgets.theme;

    setLayout(new FillLayout(SWT.VERTICAL));

    messageIterator = Logging.getMessageIterator();

    tree = createTree(this, SWT.MULTI | SWT.H_SCROLL | SWT.V_SCROLL | SWT.FULL_SELECTION);
    tree.setHeaderVisible(true);
    tree.setFont(theme.monoSpaceFont());

    for (Column column : Column.values()) {
      TreeColumn treeColumn = new TreeColumn(tree, SWT.LEFT);
      treeColumn.setText(column.name);
      treeColumn.setWidth(column.width);
    }
    updateTree();
    Logging.setListener((m) -> {
      if (!dirty.getAndSet(true)) {
        Widgets.scheduleIfNotDisposed(this, this::updateTree);
      }
    });
    addListener(SWT.Dispose, e -> Logging.setListener(null));

    CopySources.registerTreeAsCopySource(widgets.copypaste, tree, item -> {
      String[] result = new String[Column.values().length];
      for (int i = 0; i < result.length; i++) {
        result[i] = item.getText(i);
      }
      return result;
    }, false);
  }

  private void updateTree() {
    for (int i = 0; i < MAX_NEW_ITEMS_PER_UPDATE; i++) {
      Log.Message message = messageIterator.next();
      if (message == null) {
        // All messages consumed.
        dirty.set(false);
        return;
      }
      Log.Severity severity = message.getSeverity();
      TreeItem item = newItem(tree, severity);
      String[] lines = message.getText().split("(\r?\n)");
      item.setText(Column.SEVERITY.index, message.getSeverity().name().substring(0, 1));
      item.setText(Column.TIME.index, formatTime(message.getTime()));
      item.setText(Column.PROCESS.index, message.getProcess());
      item.setText(Column.TEXT.index, lines[0]);
      // Additional lines
      for (int l = 1; l < lines.length; l++) {
        TreeItem line = newItem(item, severity);
        line.setText(Column.TEXT.index, lines[l]);
      }

      boolean groupLabelsRequired = ((lines.length - 1) +
          (message.getCauseCount() > 0 ? 1 : 0) +
          (message.getValuesCount() > 0 ? 1 : 0) +
          (message.getCallstackCount() > 0 ? 1 : 0)) > 1;


      if (!message.getTag().isEmpty()) {
        newItem(item, severity).setText(Column.TEXT.index, message.getTag());
      }
      // Exceptions
      for (Log.Cause cause : message.getCauseList()) {
        causeToTreeItem(item, severity, cause);
      }
      // Values
      if (message.getValuesCount() > 0) {
        TreeItem root = item;
        if (groupLabelsRequired) {
          root = newItem(item, severity);
          root.setText(Column.TEXT.index, "Values");
        }
        for (Log.Value value : message.getValuesList()) {
          TreeItem entry = newItem(root, severity);
          String name = value.getName();
          String val = Pods.unpod(value.getValue()).toString();
          entry.setText(Column.TEXT.index, "  " + name + " = " + val);
        }
      }
      // Callstack
      if (message.getCallstackCount() > 0) {
        TreeItem root = item;
        if (groupLabelsRequired) {
          root = newItem(item, severity);
          root.setText(Column.TEXT.index, "Call Stack");
        }
        for (Log.SourceLocation location : message.getCallstackList()) {
          locationToTreeItem(root, severity, location);
        }
      }
      while (tree.getItemCount() > MAX_ITEMS) {
        tree.getItem(0).dispose();
      }
    }
    // Too many new messages to display!
    // Try again next update.
    Widgets.scheduleIfNotDisposed(this, this::updateTree);
  }

  private void causeToTreeItem(TreeItem parent, Log.Severity severity, Log.Cause trace) {
    TreeItem root = newItem(parent, severity);
    root.setText(3, trace.getMessage());
    for (Log.SourceLocation location : trace.getCallstackList()) {
      locationToTreeItem(root, severity, location);
    }
  }

  private void locationToTreeItem(TreeItem parent, Log.Severity severity, Log.SourceLocation loc) {
    TreeItem line = newItem(parent, severity);
    String text;
    if (loc.getFile().isEmpty()) {
      text = "Unknon Source";
    } else {
      text = loc.getFile() + (loc.getLine() != 0 ? ":" + loc.getLine() : "");
    }
    if (!loc.getMethod().isEmpty()) {
      text = loc.getMethod() + "(" + text + ")";
    }
    line.setText(Column.TEXT.index, "  " + text);
  }

  private static String formatTime(Timestamp time) {
    Date date = new Date(time.getSeconds() * 1000);
    Calendar calendar = Calendar.getInstance();
    calendar.setTime(date);
    int hour = calendar.get(Calendar.HOUR);
    int minute = calendar.get(Calendar.MINUTE);
    int second = calendar.get(Calendar.SECOND);
    int millis = time.getNanos() / 1000000;
    return String.format("%02d:%02d:%02d.%03d", hour, minute, second, millis);
  }

  private TreeItem newItem(Tree parent, Log.Severity severity) {
    TreeItem item = new TreeItem(parent, 0);
    item.setBackground(severityBackgroundColor(severity));
    item.setForeground(severityForegroundColor(severity));
    return item;
  }

  private TreeItem newItem(TreeItem parent, Log.Severity severity) {
    TreeItem item = new TreeItem(parent, 0);
    item.setBackground(severityBackgroundColor(severity));
    item.setForeground(severityForegroundColor(severity));
    return item;
  }

  private Color severityBackgroundColor(Log.Severity severity) {
    switch (severity) {
      case Verbose: return theme.logVerboseBackground();
      case Debug: return theme.logDebugBackground();
      case Info: return theme.logInfoBackground();
      case Warning: return theme.logWarningBackground();
      case Error: return theme.logErrorBackground();
      case Fatal: return theme.logFatalBackground();
      default: return theme.logInfoBackground();
    }
  }

  private Color severityForegroundColor(Log.Severity severity) {
    switch (severity) {
      case Verbose: return theme.logVerboseForeground();
      case Debug: return theme.logDebugForeground();
      case Info: return theme.logInfoForeground();
      case Warning: return theme.logWarningForeground();
      case Error: return theme.logErrorForeground();
      case Fatal: return theme.logFatalForeground();
      default: return theme.logInfoForeground();
    }
  }

  @Override
  public Control getControl() {
    return this;
  }
}
