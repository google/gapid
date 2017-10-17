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
}
