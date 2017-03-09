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

import org.lwjgl.opengl.GL15;

/**
 * Helper object for GL buffers.
 */
public class Buffer {
  private final int target;
  private final int handle;
  private int size;

  public Buffer(int target) {
    this.target = target;
    this.handle = GL15.glGenBuffers();
  }

  public Buffer bind() {
    GL15.glBindBuffer(target, handle);
    return this;
  }

  public Buffer loadData(float[] data) {
    this.size = data.length * 4;
    GL15.glBufferData(target, data, GL15.GL_STATIC_DRAW);
    return this;
  }

  public Buffer loadData(int[] data) {
    this.size = data.length * 4;
    GL15.glBufferData(target, data, GL15.GL_STATIC_DRAW);
    return this;
  }

  public int getSize() {
    return size;
  }

  public void delete() {
    GL15.glDeleteBuffers(handle);
  }
}
