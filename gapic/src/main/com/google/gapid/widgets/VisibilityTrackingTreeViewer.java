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

import com.google.common.collect.Lists;
import com.google.gapid.util.Trees;

import org.eclipse.jface.viewers.IBaseLabelProvider;
import org.eclipse.jface.viewers.IContentProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Collections;
import java.util.List;
import java.util.Set;

/**
 * A {@link TreeViewer} that notifies the bound {@link IContentProvider} and
 * {@link IBaseLabelProvider} of visibility changes if they also implement
 * the {@link Listener} interface.
 */
public class VisibilityTrackingTreeViewer extends TreeViewer {
  private Set<TreeItem> visible = Collections.emptySet();
  private Listener[] listeners;

  public VisibilityTrackingTreeViewer(Tree tree) {
    super(tree);
    updateListeners();
    tree.addPaintListener(e -> updateVisibility());
  }

  private void updateVisibility() {
    if (listeners == null) {
      return;
    }

    Set<TreeItem> seen = Trees.getVisibleItems(getTree());
    if (seen == null) {
      return; // No reliable data.
    }
    for (TreeItem item : visible) {
      if (!seen.contains(item) && !item.isDisposed()) {
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

  @Override
  protected void inputChanged(Object input, Object oldInput) {
    // Cause all visible items to receive a show event.
    visible = Collections.emptySet();
    super.inputChanged(input, oldInput);
  }

  @Override
  public void setLabelProvider(IBaseLabelProvider labelProvider) {
    super.setLabelProvider(labelProvider);
    updateListeners();
  }

  @Override
  public void setContentProvider(IContentProvider provider) {
    super.setContentProvider(provider);
    updateListeners();
  }

  private void updateListeners() {
    List<Listener> found = Lists.newArrayList();
    IContentProvider contentProvider = getContentProvider();
    if (contentProvider instanceof Listener) {
      found.add((Listener)contentProvider);
    }

    IBaseLabelProvider labelProvider = getLabelProvider();
    if (labelProvider instanceof Listener) {
      found.add((Listener)labelProvider);
    }

    listeners = found.isEmpty() ? null : found.toArray(new Listener[found.size()]);
  }

  /**
   * The interface the {@link IContentProvider} and {@link IBaseLabelProvider} can
   * implement to be notified about {@link TreeItem TreeItems} being made visible
   * and hidden usually by scrolling.
   */
  public static interface Listener {
    /**
     * @param item the item that has been made visible.
     */
    public default void onShow(TreeItem item) { /* empty */ }

    /**
     * @param item the item that has been made invisible.
     */
    public default void onHide(TreeItem item) { /* empty */ }
  }
}
