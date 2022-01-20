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
import org.eclipse.swt.internal.win32.OS;
import org.eclipse.swt.internal.win32.SCROLLINFO;

public class SwtUtil {
  private SwtUtil() {
  }

  public static void disableAutoHideScrollbars(@SuppressWarnings("unused") Scrollable widget) {
    // Do nothing.
  }

  public static void syncTreeAndTableScroll(Tree tree, Table table) {
    Exclusive exclusive = new Exclusive();
    table.getVerticalBar().addListener(SWT.Selection, e ->
      exclusive.runExclusive(() -> {
        int pos = table.getVerticalBar().getSelection();
        SCROLLINFO info = new SCROLLINFO ();
        info.cbSize = SCROLLINFO.sizeof;
        info.fMask = OS.SIF_POS;
        info.nPos = pos;
        OS.SetScrollInfo(tree.handle, OS.SB_VERT, info, true);
        OS.SendMessage(tree.handle, OS.WM_VSCROLL, OS.SB_THUMBPOSITION | (pos << 16), 0);
      }));

    Runnable updateTableScroll = () -> {
      exclusive.runExclusive(() -> table.setTopIndex(tree.getVerticalBar().getSelection()));
    };
    tree.getVerticalBar().addListener(SWT.Selection, e -> updateTableScroll.run());
    tree.addListener(SWT.Expand, e -> table.getDisplay().asyncExec(updateTableScroll));
    tree.addListener(SWT.Collapse, e -> table.getDisplay().asyncExec(updateTableScroll));

    // Make sure the rows are the same height in the table as the tree.
    int[] height = { tree.getItemHeight() };
    table.addListener(SWT.Paint, event -> {
      height[0] = tree.getItemHeight();
      if (table.getItemHeight() != height[0]) {
        table.setItemHeight(height[0]);
        updateTableScroll.run();
      }
    });
  }

  // Used to prevent infite loops from handling one event triggering another handled event.
  private static class Exclusive {
    private boolean ignore = false;

    public Exclusive() {
    }

    public void runExclusive(Runnable run) {
      if (!ignore) {
        ignore = true;
        try {
          run.run();
        } finally {
          ignore = false;
        }
      }
    }
  }
}
