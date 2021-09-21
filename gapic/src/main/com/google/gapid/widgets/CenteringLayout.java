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
package com.google.gapid.widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Layout;

/**
 * A {@link Layout} that centers a single child control
 */
public class CenteringLayout extends Layout {
  private final Mode mode;
  private int defaultWidth = -1, defaultHeight = -1;
  private int currentWidth = -1, currentHeight = -1;

  private CenteringLayout(Mode mode) {
    this.mode = mode;
  }

  public static CenteringLayout center() {
    return new CenteringLayout(Mode.Center);
  }

  public static CenteringLayout goldenRatio() {
    return new CenteringLayout(Mode.GoldenRatio);
  }

  @Override
  protected Point computeSize(Composite composite, int wHint, int hHint, boolean flushCache) {
    if (flushCache) {
      flushCache(null);
    }

    Control[] controls = composite.getChildren();
    if (controls.length == 0) {
      return new Point(wHint == SWT.DEFAULT ? 0 : wHint, hHint == SWT.DEFAULT ? 0 : hHint);
    }
    computeChildSize(controls[0], wHint, hHint, flushCache);
    return new Point(currentWidth, currentHeight);
  }

  @Override
  protected void layout(Composite composite, boolean flushCache) {
    if (flushCache) {
      flushCache(null);
    }

    Control[] controls = composite.getChildren();
    if (controls.length == 0) {
      return;
    }
    if (currentWidth == -1 || currentHeight == -1) {
      computeChildSize(controls[0], SWT.DEFAULT, SWT.DEFAULT, flushCache);
    }

    mode.setBounds(controls[0], composite.getClientArea(), currentWidth, currentHeight);
  }

  @Override
  protected boolean flushCache(Control control) {
    defaultWidth = defaultHeight = -1;
    currentWidth = currentHeight = -1;
    return true;
  }

  private void computeChildSize(Control control, int wHint, int hHint, boolean flushCache) {
    if (wHint == SWT.DEFAULT && hHint == SWT.DEFAULT) {
      if (defaultWidth == -1 || defaultHeight == -1) {
        Point size = control.computeSize(SWT.DEFAULT, SWT.DEFAULT, flushCache);
        defaultWidth = size.x;
        defaultHeight = size.y;
      }
      currentWidth = defaultWidth;
      currentHeight = defaultHeight;
    } else {
      Point size = control.computeSize(wHint, hHint, flushCache);
      if (wHint == SWT.DEFAULT) {
        currentWidth = size.x;
      } else {
        currentHeight = size.y;
      }
    }
  }

  public static enum Mode {
    Center() {
      @Override
      public void setBounds(Control c, Rectangle s, int w, int h) {
        c.setBounds((s.width - w) / 2, (s.height - h) / 2, w, h);
      }
    }, GoldenRatio() {
      private final double phi = 1 / (1 + Math.sqrt(5));

      @Override
      public void setBounds(Control c, Rectangle s, int w, int h) {
        c.setBounds((s.width - w) / 2, (int)((s.height - h) * phi), w, h);
      }
    };

    public abstract void setBounds(Control c, Rectangle s, int w, int h);
  }

}
