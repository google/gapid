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
package com.google.gapid.views;

import static com.google.gapid.proto.service.Service.ClientAction.Select;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.ApiContext;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.models.Models;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Dropdown that allows the user to select which API context to filter the commands by.
 */
public class ContextSelector extends Composite implements ApiContext.Listener {
  private final Models models;
  private final ComboViewer contextCombo;

  public ContextSelector(Composite parent, Models models) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));
    contextCombo = createDropDownViewer(this);
    contextCombo.setContentProvider(ArrayContentProvider.getInstance());
    contextCombo.setLabelProvider(new LabelProvider());

    models.contexts.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.contexts.removeListener(this);
    });

    contextCombo.getCombo().addListener(SWT.Selection, e -> {
      models.analytics.postInteraction(View.ContextSelector, "contet", Select);
      int selection = contextCombo.getCombo().getSelectionIndex();
      if (selection >= 0 && selection < models.contexts.count()) {
        models.contexts.selectContext(models.contexts.getData()[selection]);
      }
    });
  }

  @Override
  public void onContextsLoaded() {
    if (!models.contexts.isLoaded()) {
      return;
    }

    contextCombo.setInput(models.contexts.getData());
    contextCombo.refresh();

    onContextSelected(models.contexts.getSelectedContext());
  }

  @Override
  public void onContextSelected(FilteringContext context) {
    contextCombo.setSelection(new StructuredSelection(context));
  }
}
