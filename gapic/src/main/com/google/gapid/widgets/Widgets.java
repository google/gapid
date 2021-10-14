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
import static java.util.logging.Level.WARNING;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.Models;
import com.google.gapid.server.Client;
import com.google.gapid.util.OS;
import com.google.gapid.views.CommandEditor;
import com.google.gapid.views.Experiments;

import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.jface.viewers.CheckboxTableViewer;
import org.eclipse.jface.viewers.CheckboxTreeViewer;
import org.eclipse.jface.viewers.ColumnLabelProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.DoubleClickEvent;
import org.eclipse.jface.viewers.IDoubleClickListener;
import org.eclipse.jface.viewers.IStructuredSelection;
import org.eclipse.jface.viewers.ITreeSelection;
import org.eclipse.jface.viewers.OwnerDrawLabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.jface.viewers.TableViewerColumn;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerColumn;
import org.eclipse.jface.viewers.ViewerComparator;
import org.eclipse.swt.SWT;
import org.eclipse.swt.SWTError;
import org.eclipse.swt.browser.Browser;
import org.eclipse.swt.custom.CTabFolder;
import org.eclipse.swt.custom.CTabItem;
import org.eclipse.swt.custom.ST;
import org.eclipse.swt.custom.ScrolledComposite;
import org.eclipse.swt.custom.StyledText;
import org.eclipse.swt.graphics.Color;
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
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Layout;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.MenuItem;
import org.eclipse.swt.widgets.ProgressBar;
import org.eclipse.swt.widgets.Sash;
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
import org.eclipse.swt.widgets.TreeColumn;
import org.eclipse.swt.widgets.TreeItem;
import org.eclipse.swt.widgets.Widget;

import java.util.Arrays;
import java.util.Comparator;
import java.util.List;
import java.util.concurrent.Callable;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Consumer;
import java.util.function.Function;
import java.util.function.IntConsumer;
import java.util.function.Supplier;
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
  public final CommandEditor editor;
  public final Experiments experiments;

  public Widgets(Theme theme, CopyPaste copypaste, LoadingIndicator loading, CommandEditor editor, Experiments experiments) {
    this.theme = theme;
    this.copypaste = copypaste;
    this.loading = loading;
    this.editor = editor;
    this.experiments = experiments;
  }

  public static Widgets create(Display display, Theme theme, Client client, Models models) {
    CopyPaste copypaste = new CopyPaste(display);
    LoadingIndicator loading = new LoadingIndicator(display, theme);
    CommandEditor editor = new CommandEditor(client, models, theme);
    Experiments experiments = new Experiments(models, theme);
    return new Widgets(theme, copypaste, loading, editor, experiments);
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

  /**
   * Calls the given suppliers {@link Supplier#get get} method periodically, until either the
   * given widget is disposed, or the supplier returns {@code false}.
   */
  public static void scheduleUntilDisposed(
      Widget widget, int milliseconds, Supplier<Boolean> run) {
    scheduleIfNotDisposed(widget, milliseconds, () -> {
      if (run.get()) {
        scheduleUntilDisposed(widget, milliseconds, run);
      }
    });
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

  /**
   * Returns a function that allows manually selecting the ith item.
   */
  public static IntConsumer exclusiveSelection(ToolItem... items) {
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

    return index -> {
      for (int i = 0; i < items.length; i++) {
        items[i].setSelection(i == index);
      }
    };
  }

  public static IntConsumer exclusiveSelection(List<ToolItem> items) {
    Listener listener = e -> {
      for (ToolItem item : items) {
        item.setSelection(e.widget == item);
      }
    };
    for (ToolItem item : items) {
      item.addListener(SWT.Selection, listener);
      item.setSelection(false);
    }
    items.get(0).setSelection(true);

    return index -> {
      for (int i = 0; i < items.size(); i++) {
        items.get(i).setSelection(i == index);
      }
    };
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
    // Workaround for https://bugs.eclipse.org/bugs/show_bug.cgi?id=561592 - set label first.
    Button button = new Button(parent, SWT.CHECK);
    button.setText(label);
    button.setSelection(checked);
    return button;
  }

  public static Button createCheckbox(
      Composite parent, String label, boolean checked, Listener listener) {
    // Workaround for https://bugs.eclipse.org/bugs/show_bug.cgi?id=561592 - set label first.
    Button button = new Button(parent, SWT.CHECK);
    button.setText(label);
    button.setSelection(checked);
    button.addListener(SWT.Selection, listener);
    return button;
  }

  public static MenuItem createMenuItem(Menu parent, String text, Listener listener) {
    MenuItem item = new MenuItem(parent, SWT.PUSH);
    item.setText(text);
    item.addListener(SWT.Selection, listener);
    return item;
  }

  public static MenuItem createMenuItem(
      Menu parent, String text, int accelerator, Listener listener) {
    MenuItem item = new MenuItem(parent, SWT.PUSH);
    item.setText(text);
    item.setAccelerator(accelerator);
    item.addListener(SWT.Selection, listener);
    return item;
  }

  public static MenuItem createCheckMenuItem(Menu parent, String text, int accelerator, Listener listener) {
    MenuItem item = new MenuItem(parent, SWT.CHECK);
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

  public static Button createButton(Composite parent, int style, String text, Color color, Listener listener) {
    Button result = new Button(parent, style | SWT.PUSH);
    result.setText(text);
    result.setBackground(color);
    result.addListener(SWT.Selection, listener);
    return result;
  }

  public static Button createButtonWithImage(Composite parent, Image image, Listener listener) {
    Button result = new Button(parent, SWT.PUSH);
    result.setImage(image);
    result.addListener(SWT.Selection, listener);
    return result;
  }

  public static Spinner createSpinner(Composite parent, int value, int min, int max) {
    Spinner result = new Spinner(parent, SWT.BORDER);
    // Avoid not being able to update the minimum value.
    // According to SWT's API, min will be ignored if it's greater then the previous max.
    if (min > result.getMaximum()) {
      result.setMaximum(max);
      result.setMinimum(min);
    } else {
      result.setMinimum(min);
      result.setMaximum(max);
    }
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

  public static Spinner createSpinner(Composite parent, int value, int min, int max, Listener listener) {
    Spinner result = createSpinner(parent, value, min, max);
    result.addListener(SWT.Selection, listener);
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

  public static StyledText createSelectableLabel(Composite parent, String text) {
    StyledText result = new StyledText(parent, SWT.READ_ONLY | SWT.SINGLE);
    result.setText(text);
    result.setEditable(false);
    result.setBackground(parent.getShell().getBackground());
    result.setKeyBinding(ST.SELECT_ALL, ST.SELECT_ALL);
    // Set caret to null by default to hide caret.
    result.setCaret(null);
    // Clear selection when out of focus.
    result.addListener(SWT.FocusOut, new Listener() {
      @Override
      public void handleEvent(Event e) {
        result.setSelection(0);
      }
    });
    return result;
  }

  public static Link createLink(Composite parent, String text, Listener listener) {
    Link link = new Link(parent, SWT.NONE);
    link.setText(text);
    link.addListener(SWT.Selection, listener);
    return link;
  }

  /**
   * Use this to create a {@link Table} that you will later wrap in a {@link TableViewer} using
   * {@link #createTableViewer(Table)}.
   * If using the table for {@link #createCheckboxTableViewer(Table)}, style needs to contain
   * {@code SWT.CHECK}.
   */
  public static Table createTableForViewer(Composite parent, int style) {
    Table table = new Table(parent, style);
    table.setHeaderVisible(true);
    table.setLinesVisible(true);
    return table;
  }

  public static TableViewer createTableViewer(Composite parent, int style) {
    return createTableViewer(createTableForViewer(parent, style));
  }

  public static TableViewer createTableViewer(Table table) {
    return initTableViewer(new TableViewer(table));
  }

  public static CheckboxTableViewer createCheckboxTableViewer(Composite parent, int style) {
    return createCheckboxTableViewer(createTableForViewer(parent, style | SWT.CHECK));
  }

  public static CheckboxTableViewer createCheckboxTableViewer(Table table) {
    return initTableViewer(new CheckboxTableViewer(table));
  }

  private static <T extends TableViewer> T initTableViewer(T viewer) {
    viewer.setUseHashlookup(true);
    return viewer;
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

  private static final int IMAGE_MARGIN = 2;
  public static <T> TableViewerColumn createLazyImageTableColumn(TableViewer viewer, String title,
      Function<T, String> labelProvider, Function<T, Image> imageProvider, int imageSize) {
    TableViewerColumn column = createTableColumn(viewer, title);
    column.setLabelProvider(new OwnerDrawLabelProvider() {
      @Override
      @SuppressWarnings("unchecked")
      protected void measure(Event event, Object element) {
        String label = labelProvider.apply((T)element);
        Point size = event.gc.textExtent(label, SWT.DRAW_TRANSPARENT);
        event.width = 2 * IMAGE_MARGIN + imageSize + size.x;
        event.height = Math.max(imageSize + 2 * IMAGE_MARGIN, size.y);
      }

      @Override
      @SuppressWarnings("unchecked")
      protected void paint(Event event, Object element) {
        String label = labelProvider.apply((T)element);
        Image image = imageProvider.apply((T)element);
        if (image != null) {
          Rectangle size = image.getBounds();
          event.gc.drawImage(image, 0, 0, size.width, size.height,
              event.x + IMAGE_MARGIN, event.y + IMAGE_MARGIN, imageSize, imageSize);
        }

        Point size = event.gc.textExtent(label, SWT.DRAW_TRANSPARENT);
        event.gc.drawText(label, event.x + 2 * IMAGE_MARGIN + imageSize,
            event.y + (imageSize + 2 * IMAGE_MARGIN - size.y) / 2, SWT.DRAW_TRANSPARENT);
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

  public static <T> ColumnAndComparator<T> createLazyImageTableColumn(TableViewer viewer,
      String title, Function<T, String> labelProvider, Function<T, Image> imageProvider,
      int imageSize, Comparator<T> comp) {
    return new ColumnAndComparator<>(
        createLazyImageTableColumn(viewer, title, labelProvider, imageProvider, imageSize), comp);
  }

  @SafeVarargs
  public static <T> void sorting(TableViewer table, ColumnAndComparator<T>... columns) {
    sorting(table, Arrays.asList(columns));
  }

  public static <T> void sorting(TableViewer table, List<ColumnAndComparator<T>> columns) {
    int[] sortState = { 0, SWT.UP };
    for (int i = 0; i < columns.size(); i++) {
      final int idx = i;
      final ColumnAndComparator<T> column = columns.get(i);
      column.getTableColumn().addListener(SWT.Selection, e -> {
        table.getTable().setSortColumn(column.getTableColumn());
        if (idx == sortState[0]) {
          sortState[1] = (sortState[1] == SWT.UP) ? SWT.DOWN : SWT.UP;
          table.getTable().setSortDirection(sortState[1]);
        } else {
          table.getTable().setSortDirection(SWT.UP);
          sortState[0] = idx;
          sortState[1] = SWT.UP;
        }

        table.setComparator(column.getComparator(sortState[1] == SWT.DOWN));
      });
    }

    table.getTable().setSortColumn(columns.get(0).getTableColumn());
    table.getTable().setSortDirection(SWT.UP);
    table.setComparator(columns.get(0).getComparator(false));
  }

  public static void packColumns(Table table) {
    for (TableColumn column : table.getColumns()) {
      column.pack();
    }
  }

  public static class ColumnAndComparator<T> {
    public final ViewerColumn column;
    public final Comparator<T> comparator;

    public ColumnAndComparator(TableViewerColumn column, Comparator<T> comparator) {
      this.column = column;
      this.comparator = comparator;
    }

    public ColumnAndComparator(TreeViewerColumn column, Comparator<T> comparator) {
      this.column = column;
      this.comparator = comparator;
    }

    public TableColumn getTableColumn() {
      return ((TableViewerColumn)column).getColumn();
    }

    public TreeColumn getTreeColumn() {
      return ((TreeViewerColumn)column).getColumn();
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
    return createGroup(parent, text, new FillLayout(SWT.VERTICAL));
  }

  public static Group createGroup(Composite parent, String text, Layout layout) {
    Group group = new Group(parent, SWT.NONE);
    group.setLayout(layout);
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
   * If using the tree for {@link #createCheckboxTreeViewer(Tree)}, style needs to contain
   * {@code SWT.CHECK}.
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
    return initTreeViewer(new VisibilityTrackingTreeViewer(tree));
  }

  public static CheckboxTreeViewer createCheckboxTreeViewer(Composite parent, int style) {
    return createCheckboxTreeViewer(createTreeForViewer(parent, style | SWT.CHECK));
  }

  public static CheckboxTreeViewer createCheckboxTreeViewer(Tree tree) {
    return initTreeViewer(new CheckboxTreeViewer(tree));
  }

  private static <T extends TreeViewer> T initTreeViewer(T viewer) {
    viewer.setUseHashlookup(true);
    viewer.getTree().addListener(SWT.KeyDown, e -> {
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

  public static TreeViewerColumn createTreeColumn(TreeViewer viewer, String title) {
    TreeViewerColumn result = new TreeViewerColumn(viewer, SWT.NONE);
    TreeColumn column = result.getColumn();
    column.setText(title);
    column.setResizable(true);
    return result;
  }

  public static <T> TreeViewerColumn createTreeColumn(
      TreeViewer viewer, String title, Function<T, String> labelProvider) {
    return createTreeColumn(viewer, title, labelProvider, d -> null);
  }

  public static <T> TreeViewerColumn createTreeColumn(TreeViewer viewer, String title,
      Function<T, String> labelProvider, Function<T, Image> imageProvider) {
    TreeViewerColumn column = createTreeColumn(viewer, title);
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

  public static <T> ColumnAndComparator<T> createTreeColumn(
      TreeViewer viewer, String title, Function<T, String> labelProvider, Comparator<T> comp) {
    return new ColumnAndComparator<>(createTreeColumn(viewer, title, labelProvider), comp);
  }

  public static <T> ColumnAndComparator<T> createTreeColumn(TreeViewer viewer, String title,
      Function<T, String> labelProvider, Function<T, Image> imageProvider, Comparator<T> comp) {
    return new ColumnAndComparator<>(
        createTreeColumn(viewer, title, labelProvider, imageProvider), comp);
  }

  @SafeVarargs
  public static <T> void sorting(TreeViewer tree, ColumnAndComparator<T>... columns) {
    sorting(tree, Arrays.asList(columns));
  }

  public static <T> void sorting(TreeViewer tree, List<ColumnAndComparator<T>> columns) {
    int[] sortState = { 0, SWT.UP };
    for (int i = 0; i < columns.size(); i++) {
      final int idx = i;
      final ColumnAndComparator<T> column = columns.get(i);
      column.getTreeColumn().addListener(SWT.Selection, e -> {
        tree.getTree().setSortColumn(column.getTreeColumn());
        if (idx == sortState[0]) {
          sortState[1] = (sortState[1] == SWT.UP) ? SWT.DOWN : SWT.UP;
          tree.getTree().setSortDirection(sortState[1]);
        } else {
          tree.getTree().setSortDirection(SWT.UP);
          sortState[0] = idx;
          sortState[1] = SWT.UP;
        }

        tree.setComparator(column.getComparator(sortState[1] == SWT.DOWN));
      });
    }

    tree.getTree().setSortColumn(columns.get(0).getTreeColumn());
    tree.getTree().setSortDirection(SWT.UP);
    tree.setComparator(columns.get(0).getComparator(false));
  }

  public static void packColumns(Tree tree) {
    for (TreeColumn column : tree.getColumns()) {
      column.pack();
    }
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
    return withAsyncRefresh(viewer, () -> {/* do nothing */});
  }

  public static Refresher withAsyncRefresh(Viewer viewer, Runnable onRefresh) {
    AtomicBoolean scheduled = new AtomicBoolean();
    return () -> ifNotDisposed(viewer.getControl(), () -> {
      if (!scheduled.getAndSet(true)) {
        viewer.getControl().getDisplay().timerExec(5, () -> {
          scheduled.set(false);
          ifNotDisposed(viewer.getControl(), () -> {
            viewer.refresh();
            onRefresh.run();
          });
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

  public static Sash createHorizontalSash(Composite parent, Listener selectionListener) {
    return createSash(parent, SWT.HORIZONTAL, selectionListener);
  }

  public static Sash createVerticalSash(Composite parent, Listener selectionListener) {
    return createSash(parent, SWT.VERTICAL, selectionListener);
  }

  public static Sash createSash(Composite parent, int style, Listener selectionListener) {
    Sash sash = new Sash(parent, style);
    sash.addListener(SWT.Selection, selectionListener);
    return sash;
  }

  public static ScrolledComposite createScrolledComposite(Composite parent, Layout layout, int style) {
    ScrolledComposite composite = new ScrolledComposite(parent, style);
    composite.setLayout(layout);
    return composite;
  }

  public static Control createBrowser(Composite parent, String html) {
    try {
      Browser browser = new Browser(parent, SWT.NONE);
      browser.setText(html);
      return browser;
    } catch (SWTError e) {
      LOG.log(WARNING, "Failed to create browser widget", e);
    }

    // Failed to create browser, show as de-HTMLed text.
    Text text = new Text(
        parent, SWT.MULTI | SWT.READ_ONLY | SWT.BORDER | SWT.H_SCROLL | SWT.V_SCROLL);
    text.setText(html.replaceAll("<[^>]+>", ""));
    return text;
  }

  public static ProgressBar createProgressBar(Composite parent, int totalWork) {
    ProgressBar bar = new ProgressBar(parent, SWT.SMOOTH);
    bar.setMaximum(totalWork);
    bar.setMinimum(0);
    bar.setSelection(0);
    return bar;
  }

  /**
   * Recursively adds the given listener to the composite and all its children.
   */
  public static void recursiveAddListener(Composite composite, int eventType, Listener listener) {
    composite.addListener(eventType, listener);
    for (Control child : composite.getChildren()) {
      if (child instanceof Composite) {
        recursiveAddListener((Composite)child, eventType, listener);
      } else {
        child.addListener(eventType, listener);
      }
    }
  }

  /**
   * Recursively sets the foreground color for the composite and all its children.
   */
  public static void recursiveSetForeground(Composite composite, Color color) {
    composite.setForeground(color);
    for (Control child : composite.getChildren()) {
      if (child instanceof Composite) {
        recursiveSetForeground((Composite)child, color);
      } else {
        child.setForeground(color);
      }
    }
  }

  /**
   * Recursively sets the background color for the composite and all its children.
   */
  public static void recursiveSetBackground(Composite composite, Color color) {
    composite.setBackground(color);
    for (Control child : composite.getChildren()) {
      if (child instanceof Composite) {
        recursiveSetBackground((Composite)child, color);
      } else {
        child.setBackground(color);
      }
    }
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

  public static GridLayout withMarginOnly(GridLayout layout, int marginWidth, int marginHeight) {
    layout.marginWidth = marginWidth;
    layout.marginHeight = marginHeight;
    return layout;
  }

  public static GridLayout withMarginAndSpacing(GridLayout layout,
      int marginWidth, int marginHeight, int horizontalSpacing, int verticalSpacing) {
    layout.marginWidth = marginWidth;
    layout.marginHeight = marginHeight;
    layout.horizontalSpacing = horizontalSpacing;
    layout.verticalSpacing = verticalSpacing;
    return layout;
  }

  public static GridLayout withSpacing(GridLayout layout, int horizontalSpacing, int verticalSpacing) {
    layout.horizontalSpacing = horizontalSpacing;
    layout.verticalSpacing = verticalSpacing;
    return layout;
  }

  public static RowLayout withMargin(RowLayout layout, int marginWidth, int marginHeight) {
    layout.marginWidth = marginWidth;
    layout.marginHeight = marginHeight;
    return layout;
  }

  public static RowLayout withSpacing(RowLayout layout, int spacing) {
    layout.spacing = spacing;
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
