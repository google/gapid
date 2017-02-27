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

import com.google.gapid.glviewer.gl.Shader;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.widgets.GlComposite;

import org.eclipse.swt.SWT;
import org.eclipse.swt.events.MouseEvent;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL20;
import org.lwjgl.opengl.GL30;

import java.util.logging.Logger;

/**
 * Renders a {@link Renderable} and handles user interactions with the given {@link CameraModel}.
 * This is the main entry point of the GL model viewer.
 */
public class Viewer implements GlComposite.Listener {
  private static final Logger LOG = Logger.getLogger(Viewer.class.getName());

  private final CameraModel camera;
  private Shaders shaders;
  private Renderable renderable;
  private Renderable newRenderable;
  private Shading shading = Shading.LIT;
  private Winding winding = Winding.CCW;
  private Culling culling = Culling.OFF;

  public Viewer(CameraModel camera) {
    this.camera = camera;
  }

  /**
   * Hooks up the mouse handling event listener to the given canvas.
   */
  public void addMouseListeners(GlComposite canvas) {
    MouseHandler handler = new MouseHandler(camera, canvas);
    canvas.getControl().addMouseListener(handler);
    canvas.getControl().addMouseMoveListener(handler);
    canvas.getControl().addMouseWheelListener(handler);
  }

  public void setRenderable(Renderable renderable) {
    if (this.renderable != renderable) {
      newRenderable = renderable;
    } else {
      newRenderable = null;
    }
  }

  public Shading getShading() {
    return shading;
  }

  public Culling getCulling() {
    return culling;
  }

  public Winding getWinding() {
    return winding;
  }

  public void setShading(Shading shading) {
    this.shading = shading;
  }

  public void setCulling(Culling culling) {
    this.culling = culling;
  }

  public Culling toggleCulling() {
    culling = culling.toggle();
    return culling;
  }

  public void setWinding(Winding winding) {
    this.winding = winding;
  }

  public Winding toggleWinding() {
    winding = winding.toggle();
    return winding;
  }

  @Override
  public void init() {
    float[] background = new float[] { .2f, .2f, .2f, 1f };

    LOG.log(FINE, "GL Version:   " + GL11.glGetString(GL11.GL_VERSION));
    LOG.log(FINE, "GLSL Version: " + GL11.glGetString(GL20.GL_SHADING_LANGUAGE_VERSION));

    shaders = Shaders.init();
    if (renderable != null) {
      renderable.init();
    }

    GL11.glEnable(GL11.GL_DEPTH_TEST);
    GL11.glClearColor(background[0], background[1], background[2], background[3]);
    GL11.glPointSize(4);
    GL30.glBindVertexArray(GL30.glGenVertexArrays());
  }

  @Override
  public void reshape(int x, int y, int width, int height) {
    GL11.glViewport(x, y, width, height);
    camera.updateViewport(width, height);
  }

  @Override
  public void display() {
    GL11.glClear(GL11.GL_COLOR_BUFFER_BIT | GL11.GL_DEPTH_BUFFER_BIT);

    if (newRenderable != null) {
      if (renderable != null) {
        renderable.dispose();
      }
      renderable = newRenderable;
      renderable.init();
      newRenderable = null;
    }

    if (renderable != null) {
      Renderable.State state = shading.getState(shaders, winding.invertNormals);
      culling.apply();
      winding.apply();

      state.transform.setProjection(camera.getProjection());
      state.transform.setModelView(camera.getViewTransform());
      renderable.render(state);
    }
  }

  @Override
  public void dispose() {
    shaders.delete();
    if (renderable != null) {
      renderable.dispose();
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
        state.shader.bind();
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
            new float[] { 0.37868f, 0.56050f, 0.03703f }); // #A4C639 in linear.
        state.shader.setUniform("uSpecularColor", new float[] { 0.3f, 0.3f, 0.3f });
        state.shader.setUniform("uRoughness", 0.25f);
        return state;
      }
    },
    FLAT() {
      @Override
      public Renderable.State getState(Shaders shaders, boolean invertNormals) {
        Renderable.State state = new Renderable.State(shaders.flatShader, false);
        state.shader.bind();
        state.shader.setUniform("uDiffuseColor",
            new float[] { 0.640625f, 0.7734375f, 0.22265625f }); // #A4C639 in sRGB.
        return state;
      }
    },
    NORMALS() {
      @Override
      public Renderable.State getState(Shaders shaders, boolean invertNormals) {
        Renderable.State state = new Renderable.State(shaders.normalShader, invertNormals);
        state.shader.bind();
        return state;
      }
    };

    public abstract Renderable.State getState(Shaders shaders, boolean invertNormals);
  }

  private static class MouseHandler extends MouseAdapter {
    private final CameraModel camera;
    private final GlComposite canvas;
    private int lastX, lastY;

    public MouseHandler(CameraModel camera, GlComposite canvas) {
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

    public static Shaders init() {
      return new Shaders(
        ShaderSource.loadShader("lit"),
        ShaderSource.loadShader("flat"),
        ShaderSource.loadShader("normals"));
    }

    public void delete() {
      litShader.delete();
      flatShader.delete();
      normalShader.delete();
    }
  }
}
