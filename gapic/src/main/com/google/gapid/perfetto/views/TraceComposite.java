/*
 * Copyright (C) 2020 Google Inc.
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

import static com.google.gapid.perfetto.views.KeyboardMouseHelpDialog.showHelp;
import static com.google.gapid.perfetto.views.StyleConstants.KB_DELAY;
import static com.google.gapid.perfetto.views.StyleConstants.KB_PAN_FAST;
import static com.google.gapid.perfetto.views.StyleConstants.KB_PAN_SLOW;
import static com.google.gapid.perfetto.views.StyleConstants.KB_ZOOM_FAST;
import static com.google.gapid.perfetto.views.StyleConstants.KB_ZOOM_SLOW;
import static com.google.gapid.perfetto.views.StyleConstants.TP_PAN_FAST;
import static com.google.gapid.perfetto.views.StyleConstants.TP_PAN_SLOW;
import static com.google.gapid.perfetto.views.StyleConstants.ZOOM_FACTOR_SCALE;
import static com.google.gapid.widgets.Widgets.createButtonWithImage;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSizeHints;

import com.google.gapid.models.Analytics;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.canvas.PanelCanvas;
import com.google.gapid.perfetto.models.Selection.MultiSelection;
import com.google.gapid.perfetto.views.RootPanel.MouseMode;
import com.google.gapid.util.Keyboard;
import com.google.gapid.widgets.Theme;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.ScrollBar;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.ToolBar;

import java.util.function.Consumer;

/**
 * A {@link Composite} that displays a Perfetto trace.
 */
public abstract class TraceComposite<S extends State> extends Composite implements State.Listener {
  protected final S state;

  private final RootPanel<S> rootPanel;
  private final PanelCanvas canvas;

  public TraceComposite(Composite parent, Analytics analytics, Theme theme) {
    super(parent, SWT.NONE);
    this.state = createState();
    this.rootPanel = createRootPanel();
    state.addListener(this);

    setLayout(withMargin(new GridLayout(1, false), 0, 0));
    TopBar topBar = withLayoutData(new TopBar(this, analytics, theme, this::updateFilter),
        new GridData(SWT.FILL, SWT.TOP, true, false));
    canvas = withLayoutData(new PanelCanvas(this, SWT.H_SCROLL | SWT.V_SCROLL, theme, rootPanel),
        new GridData(SWT.FILL, SWT.FILL, true, true));

    Consumer<RootPanel.MouseMode> modeSelector =
        topBar.buildModeActions(theme, m -> rootPanel.setMouseMode(m));
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
      int mods = e.stateMask & SWT.MODIFIER_MASK;
      // Treat horizontal touch pad/scroll wheel ourselves, rather than going through the
      // scrollbar, since the scrollbar's size is limited.
      e.doit = false;
      if (mods == SWT.MOD1) {
        // Ignore horizontal scroll, when zooming.
      } else {
        if (state.dragX(
            state.getVisibleTime(), e.count * ((mods == SWT.SHIFT) ? TP_PAN_FAST : TP_PAN_SLOW))) {
          canvas.redraw(Area.FULL, true);
        }
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
        case 'h':
          KeyboardMouseHelpDialog.showHelp(getShell(), analytics, theme);
          break;
        case 'm': {
          MultiSelection selection = state.getSelection();
          if (selection != null) {
            selection.markTime(state);
            redraw = true;
          }
          break;
        }
        case 'v':
          rootPanel.toggleVSync();
          redraw = true;
          break;
        case 'z':
        case '0':
          redraw = state.setVisibleTime(state.getTraceTime());
          break;
      }

      switch (e.character) {
        case '?':
          KeyboardMouseHelpDialog.showHelp(getShell(), analytics, theme);
          break;
      }

      if (redraw) {
        canvas.redraw(Area.FULL, true);
      }
    });
    updateScrollbars();
  }

  protected abstract S createState();
  protected abstract RootPanel<S> createRootPanel();

  private void updateFilter(String search) {
    rootPanel.updateFilter(search);
    canvas.structureHasChanged();
  }

  public S getState() {
    return state;
  }

  public void requestFocus() {
    scheduleIfNotDisposed(canvas, canvas::setFocus);
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

  @Override
  public void onVisibleAreaChanged() {
    updateScrollbars();
  }

  @Override
  public void onDataChanged() {
    canvas.structureHasChanged();
    updateScrollbars();
  }

  @Override
  public void onSelectionChanged(MultiSelection selection) {
    canvas.redraw();
  }

  private void updateScrollbars() {
    updateHorizontalBar();
    updateVerticalBar();
  }

  private void updateHorizontalBar() {
    ScrollBar bar = canvas.getHorizontalBar();
    TimeSpan visible = state.getVisibleTime();
    TimeSpan total = state.getTraceTime();
    if (total.getDuration() == 0) {
      disableScrollBar(bar);
      return;
    }

    int sel = permyriad(visible.start - total.start, total.getDuration());
    int thumb = permyriad(visible.getDuration(), total.getDuration());

    bar.setEnabled(true);
    bar.setValues(sel, 0, 10000, thumb, Math.max(1, thumb / 20), 100);
  }

  private void updateVerticalBar() {
    ScrollBar bar = canvas.getVerticalBar();
    double max = state.getMaxScrollOffset();
    if (max <= 0) {
      disableScrollBar(bar);
      return;
    }

    int h = canvas.getClientArea().height;
    int sel = (int)state.getScrollOffset();

    bar.setEnabled(true);
    bar.setValues(sel, 0, (int)(h + max), h, 10, 100);
  }

  private static void disableScrollBar(ScrollBar bar) {
    bar.setEnabled(false);
    bar.setValues(0, 0, 1, 1, 5, 10);
  }

  private static int permyriad(long v, long t) {
    return Math.max(0, Math.min(10000, (int)(10000 * v / t)));
  }

  private static class TopBar extends Composite {
    private final ToolBar toolBar;

    public TopBar(Composite parent, Analytics analytics, Theme theme, Consumer<String> onSearch) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(4, false));
      withLayoutData(createLabel(this, "Mode:"),
          new GridData(SWT.BEGINNING, SWT.CENTER, false, false));
      toolBar = withLayoutData(new ToolBar(this, SWT.FLAT | SWT.HORIZONTAL | SWT.TRAIL),
          new GridData(SWT.BEGINNING, SWT.CENTER, false, false));
      Text search = withLayoutData(
          new Text(this, SWT.SINGLE | SWT.SEARCH | SWT.ICON_SEARCH | SWT.ICON_CANCEL),
          withSizeHints(new GridData(SWT.BEGINNING, SWT.CENTER, false, false), 300, SWT.DEFAULT));
      search.setMessage("Filter tracks by name...");
      withLayoutData(
          createButtonWithImage(this, theme.help(), e -> showHelp(getShell(), analytics, theme)),
          new GridData(SWT.END, SWT.CENTER, true, false));

      search.addListener(SWT.Modify, e -> {
        onSearch.accept(search.getText().trim().toLowerCase());
      });
    }

    public Consumer<RootPanel.MouseMode> buildModeActions(Theme theme, Consumer<MouseMode> onClick) {
      return RootPanel.MouseMode.createToolBar(toolBar, theme, onClick);
    }
  }
}
