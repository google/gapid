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
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.PanelCanvas;
import com.google.gapid.perfetto.views.RootPanel;
import com.google.gapid.perfetto.views.SelectionView;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.util.Loadable;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.ScrollBar;

/**
 * The main entry point of the Perfetto trace UI.
 */
public class TraceView extends Composite
    implements Capture.Listener, Perfetto.Listener, State.Listener {
  private static final double ZOOM_FACTOR_SCALE = 0.05;

  private final Models models;
  private final State state;
  private final LoadablePanel<SashForm> loading;
  private final RootPanel rootPanel;
  private final PanelCanvas canvas;

  public TraceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.state = new State(this);

    setLayout(new FillLayout());

    loading = new LoadablePanel<SashForm>(this, widgets, p -> new SashForm(p, SWT.VERTICAL));
    SashForm topBottom = loading.getContents();
    rootPanel = new RootPanel(state);
    canvas = new PanelCanvas(topBottom, SWT.H_SCROLL, widgets.theme, rootPanel);
    new SelectionView(topBottom, state);
    topBottom.setWeights(models.settings.perfettoSplitterWeights);

    canvas.addListener(SWT.MouseWheel, e -> {
      if ((e.stateMask & SWT.MODIFIER_MASK) == SWT.MOD1) {
        e.doit = false;
        if (rootPanel.zoom(e.x, Math.max(-3, Math.min(3, e.count)) * ZOOM_FACTOR_SCALE)) {
          canvas.redraw(Area.FULL);
        }
      }
    });
    canvas.getHorizontalBar().addListener(SWT.Selection, e -> {
      TimeSpan trace = state.getTraceTime();
      int sel = canvas.getHorizontalBar().getSelection();
      if (state.scrollTo(trace.start + sel * trace.getDuration() / 1000)) {
        canvas.redraw(Area.FULL);
      }
    });
    updateScrollbar();

    models.capture.addListener(this);
    models.perfetto.addListener(this);
    state.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.perfetto.removeListener(this);
      models.settings.perfettoSplitterWeights = topBottom.getWeights();
    });

    if (!models.perfetto.isLoaded()) {
      loading.startLoading();
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.startLoading();
    if (!maintainState) {
      rootPanel.clear();
    }
    updateScrollbar();
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
      state.update(models.perfetto.getData());
      canvas.structureHasChanged();
    }
    updateScrollbar();
  }

  @Override
  public void onVisibleTimeChanged() {
    updateScrollbar();
  }

  private void updateScrollbar() {
    ScrollBar bar = canvas.getHorizontalBar();
    if (!models.perfetto.isLoaded()) {
      bar.setEnabled(false);
      bar.setValues(0, 0, 1, 1, 5, 10);
      return;
    }

    TimeSpan visible = state.getVisibleTime();
    TimeSpan total = state.getTraceTime();
    int sel = permille(visible.start - total.start, total.getDuration());
    int thumb = permille(visible.getDuration(), total.getDuration());

    bar.setEnabled(true);
    bar.setValues(sel, 0, 1000, thumb, 50, 100);
  }

  private static int permille(long v, long t) {
    return Math.max(0, Math.min(1000, (int)(1000 * v / t)));
  }
}
