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

public interface Constants {
  public static final double SCENE_SCALE_FACTOR = 2.1;
  public static final String POSITION_ATTRIBUTE = "aVertexPosition";
  public static final String NORMAL_ATTRIBUTE = "aVertexNormal";
  public static final String MODEL_VIEW_UNIFORM = "uModelView";
  public static final String MODEL_VIEW_PROJECTION_UNIFORM = "uModelViewProj";
  public static final String NORMAL_MATRIX_UNIFORM = "uNormalMatrix";
  public static final String INVERT_NORMALS_UNIFORM = "uInvertNormals";

  // Camera constants.
  public static final int STANDARD_WIDTH = 36;
  public static final int STANDARD_HEIGHT = 24;
  public static final int NEAR_FOCAL_LENGTH = 105;
  public static final int FAR_FOCAL_LENGTH = 55;
  public static final double MIN_DISTANCE = 3;
  public static final double MAX_DISTANCE = 4.5;
  public static final double Z_NEAR = 0.1;
}
