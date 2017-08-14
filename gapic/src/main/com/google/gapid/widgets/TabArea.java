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

import static com.google.gapid.widgets.TabDnD.withMovableTabs;
import static com.google.gapid.widgets.Widgets.createTabFolder;
import static com.google.gapid.widgets.Widgets.createTabItem;

import com.google.common.base.Objects;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.CTabFolder;
import org.eclipse.swt.custom.CTabFolder2Adapter;
import org.eclipse.swt.custom.CTabFolderEvent;
import org.eclipse.swt.custom.CTabItem;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.custom.StackLayout;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

import java.util.function.Function;

/**
 * Manages {@link CTabItem tabs} in three {@link CTabFolder tab areas}. Allows the user to drag
 * tabs between areas and minimzing/maximzing each area. The three areas are horizontally laid out
 * with movable dividers.
 */
public class TabArea {
  private static final int MINIMIZED_SIZE = 40;

  private final SashForm sash;
  private final Folder left, center, right;
  private int[] weights;

  public TabArea(Composite parent, Persistance persistance) {
    this.sash = new SashForm(parent, SWT.HORIZONTAL);

    FolderInfo[] folders = persistance.restore();
    this.left = new Folder(sash, folders[0], true, this::updateWeights);
    this.center = new Folder(sash, folders[1], false, this::updateWeights);
    this.right = new Folder(sash, folders[2], true, this::updateWeights);
    this.weights = new int[] { folders[0].weight, folders[1].weight, folders[2].weight };

    left.folder.setSelection(0);
    center.folder.setSelection(0);
    center.folder.setMaximizeVisible(true);
    right.folder.setSelection(0);
    sash.setWeights(weights);
    updateWeights();

    TabDnD.addListener(new TabDnD.Listener() {
      @Override
      public void itemCopied(CTabItem source, CTabItem target) {
        target.setData(TabInfo.KEY, source.getData(TabInfo.KEY));
      }

      @Override
      public void onTabMoved(
          CTabFolder sourceFolder, CTabItem oldItem, CTabFolder destFolder, CTabItem newItem) {
        if (sourceFolder.getItemCount() == 0 || destFolder.getItemCount() == 1) {
          updateWeights();
        }
      }
    });
    center.folder.addCTabFolder2Listener(new CTabFolder2Adapter() {
      @Override
      public void maximize(CTabFolderEvent event) {
        TabArea.this.maximize();
        updateWeights();
      }

      @Override
      public void restore(CTabFolderEvent event) {
        TabArea.this.restore();
        updateWeights();
      }
    });
    sash.addListener(SWT.Resize, e -> updateWeights());
    sash.addListener(SWT.Dispose, e -> {
      rememberWeights();
      persistance.store(new FolderInfo[] {
          left.getInfo(weights[0]),
          center.getInfo(weights[1]),
          right.getInfo(weights[2]),
      });
    });
  }

  public boolean showTab(Object id) {
    return findAndSelect(left, id) || findAndSelect(center, id) || findAndSelect(right, id);
  }

  public boolean removeTab(Object id) {
    return findAndDispose(left, id) || findAndDispose(center, id) || findAndDispose(right, id);
  }

  public void addNewTabToCenter(TabInfo info) {
    center.addTab(info);
    center.updateState();
  }

  public void setLeftVisible(boolean visible) {
    if (!visible) {
      moveAllTabs(left.folder, center.folder);
    }
    left.setVisible(visible);
    left.requestLayout();
  }

  public void setRightVisible(boolean visible) {
    if (!visible) {
      moveAllTabs(right.folder, center.folder);
    }
    right.setVisible(visible);
    right.requestLayout();
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

    if (!folder.shown()) {
      folder.restore();
      updateWeights();
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
    updateWeights();
    return true;
  }

  protected void maximize() {
    left.minimize();
    right.minimize();
  }

  protected void restore() {
    left.restore();
    right.restore();
  }

  protected void updateWeights() {
    rememberWeights();
    updateCenter();

    if (left.shown() && right.shown()) {
      sash.setWeights(weights);
    } else if (!left.shown() && !right.shown()) {
      Rectangle bounds = sash.getBounds();
      sash.setWeights(new int[] {
          MINIMIZED_SIZE,
          Math.max(MINIMIZED_SIZE, bounds.width - 2 * MINIMIZED_SIZE),
          MINIMIZED_SIZE
      });
    } else if (!left.shown()) {
      Rectangle bounds = sash.getBounds();
      int sum = weights[1] + weights[2], width = bounds.width - MINIMIZED_SIZE;
      sash.setWeights(new int[] {
          MINIMIZED_SIZE,
          Math.max(MINIMIZED_SIZE, width * weights[1] / sum),
          Math.max(MINIMIZED_SIZE, width * weights[2] / sum)
      });
    } else {
      Rectangle bounds = sash.getBounds();
      int sum = weights[0] + weights[1], width = bounds.width - MINIMIZED_SIZE;
      sash.setWeights(new int[] {
          Math.max(MINIMIZED_SIZE, width * weights[0] / sum),
          Math.max(MINIMIZED_SIZE, width * weights[1] / sum),
          MINIMIZED_SIZE
      });
    }
  }

  private void rememberWeights() {
    boolean leftWasShown = left.updateState();
    boolean rightWasShown = right.updateState();
    if (leftWasShown && rightWasShown) {
      weights = sash.getWeights();
    } else if (leftWasShown && !left.shown()) {
      int[] curWeights = sash.getWeights();
      weights[0] = (weights[0] + weights[1]) * curWeights[0] / (curWeights[0] + curWeights[1]);
      weights[1] = (weights[0] + weights[1]) * curWeights[1] / (curWeights[0] + curWeights[1]);
    } else if (rightWasShown && !right.shown()) {
      int[] curWeights = sash.getWeights();
      weights[1] = (weights[1] + weights[2]) * curWeights[1] / (curWeights[1] + curWeights[2]);
      weights[2] = (weights[1] + weights[2]) * curWeights[2] / (curWeights[1] + curWeights[2]);
    } else {
      weights = sash.getWeights();
    }
  }

  private void updateCenter() {
    if (left.shown() || right.shown()) {
      center.folder.setMaximized(false);
      center.folder.setMaximizeVisible(true);
    } else if (left.empty() && right.empty()) {
      center.folder.setMaximized(false);
      center.folder.setMaximizeVisible(false);
    } else {
      center.folder.setMaximized(true);
      center.folder.setMaximizeVisible(true);
    }
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
    public final int weight;

    public FolderInfo(boolean minimized, TabInfo[] tabs, int weight) {
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
    private final boolean minimizable;
    public final CTabFolder folder;
    private final CTabFolder minimized;
    private TabState state = TabState.VISIBLE;

    public Folder(Composite parent, FolderInfo info, boolean minimizable, Runnable updateWeights) {
      super(parent, SWT.NONE);
      this.minimizable = minimizable;
      setLayout(new StackLayout());

      folder = withMovableTabs(createTabFolder(this));
      minimized = createTabFolder(this);
      getLayout().topControl = folder;

      folder.setMinimizeVisible(minimizable);
      minimized.setMinimizeVisible(true);
      minimized.setMinimized(true);

      folder.addCTabFolder2Listener(new CTabFolder2Adapter() {
        @Override
        public void minimize(CTabFolderEvent event) {
          Folder.this.minimize();
          updateWeights.run();
        }
      });
      minimized.addCTabFolder2Listener(new CTabFolder2Adapter() {
        @Override
        public void restore(CTabFolderEvent event) {
          Folder.this.restore();
          updateWeights.run();
        }
      });

      for (TabInfo tab : info.tabs) {
        addTab(tab);
      }
      if (minimizable && info.minimized) {
        minimize();
      }
      updateState();
    }

    public void addTab(TabInfo tab) {
      CTabItem item = createTabItem(folder, tab.label, tab.contentFactory.apply(folder));
      item.setData(TabInfo.KEY, tab);
    }

    public void minimize() {
      if (!empty()) {
        folder.setMinimized(true);
        getLayout().topControl = minimized;
      }
    }

    public void restore() {
      if (!empty()) {
        folder.setMinimized(false);
        getLayout().topControl = folder;
      }
    }

    @Override
    public StackLayout getLayout() {
      return (StackLayout)super.getLayout();
    }

    public boolean updateState() {
      boolean result = state.shown();
      if (empty()) {
        state = TabState.EMPTY;
        folder.setMinimizeVisible(false);
      } else if (folder.getMinimized()) {
        state = TabState.MINIMIZED;
      } else {
        state = TabState.VISIBLE;
        folder.setMinimizeVisible(minimizable);
      }
      return result;
    }

    public boolean shown() {
      return state.shown();
    }

    public boolean empty() {
      return folder.getItemCount() == 0;
    }

    public FolderInfo getInfo(int weight) {
      TabInfo[] tabs = new TabInfo[folder.getItemCount()];
      for (int i = 0; i < tabs.length; i++) {
        tabs[i] = (TabInfo)folder.getItem(i).getData(TabInfo.KEY);
      }
      return new FolderInfo(state == TabState.MINIMIZED, tabs, weight);
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

  /**
   * The state of a folder's tabs.
   */
  private static enum TabState {
    VISIBLE, MINIMIZED, EMPTY;

    public boolean shown() {
      return this == VISIBLE;
    }
  }
}
