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
import static com.google.gapid.util.GeoUtils.withW;
import static com.google.gapid.util.GeoUtils.withX;
import static com.google.gapid.util.GeoUtils.withXH;
import static com.google.gapid.util.GeoUtils.withXW;
import static com.google.gapid.widgets.TabDnD.withMovableTabs;
import static com.google.gapid.widgets.Widgets.createTabFolder;
import static com.google.gapid.widgets.Widgets.createTabItem;

import com.google.common.base.Objects;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.CTabFolder;
import org.eclipse.swt.custom.CTabFolder2Adapter;
import org.eclipse.swt.custom.CTabFolderEvent;
import org.eclipse.swt.custom.CTabItem;
import org.eclipse.swt.custom.StackLayout;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Layout;
import org.eclipse.swt.widgets.Sash;

import java.util.function.Function;

/**
 * Manages {@link CTabItem tabs} in three {@link CTabFolder tab areas}. Allows the user to drag
 * tabs between areas and minimzing/maximzing each area. The three areas are horizontally laid out
 * with movable dividers.
 */
public class TabArea extends Composite {
  private static final int MIN_SIZE = 40;
  private static final int LEFT = 0, CENTER = 1, RIGHT = 2;

  private final Folder left, center, right;
  private final Sash leftSash, rightSash;
  private final double[] weights;
  private boolean shouldRestoreLeft = true, shouldRestoreRight = true;

  public TabArea(Composite parent, Persistance persistance) {
    super(parent, SWT.NONE);
    setLayout(new Layout() {
      @Override
      protected Point computeSize(Composite composite, int wHint, int hHint, boolean flushCache) {
        return layoutComputeSize(wHint, hHint, flushCache);
      }

      @Override
      protected void layout(Composite composite, boolean flushCache) {
        doLayout(flushCache);
      }
    });

    FolderInfo[] folders = persistance.restore();
    left = new Folder(this, folders[0], false);
    leftSash = new Sash(this, SWT.VERTICAL);
    center = new Folder(this, folders[1], true);
    rightSash = new Sash(this, SWT.VERTICAL);
    right = new Folder(this, folders[2], false);
    weights = new double[] { folders[0].weight, folders[1].weight, folders[2].weight };
    normalizeWeights();

    leftSash.addListener(SWT.Selection, this::moveLeftSash);
    rightSash.addListener(SWT.Selection, this::moveRightSash);
    left.addHandlers(this::minimizeLeft, () -> { /* noop */ }, this::restoreLeft);
    center.addHandlers(() -> { /* noop */ }, this::maximizeCenter, this::restoreCenter);
    right.addHandlers(this::minimizeRight, () -> { /* noop */ }, this::restoreRight);
    TabDnD.addListener(new TabDnD.Listener() {
      @Override
      public void itemCopied(CTabItem source, CTabItem target) {
        target.setData(TabInfo.KEY, source.getData(TabInfo.KEY));
      }

      @Override
      public void onTabMoved(
          CTabFolder sourceFolder, CTabItem oldItem, CTabFolder destFolder, CTabItem newItem) {
        if (sourceFolder.getItemCount() == 0 || destFolder.getItemCount() == 1) {
          updateEmptyFolders();
        }
      }
    });
    addListener(SWT.Dispose, e -> {
      normalizeWeights();
      persistance.store(new FolderInfo[] {
          left.getInfo(weights[0]),
          center.getInfo(weights[1]),
          right.getInfo(weights[2]),
      });
    });
  }

  private void normalizeWeights() {
    double sum = weights[0] + weights[1] + weights[2];
    if (sum != 0) {
      weights[0] /= sum;
      weights[1] /= sum;
      weights[2] /= sum;
    }
  }

  protected void updateEmptyFolders() {
    left.folder.setTabHeight(center.folder.getTabHeight());
    if (left.isEmpty()) {
      left.minimize();
      if (right.isMinimized()) {
        shouldRestoreRight = true;
        center.maximzie();
      }
    } else {
      doLayout(false);
      left.folder.setMinimizeVisible(true);
      left.folder.requestLayout();
    }

    right.folder.setTabHeight(center.folder.getTabHeight());
    if (right.isEmpty()) {
      right.minimize();
      if (left.isMinimized()) {
        shouldRestoreLeft = true;
        center.maximzie();
      }
    } else {
      doLayout(false);
      right.folder.setMinimizeVisible(true);
      right.folder.requestLayout();
    }
  }

  private void moveLeftSash(Event e) {
    if (left.isEmpty()) {
      e.doit = false;
      return;
    } else if (left.isMinimized()) {
      left.restore();
      center.restore();
    }
    int width = getClientArea().width;
    Rectangle leftSashSize = leftSash.getBounds();
    Rectangle rightSashSize = rightSash.getBounds();
    Rectangle centerSize = center.getBounds();
    e.x = Math.max(MIN_SIZE, Math.min(rightSashSize.x - leftSashSize.width - MIN_SIZE, e.x));
    int delta = leftSashSize.x - e.x;
    if (delta != 0) {
      left.setBounds(withW(left.getBounds(), e.x));
      leftSash.setBounds(withX(leftSashSize, e.x));
      center.setBounds(withXW(centerSize, e.x + leftSashSize.width, centerSize.width + delta));
    }
    width -= leftSashSize.width + rightSashSize.width;
    weights[LEFT] = (double)e.x / width;
    weights[CENTER] = (double)centerSize.width / width;
    if (right.isMinimized()) {
      weights[LEFT] *= (1 - weights[RIGHT]);
      weights[CENTER] *= (1 - weights[RIGHT]);
    } else {
      weights[RIGHT] = 1 - weights[LEFT] - weights[CENTER];
    }
  }

  private void moveRightSash(Event e) {
    if (right.isEmpty()) {
      e.doit = false;
      return;
    } else if (right.isMinimized()) {
      right.restore();
      center.restore();
    }
    int width = getClientArea().width;
    Rectangle leftSashSize = leftSash.getBounds();
    Rectangle rightSashSize = rightSash.getBounds();
    Rectangle rightSize = right.getBounds();
    Rectangle centerSize = center.getBounds();
    e.x = Math.max(leftSashSize.x + leftSashSize.width + MIN_SIZE, Math.min(width - rightSashSize.width - MIN_SIZE, e.x));
    int delta = rightSashSize.x - e.x;
    if (delta != 0) {
      center.setBounds(withXW(centerSize, centerSize.x, centerSize.width - delta));
      rightSash.setBounds(withX(rightSashSize, e.x));
      right.setBounds(withXW(rightSize, e.x + rightSashSize.width, rightSize.width + delta));
    }
    width -= leftSashSize.width + rightSashSize.width;
    weights[CENTER] = (double)centerSize.width / width;
    weights[RIGHT] = (double)rightSize.width / width;
    if (left.isMinimized()) {
      weights[CENTER] *= (1 - weights[LEFT]);
      weights[RIGHT] *= (1 - weights[LEFT]);
    } else {
      weights[LEFT] = 1 - weights[CENTER] - weights[RIGHT];
    }
  }

  private void minimizeLeft() {
    if (right.isMinimized()) {
      center.maximzie();
      shouldRestoreLeft = true;
    } else {
      shouldRestoreLeft = false;
    }
    doLayout(false);
  }

  private void restoreLeft() {
    center.restore();
    doLayout(false);
  }

  private void maximizeCenter() {
    if (!left.isMinimized()) {
      left.minimize();
      shouldRestoreLeft = true;
    } else {
      shouldRestoreLeft = false;
    }
    if (!right.isMinimized()) {
      right.minimize();
      shouldRestoreRight = true;
    } else {
      shouldRestoreRight = false;
    }
    doLayout(false);
  }

  private void restoreCenter() {
    if (shouldRestoreLeft) {
      left.restore();
    }
    if (shouldRestoreRight) {
      right.restore();
    }
    doLayout(false);
  }

  private void minimizeRight() {
    if (left.isMinimized()) {
      center.maximzie();
      shouldRestoreRight = true;
    } else {
      shouldRestoreRight = false;
    }
    doLayout(false);
  }

  private void restoreRight() {
    center.restore();
    doLayout(false);
  }

  public boolean showTab(Object id) {
    return findAndSelect(left, id) || findAndSelect(center, id) || findAndSelect(right, id);
  }

  public boolean removeTab(Object id) {
    return findAndDispose(left, id) || findAndDispose(center, id) || findAndDispose(right, id);
  }

  public void addNewTabToCenter(TabInfo info) {
    center.addTab(info);
  }

  public void setLeftVisible(boolean visible) {
    if (!visible) {
      moveAllTabs(left.folder, center.folder);
    }
    left.setVisible(visible);
    leftSash.setVisible(visible);
    doLayout(false);
  }

  public void setRightVisible(boolean visible) {
    if (!visible) {
      moveAllTabs(right.folder, center.folder);
    }
    right.setVisible(visible);
    rightSash.setVisible(visible);
    doLayout(false);
  }

  private static void moveAllTabs(CTabFolder from, CTabFolder to) {
    CTabItem[] items = from.getItems();
    for (CTabItem item : items) {
      TabDnD.moveTab(item, to, -1);
    }
  }

  private boolean findAndSelect(Folder folder, Object id) {
    CTabItem item = folder.findItem(id);
    if (item == null) {
      return false;
    }

    if (folder.isMinimized()) {
      folder.restore();
      center.restore();
      doLayout(false);
    }
    folder.folder.setSelection(item);
    return true;
  }

  private boolean findAndDispose(Folder folder, Object id) {
    CTabItem item = folder.findItem(id);
    if (item == null) {
      return false;
    }

    // The CTabItem's control is only disposed once the parent folder is disposed.
    // So let's dispose of it early ourselves.
    Control control = item.getControl();
    if (control != null) {
      control.dispose();
    }

    item.dispose();
    updateEmptyFolders();
    return true;
  }

  protected Point layoutComputeSize(int width, int height, boolean flushCache) {
    if (width == SWT.DEFAULT || height == SWT.DEFAULT) {
      Point leftSize = left.computeSize(width, height, flushCache);
      Point leftSashSize = leftSash.computeSize(SWT.DEFAULT, height, flushCache);
      Point centerSize = center.computeSize(width, height, flushCache);
      Point rightSashSize = rightSash.computeSize(SWT.DEFAULT, height, flushCache);
      Point rightSize = right.computeSize(width, height, flushCache);

      if (width == SWT.DEFAULT) {
        width = Math.max(MIN_SIZE, centerSize.x);
        if (left.isVisible()) {
          width += Math.max(MIN_SIZE, leftSize.x) + leftSashSize.x;
        }
        if (right.isVisible()) {
          width += rightSashSize.x + Math.max(MIN_SIZE, rightSize.x);
        }
      }
      if (height == SWT.DEFAULT) {
        height = Math.max(leftSize.y, Math.max(centerSize.y, Math.max(rightSize.y,
            Math.max(leftSashSize.y, rightSashSize.y))));
      }
    }
    return new Point(width, height);
  }

  protected void doLayout(boolean flushCache) {
    Rectangle size = getClientArea();

    if (!left.isVisible() && !right.isVisible()) {
      center.setBounds(size);
      return;
    }

    Point leftSashSize = leftSash.computeSize(SWT.DEFAULT, size.height, flushCache);
    Point rightSashSize = rightSash.computeSize(SWT.DEFAULT, size.height, flushCache);

    Rectangle leftSize = left.getBounds();
    Rectangle centerSize = center.getBounds();
    Rectangle rightSize = right.getBounds();

    if (!left.isVisible()) {
      int width = size.width - rightSashSize.x;
      leftSize.width = leftSashSize.x = 0;
      if (right.isMinimized()) {
        rightSize.width = MIN_SIZE;
        centerSize.width = width - MIN_SIZE;
      } else {
        rightSize.width = (int)(weights[RIGHT] * width / (weights[CENTER] + weights[RIGHT]));
        centerSize.width = width - rightSize.width;
      }
    } else if (!right.isVisible()) {
      int width = size.width - leftSashSize.x;
      rightSize.width = rightSashSize.x = 0;
      if (left.isMinimized()) {
        leftSize.width = MIN_SIZE;
        centerSize.width = width - MIN_SIZE;
      } else {
        leftSize.width = (int)(weights[LEFT] * width / (weights[LEFT] + weights[CENTER]));
        centerSize.width = width - leftSize.width;
      }
    } else {
      int width = size.width - leftSashSize.x - rightSashSize.x;
      if (left.isMinimized() && right.isMinimized()) {
        leftSize.width = rightSize.width = MIN_SIZE;
        centerSize.width = width - 2 * MIN_SIZE;
      } else if (left.isMinimized()) {
        leftSize.width = MIN_SIZE;
        width -= MIN_SIZE;
        rightSize.width = (int)(weights[RIGHT] * width / (weights[CENTER] + weights[RIGHT]));
        centerSize.width = width - rightSize.width;
      } else if (right.isMinimized()) {
        rightSize.width = MIN_SIZE;
        width -= MIN_SIZE;
        leftSize.width = (int)(weights[LEFT] * width / (weights[LEFT] + weights[CENTER]));
        centerSize.width = width - leftSize.width;
      } else {
        double sum = weights[LEFT] + weights[CENTER] + weights[RIGHT];
        leftSize.width = (int)(weights[LEFT] * width / sum);
        rightSize.width = (int)(weights[RIGHT] * width / sum);
        centerSize.width = width - leftSize.width - rightSize.width;
      }
    }

    left.setBounds(withXH(leftSize, 0, size.height));
    leftSash.setBounds(new Rectangle(right(leftSize), 0, leftSashSize.x, size.height));
    center.setBounds(withXH(centerSize, right(leftSize) + leftSashSize.x, size.height));
    rightSash.setBounds(new Rectangle(right(centerSize), 0, leftSashSize.x, size.height));
    right.setBounds(withXH(rightSize, right(centerSize) + leftSashSize.x, size.height));
  }

  /**
   * Responsible for remembering the order and location of tabs.
   */
  public static interface Persistance {
    /**
     * @return the previously stored tab information. Has to be length 3 (left, center, right).
     */
    public FolderInfo[] restore();

    /**
     * @param folders the tab information to store. Always length 3 (left, center, right).
     */
    public void store(FolderInfo[] folders);
  }

  /**
   * Size and containing tabs information of a folder.
   */
  public static class FolderInfo {
    public final boolean minimized;
    public final TabInfo[] tabs;
    public final double weight;

    public FolderInfo(boolean minimized, TabInfo[] tabs, double weight) {
      this.minimized = minimized;
      this.tabs = tabs;
      this.weight = weight;
    }
  }

  /**
   * Information about a single tab in a folder.
   */
  public static class TabInfo {
    public static final String KEY = TabInfo.class.getName();

    public final Object id;
    public final String label;
    public final Function<Composite, Control> contentFactory;

    public TabInfo(Object id, String label, Function<Composite, Control> contentFactory) {
      this.id = id;
      this.label = label;
      this.contentFactory = contentFactory;
    }
  }

  /**
   * A folder widget containing multiple tabs.
   */
  private static class Folder extends Composite {
    public final CTabFolder folder;
    private final CTabFolder minimized;

    public Folder(Composite parent, FolderInfo info, boolean maximizable) {
      super(parent, SWT.NONE);
      setLayout(new StackLayout());

      folder = withMovableTabs(createTabFolder(this));
      minimized = createTabFolder(this);
      getLayout().topControl = folder;

      folder.setMinimizeVisible(!maximizable);
      folder.setMaximizeVisible(maximizable);
      minimized.setMinimizeVisible(true);
      minimized.setMinimized(true);

      for (TabInfo tab : info.tabs) {
        addTab(tab);
      }
      folder.setSelection(0);
      if (!maximizable && info.minimized) {
        minimize();
      }
    }

    public void addTab(TabInfo tab) {
      CTabItem item = createTabItem(folder, tab.label, tab.contentFactory.apply(folder));
      item.setData(TabInfo.KEY, tab);
    }

    public void addHandlers(Runnable onMinimize, Runnable onMaximize, Runnable onRestore) {
      folder.addCTabFolder2Listener(new CTabFolder2Adapter() {
        @Override
        public void minimize(CTabFolderEvent event) {
          Folder.this.minimize();
          onMinimize.run();
        }

        @Override
        public void maximize(CTabFolderEvent event) {
          Folder.this.maximzie();
          onMaximize.run();
        }

        @Override
        public void restore(CTabFolderEvent event) {
          Folder.this.restore();
          onRestore.run();
        }
      });
      minimized.addCTabFolder2Listener(new CTabFolder2Adapter() {
        @Override
        public void restore(CTabFolderEvent event) {
          Folder.this.restore();
          onRestore.run();
        }
      });
    }

    public void minimize() {
      if (isEmpty()) {
        folder.setMinimizeVisible(false);
        getLayout().topControl = folder;
      } else {
        getLayout().topControl = minimized;
      }
      requestLayout();
    }

    public void maximzie() {
      folder.setMaximized(true);
      folder.requestLayout();
    }

    public void restore() {
      folder.setMaximized(false);
      getLayout().topControl = folder;
      requestLayout();
    }

    public boolean isMinimized() {
      return (getLayout().topControl == minimized) || isEmpty();
    }

    public boolean isEmpty() {
      return folder.getItemCount() == 0;
    }

    @Override
    public StackLayout getLayout() {
      return (StackLayout)super.getLayout();
    }

    public FolderInfo getInfo(double weight) {
      TabInfo[] tabs = new TabInfo[folder.getItemCount()];
      for (int i = 0; i < tabs.length; i++) {
        tabs[i] = (TabInfo)folder.getItem(i).getData(TabInfo.KEY);
      }
      return new FolderInfo(isMinimized(), tabs, weight);
    }

    public CTabItem findItem(Object id) {
      for (CTabItem item : folder.getItems()) {
        if (Objects.equal(id, ((TabInfo)item.getData(TabInfo.KEY)).id)) {
          return item;
        }
      }
      return null;
    }
  }
}
