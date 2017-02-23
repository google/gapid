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
package com.google.gapid.widgets;

import com.google.gapid.models.Models;
import com.google.gapid.server.Client;

import org.eclipse.jface.viewers.DoubleClickEvent;
import org.eclipse.jface.viewers.IDoubleClickListener;
import org.eclipse.jface.viewers.IStructuredSelection;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.jface.viewers.TableViewerColumn;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.CTabFolder;
import org.eclipse.swt.custom.CTabItem;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Layout;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.MenuItem;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Spinner;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.TabItem;
import org.eclipse.swt.widgets.TableColumn;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.Widget;

import java.util.function.Consumer;

public class Widgets {
  public final Theme theme;
  public final CopyPaste copypaste;
  public final LoadingIndicator loading;
  public final AtomEditor editor;

  public Widgets(Theme theme, CopyPaste copypaste, LoadingIndicator loading, AtomEditor editor) {
    this.theme = theme;
    this.copypaste = copypaste;
    this.loading = loading;
    this.editor = editor;
  }

  public static Widgets create(Display display, Client client, Models models) {
    Theme theme = Theme.load(display);
    CopyPaste copypaste = new CopyPaste(display);
    LoadingIndicator loading = new LoadingIndicator(display, theme);
    AtomEditor editor = new AtomEditor(client, models);
    return new Widgets(theme, copypaste, loading, editor);
  }

  public void dispose() {
    copypaste.dispose();
    theme.dispose();
  }

  public static void ifNotDisposed(Widget widget, Runnable run) {
    if (!widget.isDisposed()) {
      run.run();
    }
  }

  public static void redrawIfNotDisposed(Control control) {
    if (!control.isDisposed()) {
      control.redraw();
    }
  }

  public static void scheduleIfNotDisposed(Widget widget, Runnable run) {
    if (!widget.isDisposed()) {
      widget.getDisplay().asyncExec(() -> ifNotDisposed(widget, run));
    }
  }

  public static void scheduleIfNotDisposed(Widget widget, int milliseconds, Runnable run) {
    if (!widget.isDisposed()) {
      widget.getDisplay().timerExec(milliseconds, () -> ifNotDisposed(widget, run));
    }
  }

  public static CTabFolder createTabFolder(Composite parent) {
    CTabFolder folder = new CTabFolder(parent, SWT.BORDER | SWT.FLAT | SWT.TOP);
    return folder;
  }

  public static CTabItem createTabItem(CTabFolder folder, String label) {
    CTabItem item = new CTabItem(folder, SWT.NONE);
    item.setText(label);
    return item;
  }

  public static CTabItem createTabItem(CTabFolder folder, String label, Control contents) {
    CTabItem item = createTabItem(folder, label);
    item.setControl(contents);
    return item;
  }

  public static TabFolder createStandardTabFolder(Composite parent) {
    TabFolder folder = new TabFolder(parent, SWT.BORDER | SWT.TOP);
    return folder;
  }

  public static TabItem createStandardTabItem(TabFolder folder, String label) {
    TabItem item = new TabItem(folder, SWT.NONE);
    item.setText(label);
    return item;
  }

  public static TabItem createStandardTabItem(TabFolder folder, String label, Control contents) {
    TabItem item = createStandardTabItem(folder, label);
    item.setControl(contents);
    return item;
  }

  public static ToolItem createToolItem(ToolBar bar, Image image, Listener listener, String tip) {
    return createToolItem(bar, SWT.PUSH, image, listener, tip);
  }

  public static ToolItem createToggleToolItem(
      ToolBar bar, Image image, Listener listener, String tip) {
    return createToolItem(bar, SWT.CHECK, image, listener, tip);
  }

  public static ToolItem createDropDownToolItem(
      ToolBar bar, Image image, Listener listener, String tip) {
    return createToolItem(bar, SWT.DROP_DOWN, image, listener, tip);
  }

  public static ToolItem createBaloonToolItem(
      ToolBar bar, Image image, Consumer<Shell> createContents, String tip) {
    return createToolItem(bar, SWT.PUSH, image, e -> {
      Rectangle b = ((ToolItem)e.widget).getBounds();
      Balloon.createAndShow(bar, createContents, new Point(b.x + b.width + 2, b.y));
    }, tip);
  }

  private static ToolItem createToolItem(
      ToolBar bar, int style, Image image, Listener listener, String tip) {
    ToolItem item = new ToolItem(bar, style);
    item.setImage(image);
    item.addListener(SWT.Selection, listener);
    item.setToolTipText(tip);
    return item;
  }

  public static ToolItem createSeparator(ToolBar bar) {
    return new ToolItem(bar, SWT.SEPARATOR);
  }

  public static void exclusiveSelection(ToolItem... items) {
    Listener listener = e -> {
      for (ToolItem item : items) {
        item.setSelection(e.widget == item);
      }
    };
    for (ToolItem item : items) {
      item.addListener(SWT.Selection, listener);
      item.setSelection(false);
    }
    items[0].setSelection(true);
  }

  public static Label createLabel(Composite parent, String label) {
    Label result = new Label(parent, SWT.NONE);
    result.setText(label);
    return result;
  }

  public static Label createLabel(Composite parent, String label, Image image) {
    Label result = createLabel(parent, label);
    result.setImage(image);
    return result;
  }

  public static Button createCheckbox(Composite parent, boolean checked) {
    Button button = new Button(parent, SWT.CHECK);
    button.setSelection(checked);
    return button;
  }

  public static Button createCheckbox(Composite parent, boolean checked, Listener listener) {
    Button button = createCheckbox(parent, checked);
    button.addListener(SWT.Selection, listener);
    return button;
  }

  public static Button createCheckbox(Composite parent, String label, boolean checked) {
    Button button = createCheckbox(parent, checked);
    button.setText(label);
    return button;
  }

  public static Button createCheckbox(
      Composite parent, String label, boolean checked, Listener listener) {
    Button button = createCheckbox(parent, checked, listener);
    button.setText(label);
    return button;
  }

  public static MenuItem createMenuItem(
      Menu parent, String text, int accelerator, Listener listener) {
    MenuItem item = new MenuItem(parent, SWT.PUSH);
    item.setText(text);
    item.setAccelerator(accelerator);
    item.addListener(SWT.Selection, listener);
    return item;
  }

  public static MenuItem createSubMenu(Menu parent, String text, Menu child) {
    MenuItem item = new MenuItem(parent, SWT.CASCADE);
    item.setText(text);
    item.setMenu(child);
    return item;
  }

  public static Button createButton(Composite parent, String text, Listener listener) {
    Button result = new Button(parent, SWT.PUSH);
    result.setText(text);
    result.addListener(SWT.Selection, listener);
    return result;
  }

  public static Spinner createSpinner(Composite parent, int value, int min, int max) {
    Spinner result = new Spinner(parent, SWT.BORDER);
    result.setMinimum(min);
    result.setMaximum(max);
    result.setSelection(value);
    return result;
  }

  public static Text createTextbox(Composite parent, String text) {
    Text result = new Text(parent, SWT.SINGLE | SWT.BORDER);
    result.setText(text);
    return result;
  }

  public static Text createTextarea(Composite parent, String text) {
    Text result = new Text(parent, SWT.MULTI | SWT.BORDER | SWT.V_SCROLL | SWT.H_SCROLL);
    result.setText(text);
    return result;
  }

  public static Link createLink(Composite parent, String text, Listener listener) {
    Link link = new Link(parent, SWT.NONE);
    link.setText(text);
    link.addListener(SWT.Selection, listener);
    return link;
  }

  public static TableViewerColumn createTableColum(TableViewer viewer, String title) {
    TableViewerColumn result = new TableViewerColumn(viewer, SWT.NONE);
    TableColumn column = result.getColumn();
    column.setText(title);
    column.setResizable(true);
    return result;
  }

  public static Group createGroup(Composite parent, String text) {
    Group group = new Group(parent, SWT.NONE);
    group.setLayout(new FillLayout(SWT.VERTICAL));
    group.setText(text);
    return group;
  }

  public static Tree createTree(Composite parent, int style) {
    Tree tree = new Tree(parent, style);
    tree.addListener(SWT.KeyDown, e -> {
      switch (e.keyCode) {
        case SWT.ARROW_LEFT:
        case SWT.ARROW_RIGHT:
          if (tree.getSelectionCount() == 1) {
            tree.getSelection()[0].setExpanded(e.keyCode == SWT.ARROW_RIGHT);
          }
          break;
      }
    });
    return tree;
  }

  public static Composite createComposite(Composite parent, Layout layout) {
    return createComposite(parent, layout, SWT.NONE);
  }

  public static Composite createComposite(Composite parent, Layout layout, int style) {
    Composite composite = new Composite(parent, style);
    composite.setLayout(layout);
    return composite;
  }

  public static <C extends Control> C withLayoutData(C control, Object layoutData) {
    control.setLayoutData(layoutData);
    return control;
  }

  public static GridData withSpans(GridData data, int colSpan, int rowSpan) {
    data.horizontalSpan = colSpan;
    data.verticalSpan = rowSpan;
    return data;
  }

  public static FillLayout withMargin(FillLayout layout, int marginWidth, int marginHeight) {
    layout.marginWidth = marginWidth;
    layout.marginHeight = marginHeight;
    return layout;
  }

  public static GridLayout withMargin(GridLayout layout, int marginWidth, int marginHeight) {
    layout.marginWidth = marginWidth;
    layout.marginHeight = marginHeight;
    layout.horizontalSpacing = marginWidth;
    layout.verticalSpacing = marginHeight;
    return layout;
  }

  public static RowLayout centered(RowLayout layout) {
    layout.center = true;
    return layout;
  }

  public static void expandOnDoubleClick(TreeViewer viewer) {
    viewer.addDoubleClickListener(new IDoubleClickListener() {
      @Override
      public void doubleClick(DoubleClickEvent event) {
        IStructuredSelection selection = (IStructuredSelection)event.getSelection();
        Object element = selection.getFirstElement();
        viewer.setExpandedState(element, !viewer.getExpandedState(element));
      }
    });
  }
}
