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

import com.google.gapid.util.Trees;
import org.eclipse.jface.viewers.IBaseLabelProvider;
import org.eclipse.jface.viewers.IContentProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

/**
 * A {@link TreeViewer} that notifies the bound {@link IContentProvider} and
 * {@link IBaseLabelProvider} of time visibility changes if they also implement
 * the {@link Listener} interface.
 */
public class VisibilityTrackingTreeViewer extends TreeViewer {
  private Set<TreeItem> visible = new HashSet<>();

  public VisibilityTrackingTreeViewer(Tree tree) {
    super(tree);
    tree.addPaintListener(e -> updateVisibility());
  }

  private void updateVisibility() {
    List<Listener> listeners = getListeners();
    if (listeners.isEmpty()) {
      return;
    }

    Set<TreeItem> seen = Trees.calcVisibleItems(getTree());
    if (seen == null) {
      return; // No reliable data.
    }
    for (TreeItem item : visible) {
      if (!seen.contains(item)) {
        for (Listener listener : listeners) {
          listener.onHide(item);
        }
      }
    }
    for (TreeItem item : seen) {
      if (!visible.contains(item)) {
        for (Listener listener : listeners) {
          listener.onShow(item);
        }
      }
    }
    visible = seen;
  }

  private List<Listener> getListeners() {
    List<Listener> listeners = new ArrayList<>();
    IContentProvider contentProvider = getContentProvider();
    if (contentProvider instanceof Listener) {
      listeners.add((Listener)contentProvider);
    }
    IBaseLabelProvider labelProvider = getLabelProvider();
    if (labelProvider instanceof Listener) {
      listeners.add((Listener)labelProvider);
    }
    return listeners;
  }

  /**
   * The interface the {@link IContentProvider} and {@link IBaseLabelProvider} can
   * implement to be notified about {@link TreeItem}s being made visible and hidden
   * usually by scrolling.
   */
  public interface Listener {
    /**
     * Called when the {@link TreeItem} is made visible.
     */
    void onShow(TreeItem item);
    /**
     * Called when the {@link TreeItem} is made invisible.
     */
    void onHide(TreeItem item);
  }
}
