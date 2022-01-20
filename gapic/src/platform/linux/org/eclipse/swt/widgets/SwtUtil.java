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

public class SwtUtil {
  private SwtUtil() {
  }

  public static void disableAutoHideScrollbars(@SuppressWarnings("unused") Scrollable widget) {
    // Do nothing.
  }

  public static void syncTreeAndTableScroll(Tree tree, Table table) {
    ScrollBar bar1 = tree.getVerticalBar();
    ScrollBar bar2 = table.getVerticalBar();
    bar1.addListener(SWT.Selection, event -> bar2.setSelection(bar1.getSelection()));
    bar2.addListener(SWT.Selection, event -> bar1.setSelection(bar2.getSelection()));
    bar1.setVisible(false);
    tree.getHorizontalBar().setVisible(true);
    table.getHorizontalBar().setVisible(true);
  }
}
