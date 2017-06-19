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
 * Base class for OpenGL objects owned by a {@link Renderer}.
 */
public abstract class GlObject {
  private final Renderer owner;

  private boolean deleted = false;

  GlObject(Renderer owner) {
    this.owner = owner;
  }

  /**
   * Frees the underlying object.
   * Once deleted the object should no longer be used.
   */
  public void delete() {
    if (!deleted) {
      owner.unregister(this);
      deleted = true;
      release();
    }
  }

  /** Delete the underlying OpenGL object */
  protected abstract void release();
}
