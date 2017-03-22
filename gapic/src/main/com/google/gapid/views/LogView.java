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

import com.google.gapid.proto.log.Log;
import com.google.gapid.util.Logging;
import com.google.gapid.util.Pods;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import com.google.protobuf.Timestamp;
import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.*;

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
  private final int MAX_ITEMS = 1000;
  private final int MAX_NEW_ITEMS_PER_UPDATE = 100;
  private Logging.MessageIterator messageIterator;

  private enum Column {
    SEVERITY(0, 35, "Severity"),
    TIME(1, 100, "Time"),
    PROCESS(2, 60, "Process"),
    TEXT(3, 300, "Text"),
    TAG(4, 100, "Tag");

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

    tree = new Tree(this, SWT.MULTI | SWT.H_SCROLL | SWT.V_SCROLL | SWT.FULL_SELECTION);
    tree.setHeaderVisible(true);
    tree.setFont(JFaceResources.getFont(JFaceResources.TEXT_FONT));
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
      item.setText(Column.TAG.index, message.getTag());
      // Additional lines
      for (int l = 1; l < lines.length; l++) {
        TreeItem line = newItem(item, severity);
        line.setText(Column.TEXT.index, lines[l]);
      }
      // Values
      if (message.getValuesCount() > 0) {
        TreeItem values = newItem(item, severity);
        values.setText(2, "Values");
        for (Log.Value value : message.getValuesList()) {
          TreeItem entry = newItem(values, severity);
          String name = value.getName();
          String val = Pods.unpod(value.getValue()).toString();
          entry.setText(2, name);
          entry.setText(Column.TEXT.index, val);
        }
      }
      // Callstack
      if (message.getCallstackCount() > 0) {
        TreeItem callstack = newItem(item, severity);
        callstack.setText(2, "Call Stack");
        for (Log.SourceLocation location : message.getCallstackList()) {
          TreeItem line = newItem(callstack, severity);
          line.setText(Column.TEXT.index, String.format("%s:%d", location.getFile(), location.getLine()));
        }
      }
      while (tree.getItemCount() > MAX_ITEMS) {
        tree.getTopItem().dispose();
      }
    }
    // Too many new messages to display!
    // Try again next update.
    Widgets.scheduleIfNotDisposed(this, this::updateTree);
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
      case Verbose: return theme.verboseBackground();
      case Debug: return theme.debugBackground();
      case Info: return theme.infoBackground();
      case Warning: return theme.warningBackground();
      case Error: return theme.errorBackground();
      case Fatal: return theme.fatalBackground();
      default: return theme.infoBackground();
    }
  }

  private Color severityForegroundColor(Log.Severity severity) {
    switch (severity) {
      case Verbose: return theme.verboseForeground();
      case Debug: return theme.debugForeground();
      case Info: return theme.infoForeground();
      case Warning: return theme.warningForeground();
      case Error: return theme.errorForeground();
      case Fatal: return theme.fatalForeground();
      default: return theme.infoForeground();
    }
  }

  @Override
  public Control getControl() {
    return this;
  }
}
