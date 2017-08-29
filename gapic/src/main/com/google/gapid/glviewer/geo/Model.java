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

import com.google.gapid.proto.service.api.API;

/**
 * The geometry data of a model to be displayed.
 */
public class Model {
  private final API.DrawPrimitive primitive;
  private final API.Mesh.Stats stats;
  private final float[] positions; // x, y, z
  private final float[] normals; // x, y, z
  private final int[] indices;
  private final BoundingBox bounds = new BoundingBox();

  public Model(API.DrawPrimitive primitive, API.Mesh.Stats stats, float[] positions,
      float[] normals, int[] indices) {
    this.primitive = primitive;
    this.stats = stats;
    this.positions = positions;
    this.normals = normals;
    this.indices = indices;

    if (indices == null) {
      for (int i = 0; i < positions.length; i += 3) {
        bounds.add(positions[i + 0], positions[i + 1], positions[i + 2]);
      }
    } else {
      for (int i = 0; i < indices.length; i++) {
        int idx = 3 * indices[i];
        if (idx >= 0 && idx + 2 < positions.length) {
          bounds.add(positions[idx + 0], positions[idx + 1], positions[idx + 2]);
        }
      }
    }
  }

  public API.DrawPrimitive getPrimitive() {
    return primitive;
  }

  public float[] getPositions() {
    return positions;
  }

  public float[] getNormals() {
    return normals;
  }

  public int[] getIndices() {
    return indices;
  }

  public BoundingBox getBounds() {
    return bounds;
  }

  public String getStatusMessage() {
    StringBuilder sb = new StringBuilder();
    int v = stats.getVertices(), i = stats.getIndices(), p = stats.getPrimitives();
    sb.append(v).append(v != 1 ? " vertices, " : " vertex, ");
    sb.append(i).append(i != 1 ? " indices" :  " index");
    switch (primitive) {
      case Points:
        sb.append(", ").append(p).append(p != 1 ? " points" : " point");
        break;
      case Lines:
      case LineStrip:
      case LineLoop:
        sb.append(", ").append(p).append(p != 1 ? " lines" : " line");
        break;
      case Triangles:
      case TriangleStrip:
      case TriangleFan:
        sb.append(", ").append(p).append(p != 1 ? " triangles" : " triangle");
        break;
      default:
    }
    return sb.toString();
  }
}
