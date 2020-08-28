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
import com.google.gapid.glviewer.gl.IndexBuffer;
import com.google.gapid.glviewer.gl.Renderer;
import com.google.gapid.glviewer.gl.VertexBuffer;
import com.google.gapid.glviewer.vec.MatD;
import com.google.gapid.proto.service.api.API;

import org.lwjgl.opengl.GL11;

/**
 * Renders a {@link Model}. Can render the geometry using either y-up or z-up and as either a
 * point cloud, wire mesh, or solid.
 */
public class Geometry {
  public static final Geometry NULL = new Geometry(null, false, false);

  public final Model model;
  public final boolean zUp;
  public final boolean flipUpAxis;
  public final MatD modelMatrix;

  public Geometry(Model model, boolean zUp, boolean flipUpAxis) {
    this.model = model;
    this.zUp = zUp;
    this.flipUpAxis = flipUpAxis;
    this.modelMatrix = getBounds().getCenteringMatrix(Constants.SCENE_SCALE_FACTOR, zUp, flipUpAxis);
  }

  public BoundingBox getBounds() {
    return (model == null) ? new BoundingBox() : model.getBounds();
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
      private VertexBuffer positionBuffer;
      private VertexBuffer normalBuffer;
      private IndexBuffer indexBuffer;

      @Override
      public void init(Renderer renderer) {
        positionBuffer = renderer.newVertexBuffer(positions, 3);
        if (normals != null) {
          normalBuffer = renderer.newVertexBuffer(normals, 3);
        }
        if (indices != null) {
          indexBuffer = renderer.newIndexBuffer(indices);
        }
      }

      @Override
      public void render(Renderer renderer, State state) {
        state.transform.push(modelMatrix);
        state.transform.apply(state.shader);

        GL11.glPolygonMode(GL11.GL_FRONT_AND_BACK, polygonMode);

        state.shader.setAttribute(Constants.POSITION_ATTRIBUTE, positionBuffer);
        if (normalBuffer != null) {
          state.shader.setAttribute(Constants.NORMAL_ATTRIBUTE, normalBuffer);
        } else {
          state.shader.setAttribute(Constants.NORMAL_ATTRIBUTE, 1, 0, 0);
        }
        if (indexBuffer != null) {
          Renderer.draw(state.shader, modelPrimitive, indexBuffer);
        } else {
          Renderer.draw(state.shader, GL11.GL_POINTS, positions.length / 3);
        }
        GL11.glPolygonMode(GL11.GL_FRONT_AND_BACK, GL11.GL_FILL);

        state.transform.pop();
      }

      @Override
      public void dispose(Renderer renderer) {
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

  /**
   * @return an {@link Emitter} based on the bounding box.
   */
  public Emitter getEmitter() {
    return Emitter.BoxEmitter.fromBoundingBox(getBounds().transform(modelMatrix));
  }

  /**
   * @return whether the given {@link com.google.gapid.proto.service.api.API.DrawPrimitive} will be
   * considered a polygon by GL. I.e. not points or lines.
   */
  public static boolean isPolygon(API.DrawPrimitive primitive) {
    switch (primitive) {
      case Triangles:
      case TriangleFan:
      case TriangleStrip:
        return true;
      default:
        return false;
    }
  }

  /**
   * @return whether the given {@link DisplayMode} will require special handling when rendering
   * the geometry. We control the rendering mode (points, wire mesh, solid) using glPolygonMode,
   * which only works if the underlying geometry is polygon. Since rendering lines as lines is fine,
   * even if it ignores glPolygonMode, the only case that requires special handling is rendering
   * non-polygons as points.
   */
  private boolean isNonPolygonPoints(DisplayMode displayMode) {
    return displayMode == DisplayMode.POINTS && !isPolygon(model.getPrimitive());
  }

  private static int translatePrimitive(API.DrawPrimitive primitive) {
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
