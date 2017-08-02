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
package com.google.gapid.glviewer.geo;

import java.io.IOException;
import java.io.Writer;

/**
 * Writes {@link Model models} to OBJ files.
 */
public class ObjWriter {
  private ObjWriter() {
  }

  public static void write(Writer out, Model model) throws IOException {
    // TODO: could possibly make the OBJ smaller by deduping vertices and normals.
    writePositions(out, model.getPositions());
    writeNormals(out, model.getNormals());
    out.write("s 1\n");
    switch (model.getPrimitive()) {
      case Points: writePoints(out, model.getIndices()); break;
      case Lines: writeLines(out, model.getIndices()); break;
      case LineStrip: writeLineStrip(out, model.getIndices()); break;
      case LineLoop: writeLineLoop(out, model.getIndices()); break;
      case Triangles: writeTriangles(out, model.getIndices()); break;
      case TriangleStrip: writeTriangleStrip(out, model.getIndices()); break;
      case TriangleFan: writeTriangleFan(out, model.getIndices()); break;
      default: throw new IOException("Unsupported draw primitive: " + model.getPrimitive());
    }
  }

  private static void writePositions(Writer out, float[] pos) throws IOException {
    for (int i = 2; i < pos.length; i += 3) {
      out.write("v " + pos[i - 2] + " " + pos[i - 1] + " " + pos[i - 0] + "\n");
    }
  }

  private static void writeNormals(Writer out, float[] normals) throws IOException {
    for (int i = 2; i < normals.length; i += 3) {
      out.write("vn " + normals[i - 2] + " " + normals[i - 1] + " " + normals[i - 0] + "\n");
    }
  }

  private static void writePoints(Writer out, int[] indices) throws IOException {
    for (int i : indices) {
      out.write("p " + (i + 1) + "\n");
    }
  }

  private static void writeLines(Writer out, int[] indices) throws IOException {
    for (int i = 1; i < indices.length; i += 2) {
      writeLine(out, indices[i - 1], indices[i - 0]);
    }
  }

  private static void writeLineStrip(Writer out, int[] indices) throws IOException {
    for (int i = 1; i < indices.length; i++) {
      writeLine(out, indices[i - 1], indices[i]);
    }
  }

  private static void writeLineLoop(Writer out, int[] indices) throws IOException {
    for (int i = 1; i < indices.length; i++) {
      writeLine(out, indices[i - 1], indices[i]);
    }
    writeLine(out, indices[indices.length - 1], 0);
  }

  private static void writeLine(Writer out, int a, int b) throws IOException {
    out.write("l " + (a + 1) + " " + (b + 1) + "\n");
  }

  private static void writeTriangles(Writer out, int[] indices) throws IOException {
    for (int i = 2; i < indices.length; i += 3) {
      writeTriangle(out, indices[i - 2], indices[i - 1], indices[i - 0]);
    }
  }

  private static void writeTriangleStrip(Writer out, int[] indices) throws IOException {
    for (int i = 2; i < indices.length; i += 3) {
      writeTriangle(out, indices[i - 2], indices[i - 1], indices[i - 0]);
      if ((i += 3) >= indices.length) {
        break;
      }
      writeTriangle(out, indices[i - 1], indices[i - 2], indices[i - 0]);
    }
  }

  private static void writeTriangleFan(Writer out, int[] indices) throws IOException {
    for (int i = 2; i < indices.length; i++) {
      writeTriangle(out, indices[0], indices[i - 1], indices[i]);
    }
  }

  private static void writeTriangle(Writer out, int a, int b, int c) throws IOException {
    out.write("f " + (a + 1) + "//" + (a + 1) + " " + (b + 1) + "//" + (b + 1) +
        " " + (c + 1) + "//" + (c + 1) + "\n");
  }
}
