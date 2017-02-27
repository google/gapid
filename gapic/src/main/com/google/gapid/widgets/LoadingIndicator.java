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

import com.google.common.collect.Sets;

import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
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
  private static final int MIN_SIZE = 3 * LARGE_SIZE / 2;

  private final Display display;
  private final Image[] icons;
  private final Image[] smallIcons;
  private final Image error;
  private final Set<Repaintable> componentsToRedraw = Sets.newIdentityHashSet();

  public LoadingIndicator(Display display, Theme theme) {
    this.display = display;
    this.icons = new Image[] {
        theme.loading0large(), theme.loading1large(), theme.loading2large(), theme.loading3large(),
        theme.loading4large(), theme.loading5large(), theme.loading6large(), theme.loading7large()
    };
    this.smallIcons = new Image[] {
        theme.loading0small(), theme.loading1small(), theme.loading2small(), theme.loading3small(),
        theme.loading4small(), theme.loading5small(), theme.loading6small(), theme.loading7small()
    };
    this.error = theme.error();
  }

  public void paint(GC g, int x, int y, Point size) {
    paint(g, x, y, size.x, size.y);
  }

  public Image getCurrentFrame() {
    long elapsed = System.currentTimeMillis() % CYCLE_LENGTH;
    return icons[(int)((elapsed * icons.length) / CYCLE_LENGTH)];
  }

  public Image getCurrentSmallFrame() {
    long elapsed = System.currentTimeMillis() % CYCLE_LENGTH;
    return smallIcons[(int)((elapsed * smallIcons.length) / CYCLE_LENGTH)];
  }

  public Image getErrorImage() {
    return error;
  }

  public void paint(GC g, int x, int y, int w, int h) {
    Image image = (Math.min(w, h) < MIN_SIZE) ? getCurrentSmallFrame() : getCurrentFrame();
    Rectangle s = image.getBounds();
    g.drawImage(image, 0, 0, s.width, s.height,
        x + (w - s.width) / 2, y + (h - s.height) / 2, s.width, s.height);
  }

  public void scheduleForRedraw(Repaintable c) {
    synchronized (componentsToRedraw) {
      if (componentsToRedraw.add(c) && componentsToRedraw.size() == 1) {
        display.timerExec(MS_PER_FRAME, this::redrawAll);
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

  /**
   * Object containing the loading indicator that needs to be animated.
   */
  public interface Repaintable {
    /**
     * Repaints the widget, potentially rendering the next frame in the loading animation.
     */
    public void repaint();
  }
}
