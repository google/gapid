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
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;

import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.PanelCanvas;
import com.google.gapid.perfetto.models.Selection;
import com.google.gapid.perfetto.models.Selection.MultiSelection;
import com.google.gapid.perfetto.views.RootPanel;
import com.google.gapid.perfetto.views.RootPanel.MouseMode;
import com.google.gapid.perfetto.views.SelectionView;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.util.Keyboard;
import com.google.gapid.util.Loadable;
import com.google.gapid.widgets.DrawerComposite;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.ScrollBar;
import org.eclipse.swt.widgets.ToolBar;

import java.util.function.Consumer;

/**
 * The main entry point of the Perfetto trace UI.
 */
public class TraceView extends Composite
    implements Capture.Listener, Perfetto.Listener, State.Listener {
  private final Models models;
  private final State.ForSystemTrace state;
  private final LoadablePanel<DrawerComposite> loading;
  private final RootPanel rootPanel;
  private final PanelCanvas canvas;

  public TraceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.state = new State.ForSystemTrace(this);

    setLayout(withMargin(new GridLayout(1, false), 0, 0));

    TopBar topBar = withLayoutData(new TopBar(this), new GridData(SWT.FILL, SWT.TOP, true, false));
    loading = withLayoutData(
        new LoadablePanel<DrawerComposite>(this, widgets, p -> new DrawerComposite(
            p, SWT.NONE, models.settings.ui().getPerfetto().getDrawerHeight(), widgets.theme)),
        new GridData(SWT.FILL, SWT.FILL, true, true));
    rootPanel = new RootPanel(state);

    DrawerComposite container = loading.getContents();
    container.setText("Selection");
    canvas = new PanelCanvas(
        container.getMain(), SWT.H_SCROLL | SWT.V_SCROLL, widgets.theme, rootPanel);
    new SelectionView(container.getDrawer(), state);

    Consumer<RootPanel.MouseMode> modeSelector =
        topBar.buildModeActions(widgets.theme, m -> rootPanel.setMouseMode(m));
    modeSelector.accept(RootPanel.MouseMode.Pan);

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

    // Handle repeatable keys.
    new Keyboard(canvas, KB_DELAY, kb -> {
      boolean redraw = false;
      boolean shift = kb.hasMod(SWT.SHIFT), ctrl = kb.hasMod(SWT.MOD1);
      if (kb.isKeyDown('a') || kb.isKeyDown(SWT.ARROW_LEFT)) {
        redraw = state.dragX(state.getVisibleTime(), shift ? KB_PAN_FAST : KB_PAN_SLOW) || redraw;
      } else if (kb.isKeyDown('d') || kb.isKeyDown(SWT.ARROW_RIGHT)) {
        redraw = state.dragX(state.getVisibleTime(), -(shift ? KB_PAN_FAST : KB_PAN_SLOW)) || redraw;
      }

      if (kb.isKeyDown('q') || kb.isKeyDown(SWT.ARROW_UP)) {
        redraw = state.dragY(shift ? KB_PAN_FAST : KB_PAN_SLOW) || redraw;
      } else if (kb.isKeyDown('e') || kb.isKeyDown(SWT.ARROW_DOWN)) {
        redraw = state.dragY(-(shift ? KB_PAN_FAST : KB_PAN_SLOW)) || redraw;
      }

      if (kb.isKeyDown('w') || (ctrl && (kb.isKeyDown(SWT.KEYPAD_ADD) || kb.isKeyDown('=')))) {
        Point mouse = canvas.toControl(getDisplay().getCursorLocation());
        redraw = rootPanel.zoom(mouse.x, 1.0 - (shift ? KB_ZOOM_FAST : KB_ZOOM_SLOW)) || redraw;
      } else if (kb.isKeyDown('s') || (ctrl && (kb.isKeyDown(SWT.KEYPAD_SUBTRACT) || kb.isKeyDown('-')))) {
        Point mouse = canvas.toControl(getDisplay().getCursorLocation());
        redraw = rootPanel.zoom(mouse.x, 1.0 + (shift ? KB_ZOOM_FAST : KB_ZOOM_SLOW)) || redraw;
      }

      rootPanel.setPanOverride(kb.isKeyDown(SWT.SPACE));

      if (redraw) {
        canvas.redraw(Area.FULL, true);
      }
    });

    // Handle single Keys.
    canvas.addListener(SWT.KeyDown, e -> {
      boolean redraw = false;
      switch (e.keyCode) {
        case '1':
        case SWT.KEYPAD_1:
          modeSelector.accept(RootPanel.MouseMode.Select);
          break;
        case '2':
        case SWT.KEYPAD_2:
          modeSelector.accept(RootPanel.MouseMode.Pan);
          break;
        case '3':
        case SWT.KEYPAD_3:
          modeSelector.accept(RootPanel.MouseMode.Zoom);
          break;
        case '4':
        case SWT.KEYPAD_4:
          modeSelector.accept(RootPanel.MouseMode.TimeSelect);
          break;
        case SWT.ESC:
          state.resetSelections(); // Already causes a redraw.
          break;
        case 'f': {
          MultiSelection selection = state.getSelection();
          if (selection != null) {
            selection.zoom(state);
            redraw = true;
          }
          break;
        }
        case 'm': {
          MultiSelection selection = state.getSelection();
          if (selection != null) {
            selection.markTime(state);
            redraw = true;
          }
          break;
        }
        case 'z':
        case '0':
          redraw = state.setVisibleTime(state.getTraceTime());
          break;
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
      models.settings.writeUi().getPerfettoBuilder().setDrawerHeight(container.getDrawerHeight());
      models.capture.removeListener(this);
      models.perfetto.removeListener(this);
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

  @Override
  public void onSelectionChanged(Selection.MultiSelection selection) {
    canvas.redraw();
    loading.getContents().setExpanded(selection != null);
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

  private static class TopBar extends Composite {
    private final ToolBar toolBar;

    public TopBar(Composite parent) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(2, false));
      createLabel(this, "Mode:");
      toolBar = withLayoutData(new ToolBar(this, SWT.FLAT | SWT.HORIZONTAL | SWT.TRAIL),
          new GridData(SWT.FILL, SWT.FILL, true, true));
    }

    public Consumer<RootPanel.MouseMode> buildModeActions(Theme theme, Consumer<MouseMode> onClick) {
      return RootPanel.MouseMode.createToolBar(toolBar, theme, onClick);
    }
  }
}
