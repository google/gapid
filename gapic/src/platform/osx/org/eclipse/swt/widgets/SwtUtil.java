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
package org.eclipse.swt.widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.internal.cocoa.NSClipView;
import org.eclipse.swt.internal.cocoa.NSPoint;
import org.eclipse.swt.internal.cocoa.NSRect;
import org.eclipse.swt.internal.cocoa.OS;

public class SwtUtil {
  private SwtUtil() {
  }

  public static void disableAutoHideScrollbars(Scrollable widget) {
    if (OS.VERSION >= 0x1070) {
      OS.objc_msgSend(widget.scrollView.id,
          OS.sel_registerName("setScrollerStyle:"), 0 /* NSScrollerStyleLegacy */);
      widget.scrollView.setAutohidesScrollers(false);
    }
  }

  public static void syncTreeAndTableScroll(Tree tree, Table table) {
    NSClipView treeView = tree.scrollView.contentView();
    NSClipView tableView = table.scrollView.contentView();
    Runnable updateTree = () -> {
      NSRect left = treeView.bounds();
      NSRect right = tableView.bounds();
      NSPoint point = new NSPoint();
      point.x = left.x;
      point.y = right.y;
      tree.view.scrollPoint(point);
    };
    Runnable updateTable = () -> {
      NSRect left = treeView.bounds();
      NSRect right = tableView.bounds();
      NSPoint point = new NSPoint();
      point.x = right.x;
      point.y = left.y;
      table.view.scrollPoint(point);
    };

    tree.getVerticalBar().addListener(SWT.Selection, e -> updateTable.run());
    table.getVerticalBar().addListener(SWT.Selection, e -> updateTree.run());

    tree.addListener(SWT.MouseWheel, event -> table.getDisplay().asyncExec(updateTable));
    table.addListener(SWT.MouseWheel, event -> tree.getDisplay().asyncExec(updateTree));

    // Make sure the rows are the same height in the table as the tree.
    // Note that while setItemHeight() forwards directly to the OS, getItemHeight() adds in a gap.
    int[] height = { tree.getItemHeight() };
    table.addListener(SWT.Paint, event -> {
      height[0] = tree.getItemHeight();
      if (table.getItemHeight() != height[0]) {
        table.setItemHeight(height[0] - Table.CELL_GAP);
      }
    });

    disableAutoHideScrollbars(tree);
    disableAutoHideScrollbars(table);
  }
}
