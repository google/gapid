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

import org.eclipse.swt.graphics.Color;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL13;
import org.lwjgl.opengl.GL33;

import java.nio.ByteBuffer;

/**
 * An OpenGL texture.
 */
public class Texture extends GlObject {
  private final int target;
  private final int handle;

  Texture(Renderer owner, int target) {
    super(owner);
    this.target = target;
    this.handle = GL11.glGenTextures();
    owner.register(this);
  }

  public Texture setMinMagFilter(int min, int mag) {
    bind();
    GL11.glTexParameteri(target, GL11.GL_TEXTURE_MIN_FILTER, min);
    GL11.glTexParameteri(target, GL11.GL_TEXTURE_MAG_FILTER, mag);
    return this;
  }

  public Texture setBorderColor(Color color) {
    bind();
    GL11.glTexParameterfv(target, GL11.GL_TEXTURE_BORDER_COLOR, new float[]{
        color.getRed() / 255.f,
        color.getGreen() / 255.f,
        color.getBlue() / 255.f,
        color.getAlpha() / 255.f,
    });
    return this;
  }

  public Texture setWrapMode(int wrapS, int wrapT) {
    bind();
    GL11.glTexParameteri(target, GL11.GL_TEXTURE_WRAP_S, wrapS);
    GL11.glTexParameteri(target, GL11.GL_TEXTURE_WRAP_T, wrapT);
    return this;
  }

  public Texture setSwizzle(int r, int g, int b, int a) {
    bind();
    GL11.glTexParameteriv(target, GL33.GL_TEXTURE_SWIZZLE_RGBA, new int[] { r, g, b, a });
    return this;
  }

  public Texture loadData(
      int width, int height, int internalFormat, int format, int type, ByteBuffer data) {
    bind();
    GL11.glPixelStorei(GL11.GL_UNPACK_ALIGNMENT, 1);
    GL11.glTexImage2D(target, 0, internalFormat, width, height, 0, format, type, data);
    return this;
  }

  static void activate(int unit) {
    GL13.glActiveTexture(GL13.GL_TEXTURE0 + unit);
  }

  void bind() {
    GL11.glBindTexture(target, handle);
  }

  @Override
  protected void release() {
    GL11.glDeleteTextures(handle);
  }
}
