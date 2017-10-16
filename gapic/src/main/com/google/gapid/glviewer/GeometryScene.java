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

import static java.util.logging.Level.FINE;

import com.google.gapid.glviewer.gl.Renderer;
import com.google.gapid.glviewer.gl.Scene;
import com.google.gapid.glviewer.gl.Shader;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.widgets.ScenePanel;

import org.eclipse.swt.SWT;
import org.eclipse.swt.events.MouseEvent;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL20;
import org.lwjgl.opengl.GL30;

import java.util.logging.Logger;

/**
 * Renders a {@link Geometry} and handles user interactions with the given {@link CameraModel}.
 * This is the main entry point of the GL model viewer.
 */
public class GeometryScene implements Scene<GeometryScene.Data> {
  /**
   * The geometry and display mode for the viewer to render.
   */
  public static class Data {
    public static final Data DEFAULTS = new Data(
        Geometry.NULL, Geometry.DisplayMode.TRIANGLES, Shading.LIT, Winding.CCW, Culling.OFF);

    public final Geometry geometry;
    public final Geometry.DisplayMode displayMode;
    public final Shading shading;
    public final Winding winding;
    public final Culling culling;

    public Data(Geometry geometry, Geometry.DisplayMode displayMode) {
      this(geometry, displayMode, Shading.LIT, Winding.CCW, Culling.OFF);
    }

    public Data(Geometry geometry, Geometry.DisplayMode displayMode, Shading shading,
        Winding winding, Culling culling) {
      this.geometry = geometry;
      this.displayMode = displayMode;
      this.shading = shading;
      this.winding = winding;
      this.culling = culling;
    }

    public Data withToggledWinding() {
      return new Data(geometry, displayMode, shading, winding.toggle(), culling);
    }

    public Data withToggledCulling() {
      return new Data(geometry, displayMode, shading, winding, culling.toggle());
    }

    public Data withShading(Shading newShading) {
      return new Data(geometry, displayMode, newShading, winding, culling);
    }

    public Data withGeometry(Geometry newGeometry, Geometry.DisplayMode newDisplayMode) {
      return new Data(newGeometry, newDisplayMode, shading, winding, culling);
    }
  }

  private static final Logger LOG = Logger.getLogger(GeometryScene.class.getName());

  private final CameraModel camera;
  private Shaders shaders;
  private Renderable renderable;
  private Data data;

  public GeometryScene(CameraModel camera) {
    this.camera = camera;
  }

  // TODO: This is wrong - the camera state is mutated outside of the renderer / scene systems.
  public void bindCamera(ScenePanel<?> canvas) {
    MouseHandler handler = new MouseHandler(camera, canvas);
    canvas.addMouseListener(handler);
    canvas.addMouseMoveListener(handler);
    canvas.addMouseWheelListener(handler);
  }

  @Override
  public void init(Renderer renderer) {
    float[] background = new float[] { .2f, .2f, .2f, 1f };

    LOG.log(FINE, "GL Version:   " + GL11.glGetString(GL11.GL_VERSION));
    LOG.log(FINE, "GLSL Version: " + GL11.glGetString(GL20.GL_SHADING_LANGUAGE_VERSION));

    shaders = Shaders.init(renderer);
    if (renderable != null) {
      renderable.init(renderer);
    }

    GL11.glEnable(GL11.GL_DEPTH_TEST);
    GL11.glClearColor(background[0], background[1], background[2], background[3]);
    GL11.glPointSize(4);
    GL30.glBindVertexArray(GL30.glGenVertexArrays());
  }

  @Override
  public void resize(Renderer renderer, int width, int height) {
    camera.updateViewport(width, height);
  }

  @Override
  public void update(Renderer renderer, Data newData) {
    data = newData;
    if (renderable != null) {
      renderable.dispose(renderer);
    }
    renderable = newData.geometry.asRenderable(newData.displayMode);
    renderable.init(renderer);
  }

  @Override
  public void render(Renderer renderer) {
    GL11.glClear(GL11.GL_COLOR_BUFFER_BIT | GL11.GL_DEPTH_BUFFER_BIT);

    if (data != null && renderable != null) {
      Renderable.State state = data.shading.getState(shaders, data.winding.invertNormals);
      data.culling.apply();
      data.winding.apply();

      state.transform.setProjection(camera.getProjection());
      state.transform.setModelView(camera.getViewTransform());
      renderable.render(renderer, state);
    }
  }

  /**
   * Back-face culling mode.
   */
  public static enum Culling {
    OFF() {
      @Override
      public void apply() {
        GL11.glDisable(GL11.GL_CULL_FACE);
      }
    },
    ON() {
      @Override
      public void apply() {
        GL11.glEnable(GL11.GL_CULL_FACE);
      }
    };

    public abstract void apply();

    public Culling toggle() {
      return (this == OFF) ? ON : OFF;
    }
  }

  /**
   * Triangle winding mode.
   */
  public static enum Winding {
    CCW(false) {
      @Override
      public void apply() {
        GL11.glFrontFace(GL11.GL_CCW);
      }
    },
    CW(true) {
      @Override
      public void apply() {
        GL11.glFrontFace(GL11.GL_CW);
      }
    };

    public final boolean invertNormals;

    Winding(boolean invertNormals) {
      this.invertNormals = invertNormals;
    }

    public abstract void apply();

    public Winding toggle() {
      return (this == CCW) ? CW : CCW;
    }
  }

  /**
   * Different shading options to use when rendering.
   */
  public static enum Shading {
    LIT() {
      @Override
      public Renderable.State getState(Shaders shaders, boolean invertNormals) {
        Renderable.State state = new Renderable.State(shaders.litShader, invertNormals);
        state.shader.setUniform("uLightDir", new float[] {
          0,      -0.707f, -0.707f,
          0,       0.707f, -0.707f,
         -0.707f,  0,       0.707f,
          0.707f,  0,       0.707f
        });
        state.shader.setUniform("uLightColor", new float[] {
          0.2f, 0.2f, 0.2f,
          0.4f, 0.4f, 0.4f,
          0.5f, 0.5f, 0.5f,
          1.0f, 1.0f, 1.0f
        });
        state.shader.setUniform("uLightSpecColor", new float[] {
          0.0f, 0.0f, 0.0f,
          0.5f, 0.5f, 0.5f,
          0.5f, 0.5f, 0.5f,
          1.0f, 1.0f, 1.0f
        });
        state.shader.setUniform("uLightSize", new float[] {
          0f, 0.05f, 0.05f, 0f
        });
        state.shader.setUniform("uDiffuseColor",
            new float[] { 0, 0.48777f, 0.66612f, }); // #00B8D4 in linear.
        state.shader.setUniform("uSpecularColor", new float[] { 0.3f, 0.3f, 0.3f });
        state.shader.setUniform("uRoughness", 0.25f);
        return state;
      }
    },
    FLAT() {
      @Override
      public Renderable.State getState(Shaders shaders, boolean invertNormals) {
        Renderable.State state = new Renderable.State(shaders.flatShader, false);
        state.shader.setUniform("uDiffuseColor",
            new float[] { 0, 0.721568627f, 0.831372549f }); // #00B8D4 in sRGB.
        return state;
      }
    },
    NORMALS() {
      @Override
      public Renderable.State getState(Shaders shaders, boolean invertNormals) {
        return new Renderable.State(shaders.normalShader, invertNormals);
      }
    };

    public abstract Renderable.State getState(Shaders shaders, boolean invertNormals);
  }

  private static class MouseHandler extends MouseAdapter {
    private final CameraModel camera;
    private final ScenePanel<?> canvas;
    private int lastX, lastY;

    public MouseHandler(CameraModel camera, ScenePanel<?> canvas) {
      this.camera = camera;
      this.canvas = canvas;
    }

    @Override
    public void mouseScrolled(MouseEvent e) {
      camera.onZoom(-e.count / 18.0f);
      canvas.paint();
    }

    @Override
    public void mouseDown(MouseEvent e) {
      lastX = e.x;
      lastY = e.y;
    }

    @Override
    public void mouseMove(MouseEvent e) {
      if ((e.stateMask & SWT.BUTTON1) != 0) {
        camera.onDrag(e.x - lastX, e.y - lastY);
        canvas.paint();
      }
      lastX = e.x;
      lastY = e.y;
    }
  }

  private static class Shaders {
    public final Shader litShader;
    public final Shader flatShader;
    public final Shader normalShader;

    private Shaders(Shader litShader, Shader flatShader, Shader normalShader) {
      this.litShader = litShader;
      this.flatShader = flatShader;
      this.normalShader = normalShader;
    }

    public static Shaders init(Renderer renderer) {
      return new Shaders(
          renderer.loadShader("lit"),
          renderer.loadShader("flat"),
          renderer.loadShader("normals"));
    }
  }
}
