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

import com.google.gapid.glviewer.gl.Shader;

public interface Renderable {
  public static final Renderable NOOP = new Renderable() {
    @Override
    public void init() {
      // No op.
    }

    @Override
    public void render(State state) {
      // No op.
    }

    @Override
    public void dispose() {
      // No op.
    }
  };

  public void init();
  public void render(State state);
  public void dispose();

  public static class State {
    public final Shader shader;
    public final ModelViewProjection transform;

    public State(Shader shader, boolean invertNormals) {
      this.shader = shader;
      this.transform = new ModelViewProjection(invertNormals);
    }
  }
}
