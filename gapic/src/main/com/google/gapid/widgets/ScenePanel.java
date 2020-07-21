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

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.gapid.glcanvas.GlCanvas;
import com.google.gapid.glviewer.gl.Renderer;
import com.google.gapid.glviewer.gl.Scene;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Listener;
import org.lwjgl.opengl.GL;
import org.lwjgl.opengl.GLCapabilities;

import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;

/**
 * {@link GlCanvas} used to render a {@link Scene}.
 */
public class ScenePanel<T> extends GlCanvas {
  private final GLCapabilities caps;
  private final Renderer renderer;
  private final Scene<T> scene;
  private final AtomicReference<T> sceneData = new AtomicReference<>();
  private Map<Integer, List<Listener>> eventListeners;
  private int numSuspendedUpdates;
  private boolean isPendingUpdate;
  private boolean isPendingRender;

  public ScenePanel(Composite parent, Scene<T> scene) {
    super(parent, SWT.NO_BACKGROUND);
    setLayout(new FillLayout(SWT.VERTICAL));
    this.scene = scene;
    renderer = new Renderer();

    if (!isOpenGL()) {
      caps = null;
      return;
    }

    setCurrent();
    caps = GL.createCapabilities();
    initialize();

    addListener(SWT.Resize, e -> { /* register to handle resizes in dispatchEvents */ });
    addListener(SWT.Paint, e -> update(true));
  }

  // Intercept addListener calls so that we can reduce the number of updates per event to one,
  // regardless of the number of listeners that request renders.
  @Override
  public void addListener(int eventType, Listener listener) {
    if (!isOpenGL()) {
      super.addListener(eventType, listener);
      return;
    }

    if (eventListeners == null) {
      eventListeners = Maps.newHashMap();
    }
    List<Listener> existing = eventListeners.get(eventType);
    if (existing != null) {
      existing.add(listener);
      return;
    }
    List<Listener> list = Lists.newArrayList();
    list.add(listener);
    super.addListener(eventType, this::dispatchEvents);
    eventListeners.put(eventType, list);
  }

  /**
   * Request the scene to be redrawn.
   */
  public void paint() {
    notifyListeners(SWT.Paint, new Event());
  }

  /**
   * Change the scene data and redraw.
   */
  public void setSceneData(T data) {
    sceneData.set(data);
    update(true);
  }

  @Override
  protected void terminate() {
    if (!isOpenGL()) {
      return;
    }

    setCurrent();
    GL.setCapabilities(caps);
    renderer.terminate();
  }

  private void initialize() {
    renderer.initialize();
    Point size = getSize();
    renderer.setSize(size.x, size.y, DPIUtil.getDeviceZoom() / 100f);
    scene.init(renderer);
  }

  private void dispatchEvents(Event event) {
    withSuspendedUpdate(() -> {
      if (event.type == SWT.Resize && isOpenGL()) {
        Point size = getSize();
        setCurrent();
        renderer.setSize(size.x, size.y, DPIUtil.getDeviceZoom() / 100f);
        scene.resize(renderer, size.x, size.y);
      }
      for (Listener listener : eventListeners.get(event.type)) {
        listener.handleEvent(event);
      }
    });
  }

  private void withSuspendedUpdate(Runnable r) {
    try {
      numSuspendedUpdates++;
      r.run();
    } finally {
      numSuspendedUpdates--;
    }
    if (numSuspendedUpdates == 0 && isPendingUpdate) {
      update(isPendingRender);
      isPendingUpdate = false;
      isPendingRender = false;
    }
  }

  private void update(boolean render) {
    if (numSuspendedUpdates > 0) {
      isPendingRender |= render;
      isPendingUpdate = true;
      return;
    }

    if (isOpenGL()) {
      setCurrent();
      GL.setCapabilities(caps);
      T newData = sceneData.getAndSet(null);
      if (newData != null) {
        scene.update(renderer, newData);
      }
      if (render) {
        scene.render(renderer);
        swapBuffers();
      }
    }
  }
}
