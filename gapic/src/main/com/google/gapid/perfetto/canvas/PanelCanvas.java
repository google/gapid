package com.google.gapid.perfetto.canvas;

import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.State.Location;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.widgets.Theme;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;

import java.util.Map;
import java.util.Set;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Draws a {@link Panel} and manages the user interactions.
 */
public class PanelCanvas extends Canvas {
  private static final Logger LOG = Logger.getLogger(PanelCanvas.class.getName());

  public static final Flag<Boolean> showRedraws = Flags.value(
      "show-redraws", false, "Highlight canvas redraw areas", true);

  private final Panel panel;
  private final RenderContext.Global context;
  private Point mouseDown = null;
  private boolean dragging = false;
  private Panel.Dragger dragger = Panel.Dragger.NONE;
  private Panel.Hover hover = Panel.Hover.NONE;
  private Point lastMouse = new Point(-1, -1);

  public PanelCanvas(Composite parent, int style, Theme theme, Panel panel, State state) {
    super(parent, style | SWT.V_SCROLL | SWT.NO_BACKGROUND | SWT.DOUBLE_BUFFERED);
    this.panel = panel;
    this.context = new RenderContext.Global(theme, this);

    addListener(SWT.Paint, e -> {
      long start = System.nanoTime();
      e.gc.setBackground(getDisplay().getSystemColor(colors().background));
      Rectangle size = e.gc.getClipping();
      e.gc.fillRectangle(size);
      Map<String, Long> traces;
      try (RenderContext ctx = context.newContext(e.gc)) {
        panel.render(ctx, a -> scheduleIfNotDisposed(this, () -> redraw(a, false)));
        ctx.renderOverlays();
        traces = ctx.getTraces();
      }
      long end = System.nanoTime();
      if (LOG.isLoggable(Level.FINE)) {
        LOG.log(Level.FINE, size + " (" + (end - start) / 1000000.0 + ") " + traces);
      }

      if (showRedraws.get()) {
        size.width--;
        size.height--;
        e.gc.setForeground(getDisplay().getSystemColor(SWT.COLOR_RED));
        e.gc.drawRectangle(size);
      }
    });
    addListener(SWT.Resize, e -> {
      Rectangle size = getClientArea();
      panel.setSize(size.width, size.height);
      redraw(new Area(0, 0, size.width, size.height), false);
    });
    addListener(SWT.MouseDown, e -> {
      if (e.button == 1) {
        mouseDown = new Point(e.x, e.y);
        dragging = false;
      }
    });
    addListener(SWT.MouseMove, e -> {
      updateMousePosition(e.x, e.y, e.stateMask, false, true);
    });
    addListener(SWT.MouseUp, e -> {
      mouseDown = null;
      // Reset and re-evaluate selections each time mouse is released, to allow easy deselection.
      long oldCpuUtid = state.getSelectedUtid();
      Map<Panel, Set<Location>> oldMarked = state.getMarkLocations();
      state.resetSelections();
      if (dragging) {
        dragging = false;
        setCursor(null);
        redraw(dragger.onDragEnd(e.x, e.y), false);
        dragger = Panel.Dragger.NONE;
        updateMousePosition(e.x, e.y, 0, false, true);
      } else {
        boolean shouldRedraw = hover.click();
        shouldRedraw = shouldRedraw || state.shouldChangeCpuSlicesColor(oldCpuUtid)
            || state.shouldRemoveMarks(oldMarked);
        if (shouldRedraw) {
          structureHasChanged();
        }
      }
    });
    addListener(SWT.MouseExit, e -> {
      lastMouse.x = lastMouse.y = -1;
      mouseDown = null;
      dragging = false;
      Area old = hover.getRedraw();
      hover.stop();
      hover = Panel.Hover.NONE;
      redraw(old, false);
      setCursor(null);
    });
    addListener(SWT.Dispose, e -> {
      context.dispose();
    });
  }

  private void updateMousePosition(int x, int y, int mods, boolean force, boolean redraw) {
    if (force || x != lastMouse.x || y != lastMouse.y) {
      if (mouseDown != null) {
        if (!dragging) {
          dragger = panel.onDragStart(mouseDown.x, mouseDown.y, mods);
          dragging = true;
        }
        setCursor(dragger.getCursor(getDisplay()));
        if (redraw) {
          redraw(dragger.onDrag(x, y), false);
        }
      }

      lastMouse.x = x;
      lastMouse.y = y;

      Area old = hover.getRedraw();
      hover.stop();
      if (x < 0 || y < 0) {
        if (!dragging) {
          setCursor(null);
        }
        return;
      }
      hover = panel.onMouseMove(context, x, y);
      if (!dragging) {
        setCursor(hover.getCursor(getDisplay()));
      }
      if (redraw) {
        redraw(hover.getRedraw().combine(old), false);
      }
    }
  }

  public void structureHasChanged() {
    Rectangle size = getClientArea();
    panel.setSize(size.width, size.height);
    redraw(Area.FULL, false);
  }

  public void redraw(Area area, boolean refreshMouse) {
    if (!area.isEmpty()) {
      if (refreshMouse) {
        updateMousePosition(lastMouse.x, lastMouse.y, 0, true, false);
      }

      if (hover.isOverlay()) {
        Area hoverArea = hover.getRedraw();
        if (area.intersects(hoverArea)) {
          area = area.combine(hoverArea);
        }
      }

      Rectangle size = getClientArea();
      if (area != Area.FULL) {
        size.x = Math.max(0, Math.min(size.width - 1, (int)Math.floor(area.x)));
        size.y = Math.max(0, Math.min(size.height - 1, (int)Math.floor(area.y)));
        size.width = Math.max(0, Math.min(size.width - size.x, (int)Math.ceil(area.w + (area.x - size.x))));
        size.height = Math.max(0, Math.min(size.height - size.y, (int)Math.ceil(area.h + (area.y - size.y))));
      }
      redraw(size.x, size.y, size.width, size.height, false);
    }
  }

  @Override
  public Point computeSize(int wHint, int hHint, boolean changed) {
    return new Point(wHint, (int)Math.ceil(panel.getPreferredHeight()));
  }
}
