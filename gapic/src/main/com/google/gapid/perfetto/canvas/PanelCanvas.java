package com.google.gapid.perfetto.canvas;

import static com.google.gapid.perfetto.views.StyleConstants.colors;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.gapid.widgets.Theme;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.ScrollBar;

import java.util.Map;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Draws a {@link Panel} and manages the user interactions.
 */
public class PanelCanvas extends Canvas {
  private static final Logger LOG = Logger.getLogger(PanelCanvas.class.getName());

  private final Panel panel;
  private final RenderContext.Global context;
  private int scrollOffset = 0;
  private double panelHeight = 0;
  private Point mouseDown = null;
  private boolean dragging = false;
  private Panel.Dragger dragger = Panel.Dragger.NONE;
  private Panel.Hover hover = Panel.Hover.NONE;
  private Point lastMouse = new Point(-1, -1);

  public PanelCanvas(Composite parent, int style, Theme theme, Panel panel) {
    super(parent, style | SWT.V_SCROLL | SWT.NO_BACKGROUND | SWT.DOUBLE_BUFFERED);
    this.panel = panel;
    this.context = new RenderContext.Global(theme, this);

    addListener(SWT.Paint, e -> {
      long start = System.nanoTime();
      e.gc.setBackground(getDisplay().getSystemColor(colors().background));
      Rectangle size = e.gc.getClipping();
      e.gc.fillRectangle(size);
      Map<String, Long> traces;
      try (RenderContext ctx = context.newContext(e.gc, -scrollOffset)) {
        panel.render(ctx,
            a -> scheduleIfNotDisposed(this, () -> redraw(a.translate(0, scrollOffset))));
        traces = ctx.getTraces();
      }
      long end = System.nanoTime();
      if (LOG.isLoggable(Level.FINE)) {
        LOG.log(Level.FINE, size + " (" + (end - start) / 1000000.0 + ") " + traces);
      }
    });
    addListener(SWT.Resize, e -> {
      Rectangle size = getClientArea();
      panelHeight = panel.getPreferredHeight();
      panel.setSize(size.width, panelHeight);
      updateScrollbars();
      redraw(new Area(0, 0, size.width, size.height));
    });
    addListener(SWT.MouseDown, e -> {
      if (e.button == 1) {
        mouseDown = new Point(e.x, e.y - scrollOffset);
        dragging = false;
      }
    });
    addListener(SWT.MouseMove, e -> {
      updateMousePosition(e.x, e.y - scrollOffset, e.stateMask, false, true);
    });
    addListener(SWT.MouseUp, e -> {
      mouseDown = null;
      if (dragging) {
        dragging = false;
        setCursor(null);
        redraw(dragger.onDragEnd(e.x, e.y - scrollOffset).translate(0, scrollOffset));
        dragger = Panel.Dragger.NONE;
        updateMousePosition(e.x, e.y - scrollOffset, 0, false, true);
      } else if (hover.click()) {
        structureHasChanged();
      }
    });
    addListener(SWT.MouseExit, e -> {
      lastMouse.x = lastMouse.y = -1;
      mouseDown = null;
      dragging = false;
      Area old = hover.getRedraw();
      hover.stop();
      hover = Panel.Hover.NONE;
      redraw(old.translate(0, scrollOffset));
      setCursor(null);
    });
    addListener(SWT.Dispose, e -> {
      context.dispose();
    });
    getVerticalBar().addListener(SWT.Selection, e -> {
      ScrollBar bar = getVerticalBar();
      if (bar.getVisible()) {
        int sel = bar.getSelection();
        updateMousePosition(lastMouse.x, lastMouse.y + sel + scrollOffset, 0, false, false);
        scrollOffset = -sel;
        Rectangle size = getClientArea();
        redraw(new Area(0, 0, size.width, size.height));
      }
    });
  }

  private void updateMousePosition(int x, int y, int mods, boolean force, boolean redraw) {
    if (force || x != lastMouse.x || y != lastMouse.y) {
      if (mouseDown != null) {
        if (!dragging) {
          dragger = panel.onDragStart(mouseDown.x, mouseDown.y, mods, -scrollOffset);
          dragging = true;
        }
        setCursor(dragger.getCursor(getDisplay()));
        redraw(dragger.onDrag(x, y).translate(0, scrollOffset));
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
      hover = panel.onMouseMove(context, x, y, -scrollOffset);
      if (!dragging) {
        setCursor(hover.getCursor(getDisplay()));
      }
      if (redraw) {
        redraw(hover.getRedraw().combine(old).translate(0, scrollOffset));
      }
    }
  }

  public void structureHasChanged() {
    panelHeight = panel.getPreferredHeight();
    updateScrollbars();

    Rectangle size = getClientArea();
    panel.setSize(size.width, panelHeight);
    updateMousePosition(lastMouse.x, lastMouse.y, 0, true, false);
    redraw(new Area(0, 0, size.width, size.height));
  }

  public void redraw(Area area) {
    if (!area.isEmpty()) {
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

  private void updateScrollbars() {
    Rectangle size = getClientArea();
    ScrollBar bar = getVerticalBar();
    int height = (int)Math.ceil(panelHeight);
    if (size.height < height) {
      if (bar.getVisible()) {
        int sel = Math.min(bar.getSelection(), height - size.height);
        bar.setValues(sel, 0, height, size.height, 10, 100);
        scrollOffset = -sel;
      } else {
        bar.setVisible(true);
        bar.setValues(0, 0, height, size.height, 10, 100);
        scrollOffset = 0;
      }
    } else if (bar.getVisible()) {
      bar.setVisible(false);
      scrollOffset = 0;
    }
  }

  @Override
  public Point computeSize(int wHint, int hHint, boolean changed) {
    return new Point(wHint, (int)Math.ceil(panel.getPreferredHeight()));
  }
}
