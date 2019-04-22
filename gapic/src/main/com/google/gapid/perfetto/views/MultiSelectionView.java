/*
 * Copyright (C) 2019 Google Inc.
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
package com.google.gapid.perfetto.views;

import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;

import com.google.gapid.perfetto.models.Selection;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.TabFolder;

/**
 * Displays multiple different selections.
 */
public class MultiSelectionView extends Composite {
  public MultiSelectionView(Composite parent, Selection[] selections, State state) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout());

    TabFolder folder = createStandardTabFolder(this);
    for (Selection s : selections) {
      createStandardTabItem(folder, s.getTitle(), s.buildUi(folder, state));
    }
  }
}
