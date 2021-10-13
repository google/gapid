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
package com.google.gapid.perfetto.views;

import static com.google.gapid.widgets.Widgets.createCheckboxTableViewer;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static com.google.gapid.widgets.Widgets.withLayoutData;

import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.service.Service;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.CheckboxTableViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Text;

import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Set;

/**
 * Dialog shown to configure a Fuchsia System trace.
 */
public class FuchsiaTraceConfigDialog extends DialogBase {
  private static final String[] FUCHSIA_CATEGORIES = {
      "app", "audio", "benchmark", "blobfs","dart", "dart:compiler", "dart:dart", "dart:debugger",
      "dart:embedder", "dart:gc", "dart:isolate", "dart:profiler", "dart:vm", "flutter", "gfx",
      "input", "kernel:meta", "kernel:sched", "ledger", "magma", "minfs", "modular", "view",
  };

  private final Settings settings;
  private final Set<String> currentCategories;
  private final List<String> additionalCategories;

  private CheckboxTableViewer categories;

  public FuchsiaTraceConfigDialog(Shell shell, Settings settings, Theme theme) {
    super(shell, theme);
    this.settings = settings;
    this.currentCategories = Sets.newHashSet(settings.fuchsiaTracing().getCategoriesList());
    this.additionalCategories =
        Lists.newArrayList(Sets.difference(currentCategories, Sets.newHashSet(FUCHSIA_CATEGORIES)));
    Collections.sort(additionalCategories);
  }

  public static void showFuchsiaConfigDialog(Shell shell, Models models, Widgets widgets) {
    new FuchsiaTraceConfigDialog(shell, models.settings, widgets.theme).open();
  }

  public static String getConfigSummary(Settings settings) {
    SettingsProto.FuchsiaTracingOrBuilder f = settings.fuchsiaTracing();
    switch (f.getCategoriesCount()) {
      case 0:
        return "Default categories";
      case 1:
        return "1 category";
      default:
        return f.getCategoriesCount() + " categories";
    }
  }

  public static Service.FuchsiaTraceConfig.Builder getConfig(Settings settings) {
    return Service.FuchsiaTraceConfig.newBuilder()
        .addAllCategories(settings.fuchsiaTracing().getCategoriesList());
  }

  @Override
  public String getTitle() {
    return Messages.CAPTURE_TRACE_FUCHSIA;
  }

  @Override
  protected Control createDialogArea(Composite parent) {
    Composite area = (Composite)super.createDialogArea(parent);
    Composite container = withLayoutData(createComposite(area, new FillLayout()),
        new GridData(SWT.FILL, SWT.FILL, true, true));

    Group catGroup = createGroup(container, "Categories", new GridLayout(1, false));
    categories = createCheckboxTableViewer(catGroup, SWT.NONE);
    categories.getTable().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    categories.getTable().setHeaderVisible(false);
    categories.setContentProvider(new ArrayContentProvider());

    updateCategories();
    categories.addCheckStateListener(event -> {
      if (event.getChecked()) {
        currentCategories.add((String)event.getElement());
      } else {
        currentCategories.remove(event.getElement());
      }
    });

    createLink(catGroup, "Select <a>none / default</a> | <a>all</a>", e -> {
      switch (e.text) {
        case "none / default":
          currentCategories.clear();
          categories.setAllChecked(false);
          break;
        case "all":
          currentCategories.clear();
          currentCategories.addAll(Arrays.asList(FUCHSIA_CATEGORIES));
          currentCategories.addAll(additionalCategories);
          categories.setAllChecked(true);
          break;
      }
    });

    Composite custom = withLayoutData(createComposite(catGroup, new GridLayout(3, false)),
        new GridData(SWT.FILL, SWT.BOTTOM, true, false));
    withLayoutData(createLabel(custom, "Custom:"),
        new GridData(SWT.LEFT, SWT.CENTER, false, false));
    Text customCat = withLayoutData(createTextbox(custom, ""),
        new GridData(SWT.FILL, SWT.CENTER, true, false));
    Button customAdd = withLayoutData(Widgets.createButton(custom, "Add", e -> {
      String cat = customCat.getText().trim();
      additionalCategories.add(cat);
      currentCategories.add(cat);

      categories.add(cat);
      categories.setChecked(cat, true);
      categories.reveal(cat);

      customCat.setText("");
      ((Button)e.widget).setEnabled(false);
    }), new GridData(SWT.RIGHT, SWT.CENTER, false, false));
    customAdd.setEnabled(false);
    customCat.addListener(SWT.Modify, e -> {
      String cat = customCat.getText().trim();
      customAdd.setEnabled(!cat.isEmpty() && !cat.contains(","));
    });

    return area;
  }

  @Override
  protected Point getInitialSize() {
    return new Point(convertHorizontalDLUsToPixels(300), convertVerticalDLUsToPixels(250));
  }

  @Override
  protected void okPressed() {
    SettingsProto.FuchsiaTracing.Builder fuchsia = settings.writeFuchsiaTracing();
    fuchsia.clearCategories();
    fuchsia.addAllCategories(currentCategories);

    super.okPressed();
  }

  private void updateCategories() {
    List<String> items = Lists.newArrayList(FUCHSIA_CATEGORIES);
    items.addAll(additionalCategories);
    categories.setInput(items);

    categories.setCheckedElements(currentCategories.toArray(String[]::new));
  }
}
