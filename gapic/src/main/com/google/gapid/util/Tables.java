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

import org.eclipse.swt.widgets.Table;
import org.eclipse.swt.widgets.TableItem;

import java.util.Set;

/**
 * Utilities for dealing with {@link Table Tables} and {@link TableItem TableItems}.
 */
public class Tables {
  private Tables() {
  }

  /**
   * @return the {@link TableItem TableItems} that are currently visible in the table.
   */
  public static Set<TableItem> getVisibleItems(Table table) {
    int items = table.getItemCount();
    if (items == 0) {
      return emptySet();
    }

    int tableBottom = bottom(table.getClientArea());
    Set<TableItem> visible = Sets.newIdentityHashSet();
    for (int index = table.getTopIndex(); index < items; index++) {
      TableItem item = table.getItem(index);
      if (!item.isDisposed()) {
        visible.add(item);
        if (bottom(item.getBounds()) > tableBottom) {
          break;
        }
      }
    }
    return visible;
  }
}
