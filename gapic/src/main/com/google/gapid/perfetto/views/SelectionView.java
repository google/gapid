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

import com.google.gapid.perfetto.models.Selection;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

/**
 * Displays information about the current selection.
 */
public class SelectionView extends Composite implements State.Listener {
  private final State state;

  public SelectionView(Composite parent, State state) {
    super(parent, SWT.NONE);
    this.state = state;

    setLayout(new FillLayout());

    state.addListener(this);
  }

  @Override
  public void onDataChanged() {
    onSelectionChanged(state.getSelection());
  }

  @Override
  public void onSelectionChanged(Selection.MultiSelection selection) {
    for (Control c : getChildren()) {
      c.dispose();
    }

    if (selection != null) {
      Composite composite = selection.buildUi(this, state);
      if (composite != null) {
        composite.requestLayout();
      }
    }
  }
}
