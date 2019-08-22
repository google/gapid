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

import static com.google.gapid.perfetto.views.StyleConstants.KB_DELAY;
import static com.google.gapid.perfetto.views.StyleConstants.KB_PAN_FAST;
import static com.google.gapid.perfetto.views.StyleConstants.KB_PAN_SLOW;
import static com.google.gapid.perfetto.views.StyleConstants.KB_ZOOM_FAST;
import static com.google.gapid.perfetto.views.StyleConstants.KB_ZOOM_SLOW;
import static com.google.gapid.perfetto.views.StyleConstants.ZOOM_FACTOR_SCALE;

import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.PanelCanvas;
import com.google.gapid.perfetto.views.RootPanel;
import com.google.gapid.perfetto.views.SelectionView;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.util.Keyboard;
import com.google.gapid.util.Loadable;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.ScrollBar;

/**
 * The main entry point of the Perfetto trace UI.
 */
public class TraceView extends Composite
    implements Capture.Listener, Perfetto.Listener, State.Listener {
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
        if (rootPanel.zoom(e.x, 1.0 - Math.max(-3, Math.min(3, e.count)) * ZOOM_FACTOR_SCALE)) {
          canvas.redraw(Area.FULL, true);
        }
      }
    });
    canvas.addListener(SWT.MouseHorizontalWheel, e -> {
      if ((e.stateMask & SWT.MODIFIER_MASK) == SWT.MOD1) {
        // Ignore horizontal scroll, only when zooming.
        e.doit = false;
      }
    });
    canvas.addListener(SWT.Gesture, this::handleGesture);
    canvas.getHorizontalBar().addListener(SWT.Selection, e -> {
      TimeSpan trace = state.getTraceTime();
      int sel = canvas.getHorizontalBar().getSelection();
      if (state.scrollToX(trace.start + sel * trace.getDuration() / 10000)) {
        canvas.redraw(Area.FULL, true);
      }
    });
    canvas.getVerticalBar().addListener(SWT.Selection, e -> {
      int sel = canvas.getVerticalBar().getSelection();
      if (state.scrollToY(sel)) {
        canvas.redraw(Area.FULL, true);
      }
    });
    new Keyboard(canvas, KB_DELAY, kb -> {
      boolean redraw = false;
      boolean fast = kb.isKeyDown(SWT.SHIFT);
      if (kb.isKeyDown('a') || kb.isKeyDown(SWT.ARROW_LEFT)) {
        redraw = state.dragX(state.getVisibleTime(), fast ? KB_PAN_FAST : KB_PAN_SLOW) || redraw;
      } else if (kb.isKeyDown('d') || kb.isKeyDown(SWT.ARROW_RIGHT)) {
        redraw = state.dragX(state.getVisibleTime(), -(fast ? KB_PAN_FAST : KB_PAN_SLOW)) || redraw;
      }

      if (kb.isKeyDown('w') || kb.isKeyDown(SWT.ARROW_UP)) {
        redraw = state.dragY(fast ? KB_PAN_FAST : KB_PAN_SLOW) || redraw;
      } else if (kb.isKeyDown('s') || kb.isKeyDown(SWT.ARROW_DOWN)) {
        redraw = state.dragY(-(fast ? KB_PAN_FAST : KB_PAN_SLOW)) || redraw;
      }

      if (kb.isKeyDown('q')) {
        Point mouse = canvas.toControl(getDisplay().getCursorLocation());
        redraw = rootPanel.zoom(mouse.x, 1.0 - (fast ? KB_ZOOM_FAST : KB_ZOOM_SLOW)) || redraw;
      } else if (kb.isKeyDown('e')) {
        Point mouse = canvas.toControl(getDisplay().getCursorLocation());
        redraw = rootPanel.zoom(mouse.x, 1.0 + (fast ? KB_ZOOM_FAST : KB_ZOOM_SLOW)) || redraw;
      }

      if (redraw) {
        canvas.redraw(Area.FULL, true);
      }
    });
    updateScrollbars();

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
    updateScrollbars();
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
    updateScrollbars();
  }

  @Override
  public void onVisibleAreaChanged() {
    updateScrollbars();
  }

  private double lastZoom = 1;
  private void handleGesture(Event e) {
    switch (e.detail) {
      case SWT.GESTURE_BEGIN:
        lastZoom = 1;
        break;
      case SWT.GESTURE_MAGNIFY:
        if (rootPanel.zoom(e.x, lastZoom / e.magnification)) {
          canvas.redraw(Area.FULL, true);
        }
        lastZoom = e.magnification;
        break;
      case SWT.GESTURE_END:
        break;
    }
  }

  private void updateScrollbars() {
    updateHorizontalBar();
    updateVerticalBar();
  }

  private void updateHorizontalBar() {
    ScrollBar bar = canvas.getHorizontalBar();
    if (!models.perfetto.isLoaded()) {
      bar.setEnabled(false);
      bar.setValues(0, 0, 1, 1, 5, 10);
      return;
    }

    TimeSpan visible = state.getVisibleTime();
    TimeSpan total = state.getTraceTime();
    if (total.getDuration() == 0) {
      bar.setEnabled(false);
      bar.setValues(0, 0, 1, 1, 5, 10);
      return;
    }

    int sel = permyriad(visible.start - total.start, total.getDuration());
    int thumb = permyriad(visible.getDuration(), total.getDuration());

    bar.setEnabled(true);
    bar.setValues(sel, 0, 10000, thumb, Math.max(1, thumb / 20), 100);
  }

  private void updateVerticalBar() {
    ScrollBar bar = canvas.getVerticalBar();
    if (!models.perfetto.isLoaded()) {
      bar.setEnabled(false);
      bar.setValues(0, 0, 1, 1, 5, 10);
      return;
    }

    double max = state.getMaxScrollOffset();
    if (max <= 0) {
      bar.setEnabled(false);
      bar.setValues(0, 0, 1, 1, 5, 10);
      return;
    }

    int h = canvas.getClientArea().height;
    int sel = (int)state.getScrollOffset();

    bar.setEnabled(true);
    bar.setValues(sel, 0, (int)(h + max), h, 10, 100);
  }

  private static int permyriad(long v, long t) {
    return Math.max(0, Math.min(10000, (int)(10000 * v / t)));
  }
}
