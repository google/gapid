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

import com.google.common.collect.Queues;
import com.google.gapid.glviewer.gl.Shader;
import com.google.gapid.glviewer.vec.MatD;

import java.util.Deque;

/**
 * Binds the projection and model-view transforms to a {@link Shader}. Maintains a matrix stack for
 * the model-view transform.
 */
public class ModelViewProjection {
  private final boolean invertNormals;
  private MatD modelView = MatD.IDENTITY;
  private MatD projection = MatD.IDENTITY;
  private final Deque<MatD> matrixStack = Queues.newArrayDeque();

  public ModelViewProjection(boolean invertNormals) {
    this.invertNormals = invertNormals;
  }

  public void setProjection(MatD projection) {
    this.projection = projection;
  }

  public void setModelView(MatD modelView) {
    this.modelView = modelView;
  }

  public void push(MatD transform) {
    matrixStack.push(modelView);
    modelView = modelView.multiply(transform);
  }

  public void pop() {
    modelView = matrixStack.pop();
  }

  public void apply(Shader shader) {
    shader.setUniform(Constants.MODEL_VIEW_UNIFORM, modelView.toFloatArray());
    shader.setUniform(
        Constants.MODEL_VIEW_PROJECTION_UNIFORM, projection.multiply(modelView).toFloatArray());
    shader.setUniform(Constants.NORMAL_MATRIX_UNIFORM, modelView.toNormalMatrix(invertNormals));
    shader.setUniform(Constants.INVERT_NORMALS_UNIFORM, invertNormals ? -1f : 1f);
  }
}
