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

import org.eclipse.swt.graphics.Color;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL20;

import java.nio.IntBuffer;
import java.util.Arrays;
import java.util.Map;
import java.util.logging.Logger;

/**
 * An OpenGL shader program.
 */
public class Shader extends GlObject {
  protected static Logger LOG = Logger.getLogger(Shader.class.getName());

  private final int handle;
  private final Map<String, Attribute> attributes = Maps.newHashMap();
  private final Map<String, Uniform> uniforms = Maps.newHashMap();

  Shader(Renderer owner) {
    super(owner);
    this.handle = GL20.glCreateProgram();
    owner.register(this);
  }

  public void setUniform(String name, Object value) {
    Uniform uniform = uniforms.get(name);
    if (uniform != null) {
      uniform.set(value);
    }
  }

  public void setAttribute(String name, float x, float y, float z) {
    Attribute attribute = attributes.get(name);
    if (attribute != null) {
      attribute.set(x, y, z);
    }
  }

  public void setAttribute(String name, VertexBuffer vertexBuffer) {
    Attribute attribute = attributes.get(name);
    if (attribute != null) {
      attribute.set(vertexBuffer);
    }
  }

  @Override
  protected void release() {
    detachShaders();
    GL20.glDeleteProgram(handle);
    attributes.clear();
    uniforms.clear();
  }

  boolean link(String vertexSource, String fragmentSource) {
    detachShaders();
    if (!attachShaders(vertexSource, fragmentSource) || !link()) {
      return false;
    }
    getAttributes();
    getUniforms();
    return true;
  }

  void bind() {
    GL20.glUseProgram(handle);
    for (Attribute attribute : attributes.values()) {
      attribute.bind();
    }
    for (Uniform uniform : uniforms.values()) {
      uniform.bind();
    }
  }

  void unbind() {
    for (Attribute attribute : attributes.values()) {
      attribute.unbind();
    }
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
    private float vecX, vecY, vecZ;
    private VertexBuffer vertexBuffer;

    public Attribute(int location, String name, int type) {
      super(location, name, type);
    }

    public void set(VertexBuffer vertexBuffer) {
      this.vertexBuffer = vertexBuffer;
    }

    public void set(float x, float y, float z) {
      vecX = x;
      vecY = y;
      vecZ = z;
    }

    public void bind() {
      if (vertexBuffer != null) {
        vertexBuffer.bind();
        GL20.glEnableVertexAttribArray(location);
        GL20.glVertexAttribPointer(
            location, vertexBuffer.elementsPerVertex, vertexBuffer.elementType, false, 0, 0);
      } else {
        GL20.glDisableVertexAttribArray(location);
        GL20.glVertexAttrib3f(location, vecX, vecY, vecZ);
      }
    }

    public void unbind() {
      GL20.glDisableVertexAttribArray(location);
    }
  }

  private static class Uniform extends AttributeOrUniform {
    private final Binder binder;
    private Object value;

    public Uniform(int location, String name, int type) {
      super(location, name, type);
      this.binder = getBinder();
    }

    public void set(Object value) {
      this.value = value;
    }

    public boolean bind() {
      if (value instanceof Float) {
        binder.bind(((Float)value).floatValue());
      } else if (value instanceof Integer) {
        binder.bind(((Integer)value).intValue());
      } else if (value instanceof int[]) {
        binder.bind((int[])value);
      } else if (value instanceof float[]) {
        binder.bind((float[])value);
      } else if (value instanceof Color){
        Color color = (Color)value;
        binder.bind(new float[]{
            color.getRed()   / 255.f,
            color.getGreen() / 255.f,
            color.getBlue()  / 255.f,
            color.getAlpha() / 255.f,
        });
      } else if (value instanceof Texture) {
        int unit = 0; // TODO use incrementing allocation.
        Texture texture = (Texture)value;
        Texture.activate(unit);
        texture.bind();
        binder.bind(unit);
      } else {
        return false;
      }
      return true;
    }

    private Binder getBinder() {
      switch (type) {
        case GL11.GL_SHORT:
        case GL11.GL_UNSIGNED_INT:
        case GL11.GL_FLOAT:
        case GL11.GL_INT:
        case GL20.GL_BOOL:
        case GL20.GL_SAMPLER_2D:
        case GL20.GL_SAMPLER_CUBE:
          return new Binder() {
            @Override
            public void bind(float[] vals) {
              GL20.glUniform1fv(location, vals);
            }

            @Override
            public void bind(float val) {
              GL20.glUniform1f(location, val);
            }

            @Override
            public void bind(int[] vals) {
              GL20.glUniform1iv(location, vals);
            }

            @Override
            public void bind(int val) {
              GL20.glUniform1i(location, val);
            }
          };
        case GL20.GL_INT_VEC2:
        case GL20.GL_BOOL_VEC2:
        case GL20.GL_FLOAT_VEC2:
          return new Binder() {
            @Override
            public void bind(float[] vals) {
              GL20.glUniform2fv(location, vals);
            }

            @Override
            public void bind(float val) {
              GL20.glUniform2f(location, val, 0);
            }

            @Override
            public void bind(int[] vals) {
              GL20.glUniform2iv(location, vals);
            }

            @Override
            public void bind(int val) {
              GL20.glUniform2i(location, val, 0);
            }
          };
        case GL20.GL_INT_VEC3:
        case GL20.GL_BOOL_VEC3:
        case GL20.GL_FLOAT_VEC3:
          return new Binder() {
            @Override
            public void bind(float[] vals) {
              GL20.glUniform3fv(location, vals);
            }

            @Override
            public void bind(float val) {
              GL20.glUniform3f(location, val, 0, 0);
            }

            @Override
            public void bind(int[] vals) {
              GL20.glUniform3iv(location, vals);
            }

            @Override
            public void bind(int val) {
              GL20.glUniform3i(location, val, 0, 0);
            }
          };
        case GL20.GL_INT_VEC4:
        case GL20.GL_BOOL_VEC4:
        case GL20.GL_FLOAT_VEC4:
          return new Binder() {
            @Override
            public void bind(float[] vals) {
              GL20.glUniform4fv(location, vals);
            }

            @Override
            public void bind(float val) {
              GL20.glUniform4f(location, val, 0, 0, 1);
            }

            @Override
            public void bind(int[] vals) {
              GL20.glUniform4iv(location, vals);
            }

            @Override
            public void bind(int val) {
              GL20.glUniform4i(location, val, 0, 0, 1);
            }
          };
        case GL20.GL_FLOAT_MAT2:
          return new Binder() {
            @Override
            public void bind(float[] vals) {
              GL20.glUniformMatrix2fv(location, false, vals);
            }

            @Override
            public void bind(float val) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat2): " + val);
            }

            @Override
            public void bind(int[] vals) {
              LOG.log(WARNING,
                  "Unexpected shader uniform value (expected mat2): " + Arrays.toString(vals));
            }

            @Override
            public void bind(int val) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat2): " + val);
            }
          };
        case GL20.GL_FLOAT_MAT3:
          return new Binder() {
            @Override
            public void bind(float[] vals) {
              GL20.glUniformMatrix3fv(location, false, vals);
            }

            @Override
            public void bind(float val) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat3): " + val);
            }

            @Override
            public void bind(int[] vals) {
              LOG.log(WARNING,
                  "Unexpected shader uniform value (expected mat3): " + Arrays.toString(vals));
            }

            @Override
            public void bind(int val) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat3): " + val);
            }
          };
        case GL20.GL_FLOAT_MAT4:
          return new Binder() {
            @Override
            public void bind(float[] vals) {
              GL20.glUniformMatrix4fv(location, false, vals);
            }

            @Override
            public void bind(float val) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat4): " + val);
            }

            @Override
            public void bind(int[] vals) {
              LOG.log(WARNING,
                  "Unexpected shader uniform value (expected mat4): " + Arrays.toString(vals));
            }

            @Override
            public void bind(int val) {
              LOG.log(WARNING, "Unexpected shader uniform value (expected mat4): " + val);
            }
          };
        default:
          LOG.log(WARNING, "Unexpected shader uniform type: " + type);
          throw new AssertionError();
      }
    }

    private interface Binder {
      public void bind(int value);
      public void bind(int[] values);
      public void bind(float value);
      public void bind(float[] values);
    }
  }
}
