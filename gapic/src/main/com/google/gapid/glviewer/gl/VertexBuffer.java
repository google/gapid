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

import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL15;

/**
 * An OpenGL vertex buffer.
 */
public class VertexBuffer extends GlObject {
  final int handle;
  final int vertexCount;
  final int elementsPerVertex;
  final int elementType;

  VertexBuffer(Renderer owner, float[] data, int elementsPerVertex) {
    super(owner);
    this.handle = GL15.glGenBuffers();
    this.elementsPerVertex = elementsPerVertex;
    this.vertexCount = data.length / elementsPerVertex;
    this.elementType = GL11.GL_FLOAT;
    owner.register(this);

    bind();
    GL15.glBufferData(GL15.GL_ARRAY_BUFFER, data, GL15.GL_STATIC_DRAW);
  }

  void bind() {
    GL15.glBindBuffer(GL15.GL_ARRAY_BUFFER, handle);
  }

  @Override
  protected void release() {
    GL15.glDeleteBuffers(handle);
  }
}
