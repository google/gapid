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

import static com.google.gapid.glviewer.vec.MatD.makeScaleTranslationZupToYup;

import com.google.gapid.glviewer.vec.MatD;
import com.google.gapid.glviewer.vec.VecD;

/**
 * Bounding box of a model aligned to the standard cartesian directions.
 */
public class BoundingBox {
  public final double[] min =
      new double[] { Double.POSITIVE_INFINITY, Double.POSITIVE_INFINITY, Double.POSITIVE_INFINITY };
  public final double[] max =
      new double[] { Double.NEGATIVE_INFINITY, Double.NEGATIVE_INFINITY, Double.NEGATIVE_INFINITY };

  public BoundingBox() {
  }

  public BoundingBox copy() {
    BoundingBox r = new BoundingBox();
    System.arraycopy(min, 0, r.min, 0, 3);
    System.arraycopy(max, 0, r.max, 0, 3);
    return r;
  }

  public void add(VecD vec) {
    add(vec.x, vec.y, vec.z);
  }

  public void add(double x, double y, double z) {
    VecD.min(min, x, y, z);
    VecD.max(max, x, y, z);
  }

  /**
   * @return a matrix that will center the model at the origin and scale it to the given size.
   */
  public MatD getCenteringMatrix(double diagonalSize, boolean zUp, boolean flipUpAxis) {
    VecD minV = VecD.fromArray(min), maxV = VecD.fromArray(max);
    double diagonal = maxV.distance(minV);

    VecD translation = maxV.subtract(minV).multiply(0.5f).add(minV).multiply(-1);
    double scale = (diagonal == 0) ? 1 : diagonalSize / diagonal;

    MatD flipMatrix = flipUpAxis ? MatD.makeScale(1, -1, 1) : MatD.IDENTITY;

    return zUp ? flipMatrix.multiply(MatD.makeScaleTranslationZupToYup(scale, translation)) :
        flipMatrix.multiply(MatD.makeScaleTranslation(scale, translation));
  }

  public BoundingBox transform(MatD transform) {
    VecD tMin = transform.multiply(VecD.fromArray(min));
    VecD tMax = transform.multiply(VecD.fromArray(max));
    BoundingBox result = new BoundingBox();
    result.add(tMin);
    result.add(tMax);
    return result;
  }

  @Override
  public String toString() {
    return String.format("(%g, %g, %g) - (%g, %g, %g)", min[0], min[1], min[2], max[0], max[1], max[2]);
  }
}
