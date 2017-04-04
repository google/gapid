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
import com.google.gapid.util.Tables;

import org.eclipse.jface.viewers.IBaseLabelProvider;
import org.eclipse.jface.viewers.IContentProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.swt.widgets.Table;
import org.eclipse.swt.widgets.TableItem;

import java.util.Collections;
import java.util.List;
import java.util.Set;

/**
 * A {@link TableViewer} that notifies bound {@link IContentProvider} and
 * {@link IBaseLabelProvider} of visibility changes if they also implement
 * the {@link Listener} interface.
 */
public class VisibilityTrackingTableViewer extends TableViewer {
  private Set<TableItem> visible = Collections.emptySet();
  private Listener[] listeners;

  public VisibilityTrackingTableViewer(Table table) {
    super(table);
    updateListeners();
    table.addPaintListener(e -> updateVisibility());
  }

  private void updateVisibility() {
    if (listeners == null) {
      return;
    }

    Set<TableItem> seen = Tables.getVisibleItems(getTable());
    if (seen == null) {
      return; // No reliable data.
    }

    for (TableItem item : visible) {
      if (!seen.contains(item) && !item.isDisposed()) {
        for (Listener listener : listeners) {
          listener.onHide(item);
        }
      }
    }
    for (TableItem item : seen) {
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
   * implement to be notified about {@link TableItem TableItems} being made visible
   * and hidden usually by scrolling.
   */
  public static interface Listener {
    /**
     * Called when the {@link TableItem} is made visible.
     */
    public void onShow(TableItem item);

    /**
     * Called when the {@link TableItem} is made invisible.
     */
    public void onHide(TableItem item);
  }
}
