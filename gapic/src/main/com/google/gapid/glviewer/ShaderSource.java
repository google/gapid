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
import com.google.gapid.glviewer.gl.Shader;

import java.io.IOException;
import java.net.URL;
import java.util.logging.Logger;

public class ShaderSource {
  private static final Logger LOG = Logger.getLogger(ShaderSource.class.getName());

  private ShaderSource() {
  }

  public static Shader loadShader(String name) {
    return loadShader(Resources.getResource("shaders/" + name + ".glsl"));
  }

  public static Shader loadShader(URL resource) {
    try {
      Source source = Resources.readLines(resource, Charsets.US_ASCII, new LineProcessor<Source>() {
        private static final int MODE_COMMON = 0;
        private static final int MODE_VERTEX = 1;
        private static final int MODE_FRAGMENT = 2;
        private static final String VERTEX_PREAMBLE = "#version 150\n#define varying out\n";
        private static final String FRAGMENT_PREAMBLE = "#version 150\n#define varying in\n";

        private final StringBuilder vertexSource = new StringBuilder(VERTEX_PREAMBLE);
        private final StringBuilder fragmentSource = new StringBuilder(FRAGMENT_PREAMBLE);

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
        public Source getResult() {
          return new Source(vertexSource.toString(), fragmentSource.toString());
        }
      });

      Shader shader = new Shader();
      if (!shader.link(source.vertex, source.fragment)) {
        shader.delete();
        shader = null;
      }
      return shader;
    } catch (IOException e) {
      LOG.log(WARNING, "Failed to load shader source", e);
      return null;
    }
  }

  private static class Source {
    public final String vertex, fragment;

    public Source(String vertex, String fragment) {
      this.vertex = vertex;
      this.fragment = fragment;
    }
  }
}
