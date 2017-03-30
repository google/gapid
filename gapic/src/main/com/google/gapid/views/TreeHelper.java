package com.google.gapid.views;

import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

/**
 * Created by bclayton on 3/30/17.
 */
public class TreeHelper {
  public static TreeItem getItem(Object treeOrTreeItem, int index) {
    if (treeOrTreeItem instanceof Tree) {
      return ((Tree)treeOrTreeItem).getItem(index);
    }
    return ((TreeItem)treeOrTreeItem).getItem(index);
  }

  public static int indexOf(Object treeOrTreeItem, TreeItem child) {
    if (treeOrTreeItem instanceof Tree) {
      return ((Tree)treeOrTreeItem).indexOf(child);
    }
    return ((TreeItem)treeOrTreeItem).indexOf(child);
  }

  public static int itemCount(Object treeOrTreeItem) {
    if (treeOrTreeItem instanceof Tree) {
      return ((Tree)treeOrTreeItem).getItemCount();
    }
    return ((TreeItem)treeOrTreeItem).getItemCount();
  }

  public static Object getParent(Object treeOrTreeItem) {
    if (treeOrTreeItem instanceof Tree) {
      return null;
    }
    TreeItem item = (TreeItem)treeOrTreeItem;
    TreeItem parentItem = item.getParentItem();
    if (parentItem != null) {
      return null;
    }
    return item.getParent();
  }
}
