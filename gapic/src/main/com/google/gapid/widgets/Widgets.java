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

import static com.google.gapid.util.GeoUtils.right;
import static com.google.gapid.util.GeoUtils.top;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.Models;
import com.google.gapid.server.Client;
import com.google.gapid.util.OS;
import com.google.gapid.views.AtomEditor;

import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.jface.viewers.ColumnLabelProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.DoubleClickEvent;
import org.eclipse.jface.viewers.IDoubleClickListener;
import org.eclipse.jface.viewers.IStructuredSelection;
import org.eclipse.jface.viewers.ITreeSelection;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.jface.viewers.TableViewerColumn;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerComparator;
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
import org.eclipse.swt.widgets.Combo;
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
import org.eclipse.swt.widgets.Table;
import org.eclipse.swt.widgets.TableColumn;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;
import org.eclipse.swt.widgets.Widget;

import java.util.Comparator;
import java.util.List;
import java.util.concurrent.Callable;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Consumer;
import java.util.function.Function;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Widget utilities.
 */
public class Widgets {
  private static final Logger LOG = Logger.getLogger(Widgets.class.getName());

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
    AtomEditor editor = new AtomEditor(client, models, theme);
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

  public static void schedule(Display display, Runnable run) {
    if (enqueue(run)) {
      long start = System.currentTimeMillis();
      display.asyncExec(() -> {
        Runnable[] work = drainQueue();
        if (work.length > 1 && LOG.isLoggable(Level.FINE)) {
          LOG.log(Level.FINE, "Processing a batch of {0} runnables after a delay of {1}ms",
              new Object[] { work.length, System.currentTimeMillis() - start });
        }
        for (Runnable r : work) {
          r.run();
        }
      });
    }
  }

  public static void scheduleIfNotDisposed(Widget widget, Runnable run) {
    if (!widget.isDisposed()) {
      schedule(widget.getDisplay(), () -> ifNotDisposed(widget, run));
    }
  }

  public static <T> ListenableFuture<T> submitIfNotDisposed(Widget widget, Callable<T> callable) {
    if (widget.isDisposed()) {
      return Futures.immediateCancelledFuture();
    }

    SettableFuture<T> result = SettableFuture.create();
    schedule(widget.getDisplay(), () -> {
      if (widget.isDisposed() || result.isCancelled()) {
        result.cancel(true);
      } else {
        try {
          result.set(callable.call());
        } catch (Exception e) {
          result.setException(e);
        }
      }
    });
    return result;
  }

  private static final List<Runnable> queue = Lists.newArrayList();
  private static boolean enqueue(Runnable run) {
    synchronized (queue) {
      queue.add(run);
      return queue.size() == 1;
    }
  }

  private static Runnable[] drainQueue() {
    synchronized (queue) {
      Runnable[] work = queue.toArray(new Runnable[queue.size()]);
      queue.clear();
      return work;
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
      Balloon.createAndShow(bar, createContents, new Point(right(b) + 2, top(b)));
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

  public static Label createBoldLabel(Composite parent, String label) {
    Label result = createLabel(parent, label);
    result.setFont(JFaceResources.getFontRegistry().getBold(JFaceResources.DEFAULT_FONT));
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

    if (OS.isMac) {
      result.addListener(SWT.KeyUp, e -> {
        if ((e.stateMask & (SWT.CONTROL | SWT.COMMAND)) != 0) {
          switch (e.keyCode) {
            case 'c': result.copy(); break;
            case 'v': result.paste(); break;
            case 'x': result.cut(); break;
          }
        }
      });
    }
    return result;
  }

  public static Text createTextbox(Composite parent, String text) {
    return createTextbox(parent, SWT.SINGLE | SWT.BORDER, text);
  }

  public static Text createTextbox(Composite parent, int style, String text) {
    Text result = new Text(parent, style);
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

  public static TableViewer createTableViewer(Composite parent, int style) {
    TableViewer table = new VisibilityTrackingTableViewer(new Table(parent, style));
    table.getTable().setHeaderVisible(true);
    table.getTable().setLinesVisible(true);
    table.setUseHashlookup(true);
    return table;
  }

  public static TableViewerColumn createTableColumn(TableViewer viewer, String title) {
    TableViewerColumn result = new TableViewerColumn(viewer, SWT.NONE);
    TableColumn column = result.getColumn();
    column.setText(title);
    column.setResizable(true);
    return result;
  }

  public static <T> TableViewerColumn createTableColumn(
      TableViewer viewer, String title, Function<T, String> labelProvider) {
    return createTableColumn(viewer, title, labelProvider, d -> null);
  }

  public static <T> TableViewerColumn createTableColumn(TableViewer viewer, String title,
      Function<T, String> labelProvider, Function<T, Image> imageProvider) {
    TableViewerColumn column = createTableColumn(viewer, title);
    column.setLabelProvider(new ColumnLabelProvider() {
      @Override
      @SuppressWarnings("unchecked")
      public String getText(Object element) {
        return labelProvider.apply((T)element);
      }

      @Override
      @SuppressWarnings("unchecked")
      public Image getImage(Object element) {
        return imageProvider.apply((T)element);
      }
    });
    return column;
  }

  public static <T> ColumnAndComparator<T> createTableColumn(
      TableViewer viewer, String title, Function<T, String> labelProvider, Comparator<T> comp) {
    return new ColumnAndComparator<>(createTableColumn(viewer, title, labelProvider), comp);
  }

  public static <T> ColumnAndComparator<T> createTableColumn(TableViewer viewer, String title,
      Function<T, String> labelProvider, Function<T, Image> imageProvider, Comparator<T> comp) {
    return new ColumnAndComparator<>(
        createTableColumn(viewer, title, labelProvider, imageProvider), comp);
  }

  @SafeVarargs
  public static <T> void sorting(
      TableViewer table, ColumnAndComparator<T>... columns) {
    int[] sortState = { 0, SWT.UP };
    for (int i = 0; i < columns.length; i++) {
      final int idx = i;
      columns[idx].getColumn().addListener(SWT.Selection, e -> {
        table.getTable().setSortColumn(columns[idx].getColumn());
        if (idx == sortState[0]) {
          sortState[1] = (sortState[1] == SWT.UP) ? SWT.DOWN : SWT.UP;
          table.getTable().setSortDirection(sortState[1]);
        } else {
          table.getTable().setSortDirection(SWT.UP);
          sortState[0] = idx;
          sortState[1] = SWT.UP;
        }

        table.setComparator(columns[idx].getComparator(sortState[1] == SWT.DOWN));
      });
    }

    table.getTable().setSortColumn(columns[0].getColumn());
    table.getTable().setSortDirection(SWT.UP);
    table.setComparator(columns[0].getComparator(false));
  }

  public static void packColumns(Table table) {
    for (TableColumn column : table.getColumns()) {
      column.pack();
    }
  }

  public static class ColumnAndComparator<T> {
    public final TableViewerColumn column;
    public final Comparator<T> comparator;

    public ColumnAndComparator(TableViewerColumn column, Comparator<T> comparator) {
      this.column = column;
      this.comparator = comparator;
    }

    public TableColumn getColumn() {
      return column.getColumn();
    }

    public ViewerComparator getComparator(boolean reverse) {
      return new ViewerComparator() {
        @Override
        @SuppressWarnings("unchecked")
        public int compare(Viewer viewer, Object e1, Object e2) {
          return reverse ? comparator.compare((T)e2, (T)e1) :
            comparator.compare((T)e1, (T)e2);
        }
      };
    }
  }

  public static Group createGroup(Composite parent, String text) {
    Group group = new Group(parent, SWT.NONE);
    group.setLayout(new FillLayout(SWT.VERTICAL));
    group.setText(text);
    group.setFont(JFaceResources.getFontRegistry().getBold(JFaceResources.DEFAULT_FONT));
    return group;
  }

  /**
   * Do not use this if you intend to wrap the {@link Tree} in a {@link TreeViewer}. Instead use
   * {@link #createTreeForViewer(Composite, int)} and then {@link #createTreeViewer(Tree)}.
   */
  public static Tree createTree(Composite parent, int style) {
    Tree tree = new Tree(parent, style);
    tree.setLinesVisible(true);
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
    tree.addListener(SWT.MouseDoubleClick, e -> {
      if (tree.getSelectionCount() == 1) {
        TreeItem selection = tree.getSelection()[0];
        selection.setExpanded(!selection.getExpanded());
      }
    });
    return tree;
  }

  /**
   * Use this to create a {@link Tree} that you will later wrap in a {@link TreeViewer} using
   * {@link #createTreeViewer(Tree)}, otherwise use {@link #createTree(Composite, int)}.
   */
  public static Tree createTreeForViewer(Composite parent, int style) {
    Tree tree = new Tree(parent, style);
    tree.setLinesVisible(true);
    return tree;
  }

  public static TreeViewer createTreeViewer(Composite parent, int style) {
    return createTreeViewer(createTreeForViewer(parent, style));
  }

  public static TreeViewer createTreeViewer(Tree tree) {
    TreeViewer viewer = new VisibilityTrackingTreeViewer(tree);
    viewer.setUseHashlookup(true);
    tree.addListener(SWT.KeyDown, e -> {
      switch (e.keyCode) {
        case SWT.ARROW_LEFT:
        case SWT.ARROW_RIGHT:
          ITreeSelection selection = viewer.getStructuredSelection();
          if (selection.size() == 1) {
            viewer.setExpandedState(selection.getFirstElement(), e.keyCode == SWT.ARROW_RIGHT);
          }
          break;
      }
    });
    viewer.addDoubleClickListener(new IDoubleClickListener() {
      @Override
      public void doubleClick(DoubleClickEvent event) {
        IStructuredSelection selection = (IStructuredSelection)event.getSelection();
        if (selection.size() == 1) {
          Object element = selection.getFirstElement();
          viewer.setExpandedState(element, !viewer.getExpandedState(element));
        }
      }
    });
    return viewer;
  }

  public static Combo createDropDown(Composite parent) {
    Combo combo = new Combo(parent, SWT.READ_ONLY | SWT.DROP_DOWN);
    combo.setVisibleItemCount(10);
    return combo;
  }

  public static Combo createEditDropDown(Composite parent) {
    Combo combo = new Combo(parent, SWT.DROP_DOWN);
    combo.setVisibleItemCount(10);
    return combo;
  }

  public static ComboViewer createDropDownViewer(Composite parent) {
    return createDropDownViewer(createDropDown(parent));
  }

  public static ComboViewer createDropDownViewer(Combo combo) {
    ComboViewer viewer = new ComboViewer(combo);
    viewer.setUseHashlookup(true);
    return viewer;
  }

  public static Refresher withAsyncRefresh(Viewer viewer) {
    AtomicBoolean scheduled = new AtomicBoolean();
    return () -> ifNotDisposed(viewer.getControl(), () -> {
      if (!scheduled.getAndSet(true)) {
        viewer.getControl().getDisplay().timerExec(5, () -> {
          scheduled.set(false);
          ifNotDisposed(viewer.getControl(), viewer::refresh);
        });
      }
    });
  }

  public static interface Refresher {
    public void refresh();
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

  public static GridData withSizeHints(GridData data, int widthHint, int heightHint) {
    data.widthHint = widthHint;
    data.heightHint = heightHint;
    return data;
  }

  public static GridData withIndents(GridData data, int horizontalIndent, int verticalIndent) {
    data.horizontalIndent = horizontalIndent;
    data.verticalIndent = verticalIndent;
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

  public static RowLayout filling(RowLayout layout, boolean fill, boolean justify) {
    layout.fill = fill;
    layout.justify = justify;
    return layout;
  }

  public static void disposeAllChildren(Composite parent) {
    for (Control child : parent.getChildren()) {
      child.dispose();
    }
  }
}
