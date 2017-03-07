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

import static org.lwjgl.BufferUtils.createIntBuffer;

import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL15;
import org.lwjgl.opengl.GL20;
import org.lwjgl.opengl.GL30;

import java.nio.IntBuffer;

public class Util {
  private Util() {
  }

  public static boolean isAtLeastVersion(int major, int minor) {
    int val = GL11.glGetInteger(GL30.GL_MAJOR_VERSION);
    return (val > major) || (val == major && GL11.glGetInteger(GL30.GL_MINOR_VERSION) >= minor);
  }

  public static int createBuffer() {
    return GL15.glGenBuffers();
  }

  public static int createTexture() {
    return GL11.glGenTextures();
  }

  public static int getShaderiv(int shader, int name) {
    return GL20.glGetShaderi(shader, name);
  }

  public static int getProgramiv(int program, int name) {
    return GL20.glGetProgrami(program, name);
  }

  public static String getShaderInfoLog(int shader) {
    return GL20.glGetShaderInfoLog(shader);
  }

  public static String getProgramInfoLog(int program) {
    return GL20.glGetProgramInfoLog(program);
  }

  public static int[] getAttachedShaders(int program) {
    int numShaders = getProgramiv(program, GL20.GL_ATTACHED_SHADERS);
    if (numShaders > 0) {
      int[] shaders = new int[numShaders], count = new int[1];
      GL20.glGetAttachedShaders(program, count, shaders);
      return shaders;
    }
    return new int[0];
  }

  public static AttributeOrUniform[] getActiveAttributes(int program) {
    int maxAttribNameLength = getProgramiv(program, GL20.GL_ACTIVE_ATTRIBUTE_MAX_LENGTH);
    int numAttributes = getProgramiv(program, GL20.GL_ACTIVE_ATTRIBUTES);
    IntBuffer size = createIntBuffer(1), type = createIntBuffer(1);

    AttributeOrUniform[] result = new AttributeOrUniform[numAttributes];
    for (int i = 0; i < numAttributes; i++) {
      String name = GL20.glGetActiveAttrib(program, i, maxAttribNameLength, size, type);
      result[i] = new AttributeOrUniform(
          GL20.glGetAttribLocation(program, name), name, type.get(0), size.get(0));
    }
    return result;
  }

  public static AttributeOrUniform[] getActiveUniforms(int program) {
    int maxUniformNameLength = getProgramiv(program, GL20.GL_ACTIVE_UNIFORM_MAX_LENGTH);
    int numUniforms = getProgramiv(program, GL20.GL_ACTIVE_UNIFORMS);
    IntBuffer size = createIntBuffer(1), type = createIntBuffer(1);

    AttributeOrUniform[] result = new AttributeOrUniform[numUniforms];
    for (int i = 0; i < numUniforms; i++) {
      String name = GL20.glGetActiveUniform(program, i, maxUniformNameLength, size, type);
      if (name.endsWith("[0]")) {
        name = name.substring(0, name.length() - 3);
      }
      result[i] = new AttributeOrUniform(
          GL20.glGetUniformLocation(program, name), name, type.get(0), size.get(0));
    }
    return result;
  }

  public static class AttributeOrUniform {
    public final int location;
    public final String name;
    public final int type;
    public final int size;

    public AttributeOrUniform(int location, String name, int type, int size) {
      this.location = location;
      this.name = name;
      this.type = type;
      this.size = size;
    }
  }
}
