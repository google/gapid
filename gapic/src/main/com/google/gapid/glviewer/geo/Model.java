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

import com.google.gapid.proto.service.api.API.DrawPrimitive;

/**
 * The geometry data of a model to be displayed.
 */
public class Model {
  private final DrawPrimitive primitive;
  private final float[] positions; // x, y, z
  private final float[] normals; // x, y, z
  private final int[] indices;
  private final BoundingBox bounds = new BoundingBox();

  public Model(DrawPrimitive primitive, float[] positions, float[] normals, int[] indices) {
    this.primitive = primitive;
    this.positions = positions;
    this.normals = normals;
    this.indices = indices;
    for (int i = 0; i < positions.length; i += 3) {
      bounds.add(positions[i + 0], positions[i + 1], positions[i + 2]);
    }
  }

  public DrawPrimitive getPrimitive() {
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
}
