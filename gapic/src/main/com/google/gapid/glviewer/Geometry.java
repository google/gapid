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

import com.google.gapid.glviewer.camera.Emitter;
import com.google.gapid.glviewer.geo.BoundingBox;
import com.google.gapid.glviewer.geo.Model;
import com.google.gapid.glviewer.gl.Buffer;
import com.google.gapid.glviewer.vec.MatD;
import com.google.gapid.proto.service.gfxapi.GfxAPI;
import com.google.gapid.proto.service.gfxapi.GfxAPI.DrawPrimitive;

import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL15;

public class Geometry {
  private Model model;
  private MatD modelMatrix;
  private boolean zUp;

  public Geometry() {
    updateModelMatrix();
  }

  public void setModel(Model model) {
    this.model = model;
    updateModelMatrix();
  }

  public Model getModel() {
    return model;
  }

  public boolean toggleZUp() {
    zUp = !zUp;
    updateModelMatrix();
    return zUp;
  }

  public BoundingBox getBounds() {
    if (model != null) {
      return model.getBounds();
    }
    return BoundingBox.INVALID;
  }

  private void updateModelMatrix() {
    modelMatrix = getBounds().getCenteringMatrix(Constants.SCENE_SCALE_FACTOR, zUp);
  }

  protected MatD getModelMatrix() {
    return modelMatrix;
  }

  public Renderable asRenderable(DisplayMode displayMode) {
    if (model == null) {
      return Renderable.NOOP;
    }

    final int polygonMode = displayMode.glPolygonMode;
    final int modelPrimitive = translatePrimitive(model.getPrimitive());
    final float[] positions = model.getPositions();
    final float[] normals = model.getNormals();
    final int[] indices = isNonPolygonPoints(displayMode) ? null : model.getIndices();

    return new Renderable() {
      private Buffer positionBuffer;
      private Buffer normalBuffer;
      private Buffer indexBuffer;

      @Override
      public void init() {
        positionBuffer = new Buffer(GL15.GL_ARRAY_BUFFER).bind().loadData(positions);
        if (normals != null) {
          normalBuffer = new Buffer(GL15.GL_ARRAY_BUFFER).bind().loadData(normals);
        }
        if (indices != null) {
          indexBuffer = new Buffer(GL15.GL_ELEMENT_ARRAY_BUFFER).bind().loadData(indices);
        }
      }

      @Override
      public void render(State state) {
        state.transform.push(getModelMatrix());
        state.transform.apply(state.shader);

        GL11.glPolygonMode(GL11.GL_FRONT_AND_BACK, polygonMode);

        positionBuffer.bind();
        state.shader.bindAttribute(Constants.POSITION_ATTRIBUTE, 3, GL11.GL_FLOAT, 3 * 4, 0);
        if (normalBuffer != null) {
          normalBuffer.bind();
          state.shader.bindAttribute(Constants.NORMAL_ATTRIBUTE, 3, GL11.GL_FLOAT, 3 * 4, 0);
        } else {
          state.shader.setAttribute(Constants.NORMAL_ATTRIBUTE, 1, 0, 0);
        }
        if (indexBuffer != null) {
          indexBuffer.bind();
          GL11.glDrawElements(modelPrimitive, indices.length, GL11.GL_UNSIGNED_INT, 0);
        } else {
          GL11.glDrawArrays(GL11.GL_POINTS, 0, positions.length / 3);
        }
        state.shader.unbindAttribute(Constants.POSITION_ATTRIBUTE);
        if (normalBuffer != null) {
          state.shader.unbindAttribute(Constants.NORMAL_ATTRIBUTE);
        }
        GL11.glPolygonMode(GL11.GL_FRONT_AND_BACK, GL11.GL_FILL);

        state.transform.pop();
      }

      @Override
      public void dispose() {
        if (positionBuffer != null) {
          positionBuffer.delete();
          positionBuffer = null;
        }
        if (normalBuffer != null) {
          normalBuffer.delete();
          normalBuffer = null;
        }
        if (indexBuffer != null) {
          indexBuffer.delete();
          indexBuffer = null;
        }
      }
    };
  }

  public Emitter getEmitter() {
    return Emitter.BoxEmitter.fromBoundingBox(getBounds().transform(modelMatrix));
  }

  public static boolean isPolygon(GfxAPI.DrawPrimitive primitive) {
    switch (primitive) {
      case Triangles:
      case TriangleFan:
      case TriangleStrip:
        return true;
      default:
        return false;
    }
  }

  private boolean isNonPolygonPoints(DisplayMode displayMode) {
    return displayMode == DisplayMode.POINTS && !isPolygon(model.getPrimitive());
  }

  private static int translatePrimitive(DrawPrimitive primitive) {
    switch (primitive) {
      case Points:
        return GL11.GL_POINTS;
      case Lines:
        return GL11.GL_LINES;
      case LineStrip:
        return GL11.GL_LINE_STRIP;
      case LineLoop:
        return GL11.GL_LINE_LOOP;
      case Triangles:
        return GL11.GL_TRIANGLES;
      case TriangleStrip:
        return GL11.GL_TRIANGLE_STRIP;
      case TriangleFan:
        return GL11.GL_TRIANGLE_FAN;
      default:
        throw new AssertionError();
    }
  }

  public static enum DisplayMode {
    POINTS(GL11.GL_POINT),
    LINES(GL11.GL_LINE),
    TRIANGLES(GL11.GL_FILL);

    public final int glPolygonMode;

    DisplayMode(int glPolygonMode) {
      this.glPolygonMode = glPolygonMode;
    }
  }
}
