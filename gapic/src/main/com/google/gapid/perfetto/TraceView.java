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
package com.google.gapid.perfetto;

import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.views.RootPanel;
import com.google.gapid.perfetto.views.SelectionView;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.TraceComposite;
import com.google.gapid.util.Loadable;
import com.google.gapid.widgets.DrawerComposite;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * The main entry point of the Perfetto trace UI.
 */
public class TraceView extends Composite
    implements Capture.Listener, Perfetto.Listener, State.Listener {
  private final Models models;
  private final LoadablePanel<DrawerComposite> loading;
  private final TraceComposite<State.ForSystemTrace> traceUi;

  public TraceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout());

    loading = new LoadablePanel<DrawerComposite>(this, widgets, p -> new DrawerComposite(
        p, SWT.NONE, models.settings.ui().getPerfetto().getDrawerHeight(), widgets.theme));

    DrawerComposite container = loading.getContents();
    container.setText("Selection");
    traceUi = createTraceUi(container.getMain(), models, widgets.theme);
    new SelectionView(container.getDrawer(), traceUi.getState());

    models.capture.addListener(this);
    models.perfetto.addListener(this);
    traceUi.getState().addListener(this);
    addListener(SWT.Dispose, e -> {
      models.settings.writeUi().getPerfettoBuilder().setDrawerHeight(container.getDrawerHeight());
      models.capture.removeListener(this);
      models.perfetto.removeListener(this);
    });

    if (!models.perfetto.isLoaded()) {
      loading.startLoading();
    }
  }

  private static TraceComposite<State.ForSystemTrace> createTraceUi(
      Composite parent, Models models, Theme theme) {
    return new TraceComposite<State.ForSystemTrace>(
        parent, models.analytics, models.perfetto, theme, /*fullView*/ true) {
      @Override
      protected State.ForSystemTrace createState() {
        return new State.ForSystemTrace(this);
      }

      @Override
      protected RootPanel<State.ForSystemTrace> createRootPanel() {
        return new RootPanel.ForSystemTrace(state, models.settings);
      }
    };
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.startLoading();
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(error);
    }
  }

  @Override
  public void onPerfettoLoadingStatus(Loadable.Message msg) {
    loading.showMessage(msg);
  }

  @Override
  public void onPerfettoLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(error);
    } else {
      loading.stopLoading();
      traceUi.getState().update(models.perfetto.getData());
      traceUi.requestFocus();
    }
  }

  @Override
  public void onSelectionChanged(Selection.MultiSelection selection) {
    loading.getContents().setExpanded(selection != null);
  }
}
