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
package com.google.gapid.glviewer;

import com.google.gapid.glviewer.gl.Renderer;
import com.google.gapid.glviewer.gl.Shader;

/**
 * Something to be rendered, e.g. a node in a "scene graph".
 */
public interface Renderable {
  public static final Renderable NOOP = new Renderable() {
    @Override
    public void init(Renderer renderer) {
      // No op.
    }

    @Override
    public void render(Renderer renderer, State state) {
      // No op.
    }

    @Override
    public void dispose(Renderer renderer) {
      // No op.
    }
  };

  /**
   * Called once to initialize this {@link Renderable} before any calls to
   * {@link #render(Renderer, State)}.
   */
  public void init(Renderer renderer);

  /**
   * Renders this object with the given state.
   */
  public void render(Renderer renderer, State state);

  /**
   * Called once to dispose this {@link Renderable} after the last call to
   * {@link #render(Renderer, State)}.
   */
  public void dispose(Renderer renderer);

  /**
   * Current render state, consisting of the current shader and view transforms.
   */
  public static class State {
    public final Shader shader;
    public final ModelViewProjection transform;

    public State(Shader shader, boolean invertNormals) {
      this.shader = shader;
      this.transform = new ModelViewProjection(invertNormals);
    }
  }
}
