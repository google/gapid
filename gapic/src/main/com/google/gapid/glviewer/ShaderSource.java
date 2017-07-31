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

import static java.util.logging.Level.WARNING;

import com.google.common.base.Charsets;
import com.google.common.io.LineProcessor;
import com.google.common.io.Resources;

import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL30;

import java.io.IOException;
import java.net.URL;
import java.util.logging.Logger;

/**
 * Utility to load shaders resources from the classpath. To simplify editing shaders, both the
 * vertex and fragment shaders of a program are stored in the same file. Special "comment markers"
 * are used to indicate which shader the following lines of code belong to. This utility also takes
 * care to insert the correct #version tag at the beginning of the shaders, allows sharing of code
 * between the shaders (using the "//! COMMON" marker) and simplifies declaration of vertex shader
 * output to fragment shader input by resurrecting the deprecated "varrying" keyword.
 *
 * @see
 * <a href="https://github.com/google/gapid/blob/master/docs/gapic-shaders.md">GAPIC Shaders</a>
 */
public class ShaderSource {
  private static final Logger LOG = Logger.getLogger(ShaderSource.class.getName());
  private static final String VERSION_130 = "#version 130\n";
  private static final String VERSION_150 = "#version 150\n";

  public final String vertex, fragment;

  public ShaderSource(String vertex, String fragment) {
    this.vertex = vertex;
    this.fragment = fragment;
  }

  public static ShaderSource load(String name) {
    return load(Resources.getResource("shaders/" + name + ".glsl"));
  }

  public static ShaderSource load(URL resource) {
    String version = isAtLeastVersion(3, 2) ? VERSION_150 : VERSION_130;
    try {
      ShaderSource source = Resources.readLines(resource, Charsets.US_ASCII,
          new LineProcessor<ShaderSource>() {
        private static final int MODE_COMMON = 0;
        private static final int MODE_VERTEX = 1;
        private static final int MODE_FRAGMENT = 2;
        private static final String VERTEX_PREAMBLE = "#define varying out\n";
        private static final String FRAGMENT_PREAMBLE = "#define varying in\n";

        private final StringBuilder vertexSource = new StringBuilder(version + VERTEX_PREAMBLE);
        private final StringBuilder fragmentSource = new StringBuilder(version + FRAGMENT_PREAMBLE);

        private int mode = MODE_COMMON;

        @Override
        public boolean processLine(String line) throws IOException {
          line = line.trim();
          if ("//! COMMON".equals(line)) {
            mode = MODE_COMMON;
          } else if ("//! VERTEX".equals(line)) {
            mode = MODE_VERTEX;
          } else if ("//! FRAGMENT".equals(line)) {
            mode = MODE_FRAGMENT;
          } else if (!line.startsWith("//")) {
            switch (mode) {
              case MODE_COMMON:
                vertexSource.append(line).append('\n');
                fragmentSource.append(line).append('\n');
                break;
              case MODE_VERTEX:
                vertexSource.append(line).append('\n');
                break;
              case MODE_FRAGMENT:
                fragmentSource.append(line).append('\n');
                break;
            }
          }
          return true;
        }

        @Override
        public ShaderSource getResult() {
          return new ShaderSource(vertexSource.toString(), fragmentSource.toString());
        }
      });

      return source;
    } catch (IOException e) {
      LOG.log(WARNING, "Failed to load shader source", e);
      return null;
    }
  }

  private static boolean isAtLeastVersion(int major, int minor) {
    int val = GL11.glGetInteger(GL30.GL_MAJOR_VERSION);
    return (val > major) || (val == major && GL11.glGetInteger(GL30.GL_MINOR_VERSION) >= minor);
  }
}
