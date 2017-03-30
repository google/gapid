package com.google.gapid.views;

import org.eclipse.jface.viewers.IBaseLabelProvider;
import org.eclipse.jface.viewers.IContentProvider;
import org.eclipse.jface.viewers.ILabelProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.logging.Logger;

/**
 * A virtual tree content provider.
 */
public class VirtualTreeViewer extends TreeViewer {
  private static final Logger LOG = Logger.getLogger(VirtualTreeViewer.class.getName());

  private Set<TreeItem> visible = new HashSet<>();

  public interface VisibilityListener {
    void OnShow(TreeItem item);
    void OnHide(TreeItem item);
  }

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
