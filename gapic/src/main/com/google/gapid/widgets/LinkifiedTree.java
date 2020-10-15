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

import static com.google.gapid.models.Follower.nullPrefetcher;
import static com.google.gapid.util.Arrays.last;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withAsyncRefresh;

import com.google.gapid.models.Follower;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.util.Events;
import com.google.gapid.util.GeoUtils;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.views.Formatter.LinkableStyledString;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.CopySources.ColumnTextProvider;

import org.eclipse.jface.layout.TreeColumnLayout;
import org.eclipse.jface.viewers.ColumnWeightData;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.OwnerDrawLabelProvider;
import org.eclipse.jface.viewers.StyledString;
import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.viewers.TreeSelection;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StyleRange;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.events.SelectionEvent;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.TextLayout;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;
import org.eclipse.swt.widgets.Widget;

import java.util.function.Function;
import java.util.function.Predicate;

/**
 * A {@link TreeViewer} where each label can have rich formatting (using
 * {@link com.google.gapid.views.Formatter.StylingString}), links, and custom background color.
 */
public abstract class LinkifiedTree<T, F> extends Composite {
  private final TreeViewer viewer;
  protected final Widgets.Refresher refresher;
  protected final ContentProvider<T> contentProvider;
  protected final LabelProvider labelProvider;

  public LinkifiedTree(Composite parent, int treeStyle, Widgets widgets) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout());
    this.viewer = createTreeViewer(this, treeStyle);
    this.refresher = withAsyncRefresh(viewer);
    this.contentProvider = createContentProvider();
    this.labelProvider = createLabelProvider(widgets.theme);

    viewer.setContentProvider(contentProvider);
    viewer.setLabelProvider(labelProvider);

    addListener(SWT.Dispose, e -> {
      reset();
    });

    Tree tree = viewer.getTree();
    MouseAdapter mouseHandler = new MouseAdapter() {
      @Override
      public void mouseMove(MouseEvent e) {
        updateHover(Events.getLocation(e));
      }

      @Override
      public void mouseScrolled(MouseEvent e) {
        updateHover(Events.getLocation(e));
      }

      @Override
      public void widgetSelected(SelectionEvent e) {
        // Scrollbar was moved / mouse wheel caused scrolling. This is required for systems with
        // a touchpad with scrolling inertia, where the view keeps scrolling long after the mouse
        // wheel event has been processed.
        Display disp = getDisplay();
        updateHover(disp.map(null, tree, disp.getCursorLocation()));
      }

      @Override
      public void mouseExit(MouseEvent e) {
        labelProvider.hoverItem(null, null);
      }

      @Override
      public void mouseUp(MouseEvent e) {
        Point location = new Point(e.x, e.y);
        Path.Any follow = labelProvider.getFollow(tree.getItem(location), location);
        if (follow != null) {
          follow(follow);
        }
      }

      private void updateHover(Point location) {
        TreeItem item = tree.getItem(location);
        // When hovering over the far left of deep items, getItem returns null. Let's check a few
        // more places to the right.
        if (item == null) {
          for (int testX = location.x + 20; item == null && testX < 300; testX += 20) {
            item = tree.getItem(new Point(testX, location.y));
          }
        }

        if (labelProvider.hoverItem(item, location)) {
          Path.Any follow = labelProvider.getFollow(item, location);
          setCursor((follow == null) ? null : getDisplay().getSystemCursor(SWT.CURSOR_HAND));
        } else {
          setCursor(null);
        }
      }
    };
    tree.addMouseListener(mouseHandler);
    tree.addMouseTrackListener(mouseHandler);
    tree.addMouseMoveListener(mouseHandler);
    tree.addMouseWheelListener(mouseHandler);
    tree.getVerticalBar().addSelectionListener(mouseHandler);
  }

  protected LabelProvider createLabelProvider(Theme theme) {
    return new LabelProvider(theme);
  }

  // Should be called from the constructor.
  protected void setUpStateForColumnAdding() {
    if (viewer.getTree().getColumnCount() < 1) {
      viewer.getTree().setHeaderVisible(true);
      TreeViewerColumn column = createTreeColumn(viewer, "");
      column.setLabelProvider(labelProvider);
      TreeColumnLayout layout = new TreeColumnLayout();
      layout.setColumnData(column.getColumn(), new ColumnWeightData(100));
      setLayout(layout);
    }
  }

  protected TreeViewerColumn addColumn(String title, Function<T, String> labels, int width) {
    TreeViewerColumn column = createTreeColumn(viewer, title, labels);
    ((TreeColumnLayout)getLayout())
        .setColumnData(column.getColumn(), new ColumnWeightData(0, width));
    return column;
  }

  public void packColumn() {
    packColumns(viewer.getTree());
  }

  public void setInput(T root) {
    // Clear the selection, since we handle maintaining the selection ourselves and so
    // don't want JFace's selection preserving, as it appears to be broken on input
    // change (see https://github.com/google/gapid/issues/1264)
    setSelection(null);
    viewer.setInput(root);
    if (root != null && viewer.getTree().getItemCount() > 0) {
      viewer.getTree().setSelection(viewer.getTree().getItem(0));
      viewer.getTree().showSelection();
    }
  }

  public Control getControl() {
    return viewer.getControl();
  }

  public T getSelection() {
    if (viewer.getTree().getSelectionCount() >= 1) {
      return getElement(last(viewer.getTree().getSelection()));
    }
    return null;
  }

  public void setSelection(TreePath selection) {
    if (selection == null || (selection.getSegmentCount() == 0)) {
      viewer.setSelection(new TreeSelection(), true);
    } else {
      viewer.setSelection(new TreeSelection(selection), true);
    }
  }

  public Object[] getExpandedElements() {
    return viewer.getExpandedElements();
  }

  public boolean getExpandedState(TreePath path) {
    return viewer.getExpandedState(path);
  }

  public void setExpandedState(TreePath path, boolean state) {
    viewer.setExpandedState(path, state);
  }

  public Point getScrollPos() {
    TreeItem topItem = viewer.getTree().getTopItem();
    return (topItem == null) ? null : GeoUtils.center(topItem.getBounds());
  }

  public void scrollTo(Point pos) {
    TreeItem topItem = (pos == null) ? null : viewer.getTree().getItem(pos);
    if (topItem != null) {
      viewer.getTree().setTopItem(topItem);
    }
  }

  public void setPopupMenu(Menu popup, Predicate<T> shouldShow) {
    Tree tree = viewer.getTree();
    tree.setMenu(popup);

    Predicate<T> pred = o -> o != null && shouldShow.test(o);
    tree.addListener(SWT.MenuDetect, e -> {
      TreeItem item = tree.getItem(tree.toControl(e.x, e.y));
      e.doit = item != null && pred.test(getElement(item));
    });
  }

  @SuppressWarnings("unchecked")
  public void registerAsCopySource(CopyPaste cp, ColumnTextProvider<T> columns, boolean align) {
    CopySources.registerTreeAsCopySource(cp, viewer, (ColumnTextProvider<Object>)columns, align);
  }

  protected abstract ContentProvider<T> createContentProvider();
  protected abstract <S extends StylingString> S
      format(T node, S string, Follower.Prefetcher<F> follower);
  protected abstract Color getBackgroundColor(T node);
  protected abstract Follower.Prefetcher<F> prepareFollower(T node, Runnable callback);
  protected abstract void follow(Path.Any path);

  protected void reset() {
    labelProvider.reset();
  }

  @SuppressWarnings("unchecked")
  protected static <T> T cast(Object o) {
    return (T)o;
  }

  protected T getElement(Widget item) {
    return cast(item.getData());
  }

  /**
   * View data model for the tree.
   */
  protected abstract static class ContentProvider<T> implements ITreeContentProvider {
    @Override
    public Object[] getElements(Object inputElement) {
      return getChildren(inputElement);
    }

    @Override
    public Object[] getChildren(Object parent) {
      return !hasChildren(parent) ? null : getChildNodes(cast(parent));
    }

    @Override
    public boolean hasChildren(Object element) {
      return hasChildNodes(cast(element));
    }

    @Override
    public Object getParent(Object element) {
      return getParentNode(cast(element));
    }

    protected abstract boolean hasChildNodes(T element);
    protected abstract T[] getChildNodes(T parent);
    protected abstract T getParentNode(T child);
    protected abstract boolean isLoaded(T element);
    protected abstract void load(T node, Runnable callback);
  }

  /**
   * Renders the labels in the tree.
   */
  protected class LabelProvider extends OwnerDrawLabelProvider
      implements VisibilityTrackingTreeViewer.Listener {

    private final Theme theme;
    private final TextLayout layout;
    private TreeItem lastHovered;
    private Follower.Prefetcher<F> lastPrefetcher = nullPrefetcher();

    public LabelProvider(Theme theme) {
      this.theme = theme;
      this.layout = new TextLayout(getDisplay());
    }

    @Override
    public void onShow(TreeItem item) {
      T element = getElement(item);
      contentProvider.load(element, () -> {
        if (!item.isDisposed()) {
          update(item);
          refresher.refresh();
        }
      });
    }

    @Override
    protected void erase(Event event, Object element) {
      Label label = getLabel(event);

      if (!shouldIgnoreColors(event) && label.background != null) {
        Color oldBackground = event.gc.getBackground();
        event.gc.setBackground(label.background);
        event.gc.fillRectangle(event.x, event.y, event.width, event.height);
        event.gc.setBackground(oldBackground);
      }
      // Clear the foreground bit, as we'll do our own foreground rendering.
      event.detail &= ~SWT.FOREGROUND;
    }

    @Override
    protected void measure(Event event, Object element) {
      Label label = getLabel(event);
      if (label.bounds == null) {
        updateLayout(label.string, false);
        label.bounds = layout.getBounds();
      }

      event.width = label.bounds.width;
      event.height =label.bounds.height;
    }

    @Override
    protected void paint(Event event, Object element) {
      GC gc = event.gc;
      Label label = getLabel(event);

      boolean ignoreColors = shouldIgnoreColors(event);
      Color oldForeground = event.gc.getForeground();
      Color oldBackground = event.gc.getBackground();
      if (!ignoreColors && label.background != null) {
        event.gc.setBackground(label.background);
      }

      drawText(event, label, ignoreColors);
      if (shouldDrawFocus(event)) {
        drawFocus(event);
      }

      if (!ignoreColors) {
        gc.setForeground(oldForeground);
        gc.setBackground(oldBackground);
      }
    }

    private void drawText(Event event, Label label, boolean ignoreColors) {
      Rectangle bounds = ((TreeItem)event.item).getTextBounds(0);
      if (bounds == null) {
        return;
      }
      drawText(getElement(event.item), event.gc, bounds, label, ignoreColors);
    }

    protected void drawText(@SuppressWarnings("unused") T node, GC gc, Rectangle bounds,
        Label label, boolean ignoreColors) {
      updateLayout(label.string, ignoreColors);
      layout.draw(gc, bounds.x, bounds.y + (bounds.height - label.bounds.height) / 2);
    }

    private void drawFocus(Event event) {
      Rectangle focusBounds = ((TreeItem)event.item).getBounds();
      event.gc.drawFocus(focusBounds.x, focusBounds.y, focusBounds.width, focusBounds.height);
    }

    private void updateLayout(StyledString string, boolean ignoreColors) {
      layout.setText(string.getString());
      for (StyleRange range : string.getStyleRanges()) {
        if (ignoreColors && (range.foreground != null || range.background != null)) {
          range = (StyleRange)range.clone();
          range.foreground = null;
          range.background = null;
        }
        layout.setStyle(range, range.start, range.start + range.length - 1);
      }
    }

    private boolean shouldDrawFocus(Event event) {
      return (event.detail & SWT.FOCUSED) != 0;
    }

    private boolean shouldIgnoreColors(Event event) {
      return (event.detail & SWT.SELECTED) != 0;
    }

    private void update(TreeItem item) {
      Label label = getLabelNoUpdate(item);
      T element = getElement(item);

      label.background = getBackgroundColor(element);
      updateText(item, label, element);
      label.bounds = null;
      label.loaded = contentProvider.isLoaded(element);
      item.setText(label.string.getString());
    }

    private void updateText(TreeItem item, Label label, T element) {
      label.string = format(element, LinkableStyledString.ignoring(theme),
          (item == lastHovered) ? lastPrefetcher : nullPrefetcher()).getString();
    }

    public boolean hoverItem(TreeItem item, @SuppressWarnings("unused") Point location) {
      if (item != lastHovered) {
        TreeItem tmp = lastHovered;
        lastHovered = item;
        lastPrefetcher.cancel();

        if (tmp != null && !tmp.isDisposed()) {
          updateText(tmp, getLabelNoUpdate(tmp), getElement(tmp));
        }

        if (item == null) {
          lastPrefetcher = nullPrefetcher();
        } else {
          lastPrefetcher = prepareFollower(getElement(item), () -> {
            Widgets.scheduleIfNotDisposed(item, () -> {
              updateText(item, getLabelNoUpdate(item), getElement(item));
              refresher.refresh();
            });
          });
        }
        refresher.refresh();
      }
      return item != null;
    }

    public Path.Any getFollow(TreeItem item, Point location) {
      if (item == null || item != lastHovered) {
        return null;
      }
      Rectangle bounds = item.getTextBounds(0);
      if (!bounds.contains(location)) {
        return null;
      }

      LinkableStyledString string =
          format(getElement(item), LinkableStyledString.create(theme), lastPrefetcher);
      string.endLink();
      string.append("placeholder", string.defaultStyle());
      updateLayout(string.getString(), false);

      Rectangle textBounds = layout.getBounds();
      textBounds.x = bounds.x;
      textBounds.y = bounds.y + (bounds.height - textBounds.height) / 2;
      if (!textBounds.contains(location)) {
        return null;
      }

      int offset = layout.getOffset(location.x - textBounds.x, location.y - textBounds.y, null);
      return (Path.Any)string.getLinkTarget(offset);
    }

    private Label getLabel(Event event) {
      return getLabel(event.item);
    }

    private Label getLabel(Widget item) {
      Label result = (Label)item.getData(Label.KEY);
      if (result == null) {
        item.setData(Label.KEY, result = new Label(theme));
        update((TreeItem)item);
      } else if (contentProvider.isLoaded(getElement(item)) != result.loaded) {
        update((TreeItem)item);
      }
      return result;
    }

    private Label getLabelNoUpdate(Widget item) {
      Label result = (Label)item.getData(Label.KEY);
      if (result == null) {
        item.setData(Label.KEY, result = new Label(theme));
      }
      return result;
    }

    public void reset() {
      layout.dispose();
    }
  }

  /**
   * POJO containing cached rendering information for a label.
   */
  protected static class Label {
    public static String KEY = Label.class.getName();

    public Color background;
    public StyledString string;
    public Rectangle bounds;
    public boolean loaded;

    public Label(Theme theme) {
      this.background = null;
      this.string = new StyledString("Loading...", theme.structureStyler());
      this.bounds = null;
      this.loaded = false;
    }
  }
}
