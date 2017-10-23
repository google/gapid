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
package com.google.gapid.widgets;

import static com.google.gapid.widgets.Widgets.redrawIfNotDisposed;

import com.google.common.collect.Sets;
import com.google.gapid.util.Loadable;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Display;

import java.util.Set;

/**
 * Widget to draw an animated loading indicator.
 */
public class LoadingIndicator {
  private static final int FRAME_COUNT = 8;
  private static final int CYCLE_LENGTH = 1000;
  private static final int MS_PER_FRAME = CYCLE_LENGTH / FRAME_COUNT;
  private static final int LARGE_SIZE = 32;
  private static final int SIZE_THRESHOLD = 3 * LARGE_SIZE / 2;
  private static final int SMALL_SIZE = 16;

  private final Display display;
  private final Image[] icons;
  private final Image[] smallIcons;
  private final Image refresh;
  private final Set<Repaintable> componentsToRedraw = Sets.newIdentityHashSet();

  public LoadingIndicator(Display display, Theme theme) {
    this.display = display;
    this.icons = theme.loadingLarge();
    this.smallIcons = theme.loadingSmall();
    this.refresh = theme.refresh();
  }

  public void paint(GC g, int x, int y, Point size) {
    paint(g, x, y, size.x, size.y);
  }

  public void paint(GC g, int x, int y, int w, int h) {
    Image image = (Math.min(w, h) < SIZE_THRESHOLD) ? getCurrentSmallFrame() : getCurrentFrame();
    paint(image, g, x, y, w, h);
  }

  public void paintRefresh(GC g, int x, int y, Point size) {
    paintRefresh(g, x, y, size.x, size.y);
  }

  public void paintRefresh(GC g, int x, int y, int w, int h) {
    paint(refresh, g, x, y, w, h);
  }

  private static void paint(Image image, GC g, int x, int y, int w, int h) {
    Rectangle s = image.getBounds();
    g.drawImage(image, 0, 0, s.width, s.height,
        x + (w - s.width) / 2, y + (h - s.height) / 2, s.width, s.height);
  }

  public Image getCurrentFrame() {
    long elapsed = System.currentTimeMillis() % CYCLE_LENGTH;
    return icons[(int)((elapsed * icons.length) / CYCLE_LENGTH)];
  }

  public Image getCurrentSmallFrame() {
    long elapsed = System.currentTimeMillis() % CYCLE_LENGTH;
    return smallIcons[(int)((elapsed * smallIcons.length) / CYCLE_LENGTH)];
  }

  public void scheduleForRedraw(Repaintable c) {
    synchronized (componentsToRedraw) {
      if (componentsToRedraw.add(c) && componentsToRedraw.size() == 1) {
        display.timerExec(MS_PER_FRAME, () -> {
          // Don't starve async runnables just for the animation.
          display.asyncExec(this::redrawAll);
        });
      }
    }
  }

  public void cancelRedraw(Repaintable c) {
    synchronized (componentsToRedraw) {
      componentsToRedraw.remove(c);
    }
  }

  private void redrawAll() {
    Repaintable[] components;
    synchronized (componentsToRedraw) {
      components = componentsToRedraw.toArray(new Repaintable[componentsToRedraw.size()]);
      componentsToRedraw.clear();
    }
    for (Repaintable c : components) {
      c.repaint();
    }
  }

  public Widget createWidget(Composite parent) {
    return new Widget(parent, false);
  }

  public Widget createWidgetWithRefresh(Composite parent) {
    return new Widget(parent, true);
  }

  /**
   * Object containing the loading indicator that needs to be animated.
   */
  public interface Repaintable {
    /**
     * Repaints the widget, potentially rendering the next frame in the loading animation.
     */
    public void repaint();
  }

  /**
   * Widget that shows the loading indicator while loading and is blank once done.
   */
  public class Widget extends Canvas implements Loadable, Repaintable {
    private boolean loading = false;

    public Widget(Composite parent, boolean showRefresh) {
      super(parent, SWT.DOUBLE_BUFFERED);
      addListener(SWT.Paint, e -> {
        if (loading) {
          paint(e.gc, 0, 0, getSize());
          scheduleForRedraw(this);
        } else if (showRefresh) {
          paintRefresh(e.gc, 0, 0, getSize());
        }
      });
    }

    @Override
    public void startLoading() {
      loading = true;
      scheduleForRedraw(this);
    }

    @Override
    public void stopLoading() {
      loading = false;
      scheduleForRedraw(this);
    }

    @Override
    public void repaint() {
      redrawIfNotDisposed(this);
    }

    @Override
    public Point computeSize(int wHint, int hHint, boolean changed) {
      return new Point(SMALL_SIZE, SMALL_SIZE);
    }
  }
}
