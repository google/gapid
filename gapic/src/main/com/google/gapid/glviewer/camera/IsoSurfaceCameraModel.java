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

import static com.google.gapid.glviewer.Constants.MAX_DISTANCE;
import static com.google.gapid.glviewer.Constants.MIN_DISTANCE;

import com.google.gapid.glviewer.CameraModel;
import com.google.gapid.glviewer.vec.MatD;
import com.google.gapid.glviewer.vec.VecD;

//  _______     .     _____     ___
// |       |   / \   /     \  ."   ",
// |       |  /   \ /       \/       \
// |       | /     \\       /\       /
// |_______|/_______\\_____/  '.___.'
//

/**
 * A {@link CameraModel} that moves the camera constrained to the isosurface of a given
 * {@link Emitter}. The camera's position and direction is interpolated between a given base
 * {@link CameraModel} and the isosurface based on the zoom level. Thus, this model smoothly
 * transitions from the base model when fully zoomed out to the isosurface model when zoomed all
 * the way in.
 *
 * The position and direction of the camera is determined by casting a ray from the base camera's
 * position towards the origin. Let P be the closest interception of this ray with the isosurface.
 * The direction of the camera is the inverse of the normal of the isosurface at P. The normal is
 * computed numerically by sampling the isosurface around P, The camera's position is placed at the
 * desired zoom distance away from P along the normal. Thus, the camera is position at
 * <code>C = P + zoom * N</code>, looking at P.
 */
public class IsoSurfaceCameraModel implements CameraModel {
  private static final double SMOOTHNESS_ROTATION = 0.8;
  private static final int SMOOTHNESS_GRID_SIZE = 5; // must be odd
  private static final int INDEX_OF_Y1_X0 =
      (SMOOTHNESS_GRID_SIZE - 1) / 2 + SMOOTHNESS_GRID_SIZE * ((SMOOTHNESS_GRID_SIZE - 1) / 2 + 1);

  private final CameraModel base;
  private RayCaster rayCaster;
  private double inflation;

  private MatD viewTransform = MatD.IDENTITY;

  private double lastDistance = 1;

  public IsoSurfaceCameraModel(CameraModel base) {
    this.base = base;
    update();
  }

  public void setEmitter(Emitter emitter) {
    rayCaster = new RayCaster(emitter);
    inflation = Math.max(0, MIN_DISTANCE - emitter.getOffset());
  }

  @Override
  public void updateViewport(double screenWidth, double screenHeight) {
    base.updateViewport(screenWidth, screenHeight);
  }

  @Override
  public void onDrag(double dx, double dy) {
    base.onDrag(dx, dy);
    update();
  }

  @Override
  public void onZoom(double dz) {
    base.onZoom(dz);
    update();
  }

  private void update() {
    if (rayCaster == null || base.getZoom() == 0 || !updateUsingIsoSurface()) {
      viewTransform = base.getViewTransform();
    }
  }

  private boolean updateUsingIsoSurface() {
    double[] m = base.getViewTransform().inverseOfTop3x3();
    VecD up = new VecD(m[3], m[4], m[5]).normalize();
    VecD direction = new VecD(-m[6], -m[7], -m[8]).normalize();
    return evaluateIsoSurface(up, direction);
  }

  private boolean evaluateIsoSurface(VecD up, VecD direction) {
    VecD pos = getFirstIntersectionWithHint(direction);
    if (pos == null) {
      return false;
    }

    VecD right = direction.cross(up).normalize().multiply(
        SMOOTHNESS_ROTATION / SMOOTHNESS_GRID_SIZE);
    up = up.multiply(SMOOTHNESS_ROTATION / SMOOTHNESS_GRID_SIZE);

    // Generate the samples to compute the normal.
    VecD[] grid = new VecD[SMOOTHNESS_GRID_SIZE * SMOOTHNESS_GRID_SIZE];
    int size = (SMOOTHNESS_GRID_SIZE - 1) / 2;
    for (int y = -size, index = 0; y <= size; y++) {
      for (int x = -size; x <= size; x++, index++) {
        grid[index] = pos.addScaled(up, y).addScaled(right, x);
        VecD dir = grid[index].multiply(-1).normalize();
        if ((grid[index] = rayCaster.getIntersection(grid[index], dir)) == null) {
          return false;
        }
      }
    }

    // Compute the normal based on the sampling grid.
    VecD normal = new VecD();
    for (int y = 0; y < SMOOTHNESS_GRID_SIZE - 1; y++) {
      for (int x = 0; x < SMOOTHNESS_GRID_SIZE - 1; x++) {
        int p0 = y * SMOOTHNESS_GRID_SIZE + x;
        int p1 = (y + 1) * SMOOTHNESS_GRID_SIZE + x;
        int p2 = y * SMOOTHNESS_GRID_SIZE + x + 1;

        VecD upOffset = grid[p1].subtract(grid[p0]);
        VecD rightOffset = grid[p2].subtract(grid[p0]);
        normal = normal.add(rightOffset.cross(upOffset).normalize());
      }
    }

    normal = normal.normalize();
    up = grid[INDEX_OF_Y1_X0].subtract(pos).normalize();
    pos = pos.addScaled(normal, inflation);

    // Interpolate the isosurface position & direction with the base's based on the zoom amount.
    double zoom = getZoom();
    normal = normal.multiply(-1).lerp(direction, zoom);
    pos = pos.lerp(direction.multiply(-MAX_DISTANCE), zoom);
    viewTransform = MatD.lookAt(pos, pos.add(normal), up);
    return true;
  }

  /**
   * Finds the the closest interception with the isosurface for the given direction. Uses the last
   * computed distance as a starting point for optimization (typically, the isosurface ought to be
   * smooth for smooth camera movement).
   */
  private VecD getFirstIntersectionWithHint(VecD direction) {
    VecD pos = direction.multiply(-lastDistance);
    VecD result = rayCaster.getIntersection(pos, direction);
    if (result != null) {
      lastDistance = result.magnitude();
      return result;
    }
    return null;
  }

  @Override
  public MatD getViewTransform() {
    return viewTransform;
  }

  @Override
  public MatD getProjection() {
    return base.getProjection();
  }

  @Override
  public double getZoom() {
    return base.getZoom();
  }
}
