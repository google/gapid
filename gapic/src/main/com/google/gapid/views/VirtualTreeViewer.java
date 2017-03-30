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

import org.eclipse.jface.viewers.IBaseLabelProvider;
import org.eclipse.jface.viewers.IContentProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

/**
 * A {@link TreeViewer} that notifies the bound {@link IContentProvider} and
 * {@link IBaseLabelProvider} of time visibility changes if they also implement
 * the {@link VisibilityListener} interface.
 */
public class VirtualTreeViewer extends TreeViewer {
  /**
   * The interface the {@link IContentProvider} and {@link IBaseLabelProvider} can
   * implement to be notified about {@link TreeItem}s being made visible and hidden
   * usually by scrolling.
   */
  public interface VisibilityListener {
    /**
     * Called when the {@link TreeItem} is made visible.
     */
    void OnShow(TreeItem item);
    /**
     * Called when the {@link TreeItem} is made invisible.
     */
    void OnHide(TreeItem item);
  }

  private Set<TreeItem> visible = new HashSet<>();

  public VirtualTreeViewer(Tree tree) {
    super(tree);
    tree.addPaintListener((e) -> updateVisibility());
  }

  private void updateVisibility() {
    List<VisibilityListener> listeners = getListeners();
    if (listeners.size() == 0) {
      return;
    }

    Set<TreeItem> seen = calcVisibleItems();
    if (seen == null) {
      return; // No reliable data.
    }
    for (TreeItem item : visible) {
      if (!seen.contains(item)) {
        for (VisibilityListener listener : listeners) {
          listener.OnHide(item);
        }
      }
    }
    for (TreeItem item : seen) {
      if (!visible.contains(item)) {
        for (VisibilityListener listener : listeners) {
          listener.OnShow(item);
        }
      }
    }
    visible = seen;
  }

  private List<VisibilityListener> getListeners() {
    List<VisibilityListener> listeners = new ArrayList<>();
    IContentProvider contentProvider = getContentProvider();
    if (contentProvider instanceof VisibilityListener) {
      listeners.add((VisibilityListener)contentProvider);
    }
    IBaseLabelProvider labelProvider = getLabelProvider();
    if (labelProvider instanceof VisibilityListener) {
      listeners.add((VisibilityListener)labelProvider);
    }
    return listeners;
  }

  private TreeItem nextItem(TreeItem item) {
    if (item.getExpanded()) {
      // Descend into children.
      int children = TreeHelper.itemCount(item);
      if (children > 0) {
        return TreeHelper.getItem(item, 0);
      }
    }
    // Ascend ancestors looking for siblings.
    Object parent = TreeHelper.getParent(item);
    while (parent != null) {
      int index = TreeHelper.indexOf(parent, item);
      int siblings = TreeHelper.itemCount(parent);
      if (index+1 < siblings) {
        return TreeHelper.getItem(parent,index + 1);
      }
      if (!(parent instanceof TreeItem)) {
        break;
      }
      item = (TreeItem)parent;
      parent = TreeHelper.getParent(parent);
    }
    return null;
  }

  private Set<TreeItem> calcVisibleItems() {
    Set<TreeItem> visible = new HashSet<>();
    Tree tree = VirtualTreeViewer.this.getTree();
    Rectangle rect = tree.getClientArea();
    if (tree.getTopItem() == null && tree.getItemCount() != 0) {
      // Work around bug where getTopItem() returns null when scrolling
      // up past the top item (elastic scroll).
      return null;
    }
    for (TreeItem item = tree.getTopItem(); item != null; item = nextItem(item)) {
      visible.add(item);
      Rectangle itemRect = item.getBounds();
      if (itemRect.y + itemRect.height > rect.y + rect.height) {
        break;
      }
    }
    return visible;
  }

}
