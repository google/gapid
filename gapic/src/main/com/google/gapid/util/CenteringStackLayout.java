/*
 * Copyright (C) 2021 Google Inc.
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

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StackLayout;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

/**
 * A {@link StackLayout} that vertically centers the control currently being displayed.
 */
public class CenteringStackLayout extends StackLayout {
  @Override
  protected void layout(Composite composite, boolean flushCache) {
    Rectangle rect = composite.getClientArea();
    rect.x += marginWidth;
    rect.y += marginHeight;
    rect.width -= 2 * marginWidth;
    rect.height -= 2 * marginHeight;
    for (Control element : composite.getChildren()) {
      Point size = element.computeSize(rect.width, SWT.DEFAULT);
      Rectangle bounds = rect;
      if (size.y < rect.height) {
        bounds = new Rectangle(rect.x, rect.y + (rect.height - size.y) / 2, rect.width, size.y);
      }
      element.setBounds(bounds);
      element.setVisible(element == topControl);
    }
  }
}
