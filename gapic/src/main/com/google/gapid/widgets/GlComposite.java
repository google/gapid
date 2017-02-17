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

import com.google.gapid.glcanvas.GlCanvas;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.lwjgl.opengl.GL;
import org.lwjgl.opengl.GLCapabilities;

public class GlComposite extends Composite {
  private final GlCanvas canvas;
  private final GLCapabilities caps;

  public GlComposite(Composite parent) {
    super(parent, SWT.NO_BACKGROUND);
    setLayout(new FillLayout(SWT.VERTICAL));
    canvas = new GlCanvas(this, SWT.NO_BACKGROUND);
    canvas.setCurrent();
    caps = GL.createCapabilities();
  }

  public void addListener(Listener listener) {
    canvas.addListener(SWT.Resize, e -> {
      render(() -> {
        Point size = getSize();
        listener.reshape(0, 0, size.x, size.y);
        listener.display();
      }, true);
    });
    canvas.addListener(SWT.Paint, e -> {
      render(listener::display, true);
    });
    addListener(SWT.Dispose, e -> {
      render(listener::dispose, false);
    });

    render(() -> {
      listener.init();
      Point size = canvas.getSize();
      listener.reshape(0, 0, size.x, size.y);
      listener.display();
    }, true);
  }

  private void render(Runnable r, boolean swap) {
    canvas.setCurrent();
    GL.setCapabilities(caps);
    r.run();
    if (swap) {
      canvas.swapBuffers();
    }
  }

  public Control getControl() {
    return canvas;
  }

  public void paint() {
    canvas.notifyListeners(SWT.Paint, new Event());
  }

  public static interface Listener {
    public void init();
    public void reshape(int x, int y, int w, int h);
    public void display();
    public void dispose();
  }
}
