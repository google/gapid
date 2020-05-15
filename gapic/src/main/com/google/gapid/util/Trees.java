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
package com.google.gapid.util;

import static com.google.gapid.util.GeoUtils.bottom;
import static java.util.Collections.emptySet;

import com.google.common.collect.Sets;

import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Set;

/**
 * Utilities for dealing with {@link Tree Trees} and {@link TreeItem TreeItems}.
 */
public class Trees {
  private Trees() {
  }

  public static TreeItem getTopItem(Tree tree) {
    if (!OS.isLinux) {
      return tree.getTopItem();
    }

    // Dirty work around for https://bugs.eclipse.org/bugs/show_bug.cgi?id=563232.
    // Note this will often return an item that is above the visible one, as it only considers
    // root items. The actual correct top item may be a child of the returned item.
    int count = tree.getItemCount();
    if (count == 0) {
      return null;
    }

    TreeItem item = tree.getItem(0);
    if (count <= 2) {
      return item;
    }
    Rectangle bounds = item.getBounds();
    if (bounds.y >= 0 || (bounds.y <= 0 && bounds.y + bounds.height >= 0)) {
      return item;
    }

    int start = 0, end = count - 1;
    while (end - start > 1) {
      int mid = (start + end) / 2;
      item = tree.getItem(mid);
      bounds = item.getBounds();
      if (bounds.y <= 0 && bounds.y + bounds.height >= 0) {
        return item;
      } else if (bounds.y > 0) {
        end = mid;
      } else {
        start = mid;
      }
    }
    return tree.getItem(start);
  }

  /**
   * @return the {@link TreeItem TreeItems} that are currently visible in the tree.
   */
  public static Set<TreeItem> getVisibleItems(Tree tree) {
    TreeItem top = getTopItem(tree);
    if (top == null) {
      // Work around bug where getTopItem() returns null when scrolling
      // up past the top item (elastic scroll).
      return (tree.getItemCount() != 0) ? null : emptySet();
    }

    int treeBottom = bottom(tree.getClientArea());
    Set<TreeItem> visible = Sets.newIdentityHashSet();
    if (!getVisibleItems(top, treeBottom, visible)) {
      do {
        top = getVisibleSiblings(top, treeBottom, visible);
      } while (top != null);
    }
    return visible;
  }

  /**
   * Adds the given item and all visible descendants into the given set.
   * @return whether the bottom has been reached.
   */
  private static boolean getVisibleItems(TreeItem item, int treeBottom, Set<TreeItem> visible) {
    visible.add(item);
    if (bottom(item.getBounds()) > treeBottom) {
      return true;
    }

    if (item.getExpanded()) {
      for (TreeItem child : item.getItems()) {
        if (getVisibleItems(child, treeBottom, visible)) {
          return true;
        }
      }
    }
    return false;
  }

  /**
   * Adds all visible siblings (and their visible descendants) into the given set.
   * @return the next parent or {@code null} if the bottom has been reached.
   */
  private static TreeItem getVisibleSiblings(TreeItem item, int treeBottom, Set<TreeItem> visible) {
    TreeItem parent = item.getParentItem();
    TreeItem[] siblings = (parent == null) ? item.getParent().getItems() : parent.getItems();
    int idx = 0;
    for (; idx < siblings.length && siblings[idx] != item; idx++) {
      // Do nothing.
    }
    for (idx++; idx < siblings.length; idx++) {
      if (getVisibleItems(siblings[idx], treeBottom, visible)) {
        return null;
      }
    }
    return parent;
  }
}
