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
package com.google.gapid.glviewer.camera;

import com.google.gapid.glviewer.geo.BoundingBox;
import com.google.gapid.glviewer.vec.VecD;

/**
 * Emits a potential field to define an isosurface to be used by the {@link IsoSurfaceCameraModel}.
 * The isosurface is located where this potential is 0 and is used to determine the camera's
 * position and view direction. Points with positive potential are outside the isosurface and
 * points with negative potential are inside.
 *
 * Typically the geometry of the object the  camera is looking at, or a representative
 * approximation, along with a desired distance is used to define the isosurface. The isosurface
 * then is comprised of all the points that are the given distance away from the underlying
 * geometry.
 */
public interface Emitter {
  /**
   * @return the offset to be used to adjust the zoom distance in the camera model.
   */
  public double getOffset();

  /**
   * @return the potential of this emitter's field at the given point. Must be positive. The
   * isosurface is defined by all the points where this returns 0.
   */
  public double getPotentialAt(VecD pos);

  /**
   * {@link Emitter} based on a box. The potential is 0 at the given radius outside the box, thus
   * defining an isosurface that is a box with rounded corners and edges of the given radius. The
   * potential is computed as the distance away from that rounded box. The implementation acutally
   * uses the distance squared, to avoid taking many square roots.
   */
  public static class BoxEmitter implements Emitter {
    private static final double RADIUS = 0.2;
    private static final double MIN_SIZE = 2.08 * RADIUS;

    private final VecD size;
    private final VecD center;
    private final double radiusSquared;
    private final double offset;

    public BoxEmitter(VecD min, VecD max, double radiusSquared, double offset) {
      this.size = max.subtract(min).multiply(0.5);
      this.center = min.add(size);
      this.radiusSquared = radiusSquared;
      this.offset = offset;
    }

    public static BoxEmitter fromBoundingBox(BoundingBox bbox) {
      VecD min = VecD.fromArray(bbox.min), max = VecD.fromArray(bbox.max);

      VecD size = max.subtract(min);
      VecD delta = size.subtract(size.subtract(2 * RADIUS).max(MIN_SIZE)).multiply(0.5);
      min = min.add(delta);
      max = max.subtract(delta);
      return new BoxEmitter(min, max, RADIUS * RADIUS, Math.max(size.x, Math.max(size.y, size.z)));
    }

    @Override
    public double getOffset() {
      return offset;
    }

    @Override
    public double getPotentialAt(VecD pos) {
      VecD d = pos.subtract(center).abs().subtract(size);
      double r = Math.max(d.x, Math.max(d.y, d.z));
      if (r < 0) {
        // Inside box.
        return -(r * r + this.radiusSquared);
      }
      d = d.max(0);
      return d.magnitudeSquared() - radiusSquared;
    }
  }
}
