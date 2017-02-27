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

import static java.util.logging.Level.WARNING;
import static org.lwjgl.BufferUtils.createIntBuffer;

import com.google.common.collect.Maps;

import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL20;

import java.nio.IntBuffer;
import java.util.Arrays;
import java.util.Map;
import java.util.logging.Logger;

/**
 * Helper object for GL shaders and programs.
 */
public class Shader {
  protected static Logger LOG = Logger.getLogger(Shader.class.getName());

  private final int handle;
  private final Map<String, Attribute> attributes = Maps.newHashMap();
  private final Map<String, Uniform> uniforms = Maps.newHashMap();

  public Shader() {
    this.handle = GL20.glCreateProgram();
  }

  public boolean link(String vertexSource, String fragmentSource) {
    detachShaders();
    if (!attachShaders(vertexSource, fragmentSource) || !link()) {
      return false;
    }
    getAttributes();
    getUniforms();
    return true;
  }

  public void bind() {
    GL20.glUseProgram(handle);
  }

  /**
   * Allowed types are Float, Integer, int[] and float[].
   */
  public void setUniform(String name, Object value) {
    Uniform uniform = uniforms.get(name);
    if (uniform != null && !uniform.set(value)) {
      LOG.log(WARNING,
          "Unexpected uniform value: " + value + " (" + value.getClass() + ") for " + name);
    }
  }

  public void setAttribute(String name, float x, float y, float z) {
    Attribute attribute = attributes.get(name);
    if (attribute != null) {
      attribute.set(x, y, z);
    }
  }

  public void bindAttribute(
      String name, int elementSize, int elementType, int strideBytes, int offsetBytes) {
    Attribute attribute = attributes.get(name);
    if (attribute != null) {
      attribute.bind(elementSize, elementType, strideBytes, offsetBytes);
    }
  }

  public void unbindAttribute(String name) {
    Attribute attribute = attributes.get(name);
    if (attribute != null) {
      attribute.unbind();
    }
  }

  public void delete() {
    detachShaders();
    GL20.glDeleteProgram(handle);
    attributes.clear();
    uniforms.clear();
  }

  private void detachShaders() {
    int[] shaders = getAttachedShaders(handle);
    for (int i = 0; i < shaders.length; i++) {
      GL20.glDetachShader(handle, shaders[i]);
      GL20.glDeleteShader(shaders[i]);
    }
  }

  private boolean attachShaders(String vertexSource, String fragmentSource) {
    int vertexShader = createShader(GL20.GL_VERTEX_SHADER, vertexSource);
    if (vertexShader < 0) {
      return false;
    }

    int fragmentShader = createShader(GL20.GL_FRAGMENT_SHADER, fragmentSource);
    if (fragmentShader < 0) {
      GL20.glDeleteShader(vertexShader);
      return false;
    }

    GL20.glAttachShader(handle, vertexShader);
    GL20.glAttachShader(handle, fragmentShader);
    return true;
  }

  private boolean link() {
    GL20.glLinkProgram(handle);
    if (GL20.glGetProgrami(handle, GL20.GL_LINK_STATUS) != GL11.GL_TRUE) {
      LOG.log(WARNING, "Failed to link program:\n" + GL20.glGetProgramInfoLog(handle));
      return false;
    }
    return true;
  }

  private void getAttributes() {
    attributes.clear();
    for (Attribute attribute : getActiveAttributes(handle)) {
      attributes.put(attribute.name, attribute);
    }
  }

  private void getUniforms() {
    uniforms.clear();
    for (Uniform uniform : getActiveUniforms(handle)) {
      uniforms.put(uniform.name, uniform);
    }
  }

  private static int createShader(int type, String source) {
    int shader = GL20.glCreateShader(type);
    GL20.glShaderSource(shader, source);
    GL20.glCompileShader(shader);
    if (GL20.glGetShaderi(shader, GL20.GL_COMPILE_STATUS) != GL11.GL_TRUE) {
      LOG.log(WARNING, "Failed to compile shader:\n" + GL20.glGetShaderInfoLog(shader) +
          "\n\nSource:\n" + source);
      GL20.glDeleteShader(shader);
      return -1;
    }
    return shader;
  }

  private static int[] getAttachedShaders(int program) {
    int numShaders = GL20.glGetProgrami(program, GL20.GL_ATTACHED_SHADERS);
    if (numShaders > 0) {
      int[] shaders = new int[numShaders], count = new int[1];
      GL20.glGetAttachedShaders(program, count, shaders);
      return shaders;
    }
    return new int[0];
  }

  private static Attribute[] getActiveAttributes(int program) {
    int maxAttribNameLength = GL20.glGetProgrami(program, GL20.GL_ACTIVE_ATTRIBUTE_MAX_LENGTH);
    int numAttributes = GL20.glGetProgrami(program, GL20.GL_ACTIVE_ATTRIBUTES);
    IntBuffer size = createIntBuffer(1), type = createIntBuffer(1);

    Attribute[] result = new Attribute[numAttributes];
    for (int i = 0; i < numAttributes; i++) {
      String name = GL20.glGetActiveAttrib(program, i, maxAttribNameLength, size, type);
      result[i] = new Attribute(GL20.glGetAttribLocation(program, name), name, type.get(0));
    }
    return result;
  }

  private static Uniform[] getActiveUniforms(int program) {
    int maxUniformNameLength = GL20.glGetProgrami(program, GL20.GL_ACTIVE_UNIFORM_MAX_LENGTH);
    int numUniforms = GL20.glGetProgrami(program, GL20.GL_ACTIVE_UNIFORMS);
    IntBuffer size = createIntBuffer(1), type = createIntBuffer(1);

    Uniform[] result = new Uniform[numUniforms];
    for (int i = 0; i < numUniforms; i++) {
      String name = GL20.glGetActiveUniform(program, i, maxUniformNameLength, size, type);
      if (name.endsWith("[0]")) {
        name = name.substring(0, name.length() - 3);
      }
      result[i] = new Uniform(GL20.glGetUniformLocation(program, name), name, type.get(0));
    }
    return result;
  }

  private static class AttributeOrUniform {
    public final int location;
    public final String name;
    public final int type;

    public AttributeOrUniform(int location, String name, int type) {
      this.location = location;
      this.name = name;
      this.type = type;
    }
  }

  private static class Attribute extends AttributeOrUniform {
    public Attribute(int location, String name, int type) {
      super(location, name, type);
    }

    public void set(float x, float y, float z) {
      GL20.glDisableVertexAttribArray(location);
      GL20.glVertexAttrib3f(location, x, y, z);
    }

    public void bind(int elementSize, int elementType, int strideBytes, int offsetBytes) {
      GL20.glEnableVertexAttribArray(location);
      GL20.glVertexAttribPointer(
          location, elementSize, elementType, false, strideBytes, offsetBytes);
    }

    public void unbind() {
      GL20.glDisableVertexAttribArray(location);
    }
  }

  private static class Uniform extends AttributeOrUniform {
    private final Setter setter;

    public Uniform(int location, String name, int type) {
      super(location, name, type);
      this.setter = getSetter();
    }

    private Setter getSetter() {
      switch (type) {
        case GL11.GL_SHORT:
        case GL11.GL_UNSIGNED_INT:
        case GL11.GL_FLOAT:
        case GL11.GL_INT:
        case GL20.GL_BOOL:
        case GL20.GL_SAMPLER_2D:
        case GL20.GL_SAMPLER_CUBE:
          return new Setter() {
            @Override
            public void set(float[] values) {
              GL20.glUniform1fv(location, values);
            }

            @Override
            public void set(float value) {
              GL20.glUniform1f(location, value);
            }

            @Override
            public void set(int[] values) {
              GL20.glUniform1iv(location, values);
            }

            @Override
            public void set(int value) {
              GL20.glUniform1i(location, value);
            }
          };
        case GL20.GL_INT_VEC2:
        case GL20.GL_BOOL_VEC2:
        case GL20.GL_FLOAT_VEC2:
          return new Setter() {
            @Override
            public void set(float[] values) {
              GL20.glUniform2fv(location, values);
            }

            @Override
            public void set(float value) {
              GL20.glUniform2f(location, value, 0);
            }

            @Override
            public void set(int[] values) {
              GL20.glUniform2iv(location, values);
            }

            @Override
            public void set(int value) {
              GL20.glUniform2i(location, value, 0);
            }
          };
        case GL20.GL_INT_VEC3:
        case GL20.GL_BOOL_VEC3:
        case GL20.GL_FLOAT_VEC3:
          return new Setter() {
            @Override
            public void set(float[] values) {
              GL20.glUniform3fv(location, values);
            }

            @Override
            public void set(float value) {
              GL20.glUniform3f(location, value, 0, 0);
            }

            @Override
            public void set(int[] values) {
              GL20.glUniform3iv(location, values);
            }

            @Override
            public void set(int value) {
              GL20.glUniform3i(location, value, 0, 0);
            }
          };
        case GL20.GL_INT_VEC4:
        case GL20.GL_BOOL_VEC4:
        case GL20.GL_FLOAT_VEC4:
          return new Setter() {
            @Override
            public void set(float[] values) {
              GL20.glUniform4fv(location, values);
            }

            @Override
            public void set(float value) {
              GL20.glUniform4f(location, value, 0, 0, 1);
            }

            @Override
            public void set(int[] values) {
              GL20.glUniform4iv(location, values);
            }

            @Override
            public void set(int value) {
              GL20.glUniform4i(location, value, 0, 0, 1);
            }
          };
        case GL20.GL_FLOAT_MAT2:
          return new Setter() {
            @Override
            public void set(float[] values) {
              GL20.glUniformMatrix2fv(location, false, values);
            }

            @Override
            public void set(float value) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat2): " + value);
            }

            @Override
            public void set(int[] values) {
              LOG.log(WARNING,
                  "Unexpected shader uniform value (expected mat2): " + Arrays.toString(values));
            }

            @Override
            public void set(int value) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat2): " + value);
            }
          };
        case GL20.GL_FLOAT_MAT3:
          return new Setter() {
            @Override
            public void set(float[] values) {
              GL20.glUniformMatrix3fv(location, false, values);
            }

            @Override
            public void set(float value) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat3): " + value);
            }

            @Override
            public void set(int[] values) {
              LOG.log(WARNING,
                  "Unexpected shader uniform value (expected mat3): " + Arrays.toString(values));
            }

            @Override
            public void set(int value) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat3): " + value);
            }
          };
        case GL20.GL_FLOAT_MAT4:
          return new Setter() {
            @Override
            public void set(float[] values) {
              GL20.glUniformMatrix4fv(location, false, values);
            }

            @Override
            public void set(float value) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat4): " + value);
            }

            @Override
            public void set(int[] values) {
              LOG.log(WARNING,
                  "Unexpected shader uniform value (expected mat4): " + Arrays.toString(values));
            }

            @Override
            public void set(int value) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat4): " + value);
            }
          };
        default:
          LOG.log(WARNING, "Unexpected shader uniform type: " + type);
          throw new AssertionError();
      }
    }

    public boolean set(Object value) {
      if (value instanceof Float) {
        setter.set(((Float)value).floatValue());
      } else if (value instanceof Integer) {
        setter.set(((Integer)value).intValue());
      } else if (value instanceof int[]) {
        setter.set((int[])value);
      } else if (value instanceof float[]) {
        setter.set((float[])value);
      } else {
        return false;
      }
      return true;
    }

    private interface Setter {
      public void set(int value);
      public void set(int[] values);
      public void set(float value);
      public void set(float[] values);
    }
  }
}
