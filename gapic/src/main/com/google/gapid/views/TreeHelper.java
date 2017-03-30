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

import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

/**
 * A static helper class for dealing with {@link Tree}s and {@link TreeItem}s.
 */
public class TreeHelper {
  /**
   * @return the i'th child {@link TreeItem} of the {@link Tree} or {@link TreeItem}.
   */
  public static TreeItem getItem(Object treeOrTreeItem, int index) {
    if (treeOrTreeItem instanceof Tree) {
      return ((Tree)treeOrTreeItem).getItem(index);
    }
    return ((TreeItem)treeOrTreeItem).getItem(index);
  }

  /**
   * @return the index of the child {@link TreeItem} of the {@link Tree} or {@link TreeItem}.
   */
  public static int indexOf(Object treeOrTreeItem, TreeItem child) {
    if (treeOrTreeItem instanceof Tree) {
      return ((Tree)treeOrTreeItem).indexOf(child);
    }
    return ((TreeItem)treeOrTreeItem).indexOf(child);
  }

  /**
   * @return the number of child {@link TreeItem}s of the {@link Tree} or {@link TreeItem}.
   */
  public static int itemCount(Object treeOrTreeItem) {
    if (treeOrTreeItem instanceof Tree) {
      return ((Tree)treeOrTreeItem).getItemCount();
    }
    return ((TreeItem)treeOrTreeItem).getItemCount();
  }

  /**
   * @return the parent {@link Tree} or {@link TreeItem} of the {@link TreeItem}.
   */
  public static Object getParent(Object treeOrTreeItem) {
    if (!(treeOrTreeItem instanceof TreeItem)) {
      return null;
    }
    TreeItem item = (TreeItem)treeOrTreeItem;
    TreeItem parentItem = item.getParentItem();
    if (parentItem != null) {
      return parentItem;
    }
    return item.getParent();
  }
}
