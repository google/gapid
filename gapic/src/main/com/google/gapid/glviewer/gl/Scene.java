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
package com.google.gapid.glviewer.gl;

/**
 * The interface implemented by classes that render 3D scenes.
 *
 * @param <T> the immutable scene data.
 */
public interface Scene<T> {
  /** Called before any other methods to initialize the scene. */
  public void init(Renderer renderer);

  /** Updates the scene with new data. */
  public void update(Renderer renderer, T data);

  /** Renders the scene. */
  public void render(Renderer renderer);

  /**
   * Called when the back-buffer has been resized.
   * TODO: Move as a listener on the renderer?
   */
  public void resize(Renderer renderer, int width, int height);
}
