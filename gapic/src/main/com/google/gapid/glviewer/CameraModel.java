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

import com.google.gapid.glviewer.vec.MatD;

/**
 * Controls the 3D camera, responsible for translating user input events into a view transform.
 */
public interface CameraModel {
  /**
   * Event that indicates the viewport has changed and the camera should update the projection as
   * the aspect ratio might have changed.
   */
  public void updateViewport(double screenWidth, double screenHeight);

  /**
   * Event that indicates the user has requested the camera to zoom in/out.
   */
  public void onZoom(double dz);

  /**
   * Event that indicates the user has requested to drag the camera.
   */
  public void onDrag(double dx, double dy);

  /**
   * @return the current projection transform to use.
   */
  public MatD getProjection();

  /**
   * @return the current view transform to use.
   */
  public MatD getViewTransform();

  /**
   * @return the current zoom level as a value in the range [0, 1], where 0 means fully zoomed out.
   */
  public double getZoom();
}
