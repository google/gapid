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

import static com.google.gapid.glviewer.Constants.FAR_FOCAL_LENGTH;
import static com.google.gapid.glviewer.Constants.MAX_DISTANCE;
import static com.google.gapid.glviewer.Constants.MIN_DISTANCE;
import static com.google.gapid.glviewer.Constants.NEAR_FOCAL_LENGTH;
import static com.google.gapid.glviewer.Constants.STANDARD_HEIGHT;
import static com.google.gapid.glviewer.Constants.STANDARD_WIDTH;
import static com.google.gapid.glviewer.Constants.Z_NEAR;

import com.google.gapid.glviewer.CameraModel;
import com.google.gapid.glviewer.vec.MatD;

/**
 * A {@link CameraModel} that allows rotating the camera fully around the up axis, and between
 * +/- 90 degrees around the horizontal axis. Comparable to an ArcBall, except that the model is
 * never "upside down".
 */
public class CylindricalCameraModel implements CameraModel {
  private MatD viewTransform;
  private MatD projection;
  private double distance = MAX_DISTANCE;
  private double angleX, angleY;
  private double width = STANDARD_WIDTH, height = STANDARD_HEIGHT;
  private double focallength = FAR_FOCAL_LENGTH;

  public CylindricalCameraModel() {
    updateModelView();
    updateProjection();
  }

  @Override
  public void updateViewport(double screenWidth, double screenHeight) {
    if (screenWidth * STANDARD_HEIGHT > screenHeight * STANDARD_WIDTH) {
      // Aspect ratio is wider than default.
      height = STANDARD_HEIGHT;
      width = screenWidth * STANDARD_HEIGHT / screenHeight;
    } else {
      // Aspect ratio is taller than or equal to the default.
      height = screenHeight * STANDARD_WIDTH / screenWidth;
      width = STANDARD_WIDTH;
    }
    updateProjection();
  }

  @Override
  public void onDrag(double dx, double dy) {
    angleX += dy / 3;
    angleY += dx / 3;

    angleX = Math.min(Math.max(angleX, -90), 90);
    updateModelView();
  }

  @Override
  public void onZoom(double dz) {
    distance = Math.max(MIN_DISTANCE, Math.min(MAX_DISTANCE, distance + dz));
    double scale = (distance - MIN_DISTANCE) / (MAX_DISTANCE - MIN_DISTANCE);
    focallength = NEAR_FOCAL_LENGTH - (scale * (NEAR_FOCAL_LENGTH - FAR_FOCAL_LENGTH));
    updateModelView();
    updateProjection();
  }

  private void updateModelView() {
    viewTransform = MatD.makeTranslationRotXY(0, 0, -distance, angleX, angleY);
  }

  private void updateProjection() {
    projection = MatD.projection(width, height, focallength, Z_NEAR);
  }

  @Override
  public MatD getProjection() {
    return projection;
  }

  @Override
  public MatD getViewTransform() {
    return viewTransform;
  }

  @Override
  public double getZoom() {
    return 1 - ((distance - MIN_DISTANCE) / (MAX_DISTANCE - MIN_DISTANCE));
  }
}
