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

import com.google.gapid.glviewer.vec.VecD;

/**
 * Casts rays to find intersections with the isosurface of an {@link Emitter}.
 */
public class RayCaster {
  private static final int MAX_LINEAR_STEPS = 128;
  private static final int MAX_BINARY_STEPS = 32;
  private static final double PRECISION = 0.00001;
  private static final double MIN_STEP = 0.05;

  private final Emitter emitter;

  public RayCaster(Emitter emitter) {
    this.emitter = emitter;
  }

  // direction needs to be normalized
  // result is null if no intersection is found.
  public VecD getIntersection(VecD src, VecD direction) {
    double potential = emitter.getPotentialAt(src);
    boolean inside = potential < 0;
    VecD prev = src;
    for (int i = 0; true; i++) {
      double absPotential = Math.abs(potential);
      if (absPotential < PRECISION) {
        return src;
      } else if ((potential < 0) != inside) {
        break;
      } else if (absPotential < MIN_STEP) {
        potential = Math.signum(potential) * MIN_STEP;
      }
      prev = src;
      src = src.addScaled(direction, potential);
      potential = emitter.getPotentialAt(src);

      if (i >= MAX_LINEAR_STEPS) {
        return null;
      }
    }

    return inside ? binarySearch(prev, src) : binarySearch(src, prev);
  }

  private VecD binarySearch(VecD start, VecD end) {
    for (int i = 0; i < MAX_BINARY_STEPS; i++) {
      VecD mid = start.add(end).multiply(0.5);
      double potential = emitter.getPotentialAt(mid);
      if (Math.abs(potential) < PRECISION) {
        return mid;
      }

      if (potential < 0) {
        start = mid;
      } else {
        end = mid;
      }
    }
    return null;
  }
}
