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

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FormAttachment;
import org.eclipse.swt.layout.FormData;
import org.eclipse.swt.layout.FormLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Sash;

public abstract class FixedTopSplitter extends Composite {
  private static final int MIN_SIZE = 30;

  private final Control top, bottom;
  private final Sash sash;

  public FixedTopSplitter(Composite parent, int topHeight) {
    super(parent, SWT.NONE);
    setLayout(new FormLayout());

    top = createTopControl();
    sash = new Sash(this, SWT.HORIZONTAL);
    bottom = createBottomControl();

    top.setLayoutData(createTopFormData(sash));
    sash.setLayoutData(createSashFormData(topHeight));
    bottom.setLayoutData(createBottomFormData(sash));

    sash.addListener(SWT.Selection, e -> {
      Rectangle size = getClientArea();
      Rectangle sashSize = sash.getBounds();
      e.y = Math.max(MIN_SIZE, Math.min(size.height - sashSize.height - MIN_SIZE, e.y));
      if (e.y != sashSize.y) {
        ((FormData)sash.getLayoutData()).top = new FormAttachment(0, e.y);
        layout();
      }
    });
  }

  public int getTopHeight() {
    return ((FormData)sash.getLayoutData()).top.offset;
  }

  public void setTopVisible(boolean visible) {
    top.setVisible(visible);
    sash.setVisible(visible);
    if (visible) {
      bottom.setLayoutData(createBottomFormData(sash));
    } else {
      bottom.setLayoutData(createFullFormData());
    }
    layout();
  }

  private static FormData createTopFormData(Sash sash) {
    FormData data = new FormData();
    data.left = new FormAttachment(0, 0);
    data.right = new FormAttachment(100, 0);
    data.top = new FormAttachment(0, 0);
    data.bottom = new FormAttachment(sash, 0);
    return data;
  }

  private static FormData createSashFormData(int topHeight) {
    FormData data = new FormData();
    data.left = new FormAttachment(0, 0);
    data.right = new FormAttachment(100, 0);
    data.top = new FormAttachment(0, topHeight);
    return data;
  }

  private static FormData createBottomFormData(Sash sash) {
    FormData data = new FormData();
    data.left = new FormAttachment(0, 0);
    data.right = new FormAttachment(100, 0);
    data.top = new FormAttachment(sash, 0);
    data.bottom = new FormAttachment(100, 0);
    return data;
  }

  private static FormData createFullFormData() {
    FormData data = new FormData();
    data.left = new FormAttachment(0, 0);
    data.right = new FormAttachment(100, 0);
    data.top = new FormAttachment(0, 0);
    data.bottom = new FormAttachment(100, 0);
    return data;
  }

  protected abstract Control createTopControl();
  protected abstract Control createBottomControl();
}
