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
package com.google.gapid.glviewer.gl;

import com.google.common.base.Preconditions;
import com.google.gapid.glviewer.Constants;
import com.google.gapid.glviewer.ShaderSource;
import com.google.gapid.glviewer.vec.MatD;
import com.google.gapid.glviewer.vec.VecD;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.internal.DPIUtil;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL30;

import java.util.HashSet;
import java.util.Set;

/**
 * Renderer maintains the objects and state for an OpenGL context.
 */
public final class Renderer {
  /** The set of {@link GlObject}s owned by this renderer */
  private final Set<GlObject> objects = new HashSet<>();

  // The primitive shaders, vertex-buffers and index-buffers.
  private Shader solidShader;
  private Shader checkerShader;
  private Shader borderShader;
  private VertexBuffer quadVB;
  private VertexBuffer borderVB;
  private IndexBuffer borderIB;

  /** Width of the canvas in pixels. */
  private int physicalWidth;
  /** Height of the canvas in pixels. */
  private int physicalHeight;

  /** Width of the canvas in device-independent pixels. */
  private int dipWidth;
  /** Height of the canvas in device-independent pixels. */
  private int dipHeight;

  /**
   * Initializes the renderer for use.
   * Must be called before any other methods.
   */
  public void initialize() {
    GL30.glBindVertexArray(GL30.glGenVertexArrays());
    buildPrimitives();
  }

  /**
   * Releases the objects allocated by the renderer.
   * It is not safe to call any other methods after calling terminate.
   */
  public void terminate() {
    deleteOwnedObjects();
  }

  /**
   * Updates the size of the back-buffer.
   * @param width of the back-buffer in real pixels.
   * @param height of the back-buffer in real pixels.
   */
  public void setSize(int width, int height) {
    physicalWidth = width;
    physicalHeight = height;
    dipWidth = DPIUtil.autoScaleDown(width);
    dipHeight = DPIUtil.autoScaleDown(height);
    GL11.glViewport(0, 0, physicalWidth, physicalHeight);
  }

  /** @return the size of the viewport in device-independent pixels. */
  public VecD getViewSize() {
    return new VecD(dipWidth, dipHeight, 0);
  }

  /** @return the width of the viewport in device-independent pixels. */
  public int getViewWidth() {
    return dipWidth;
  }

  /** @return the height of the viewport in device-independent pixels. */
  public int getViewHeight() {
    return dipHeight;
  }

  /**
   * Constructs and returns a new {@link VertexBuffer} filled with {@param data}.
   * The returned {@link VertexBuffer} must only be used with this {@link Renderer}.
   *
   * @param data the vertex data.
   * @param elementsPerVertex number of data elements per vertex.
   */
  public VertexBuffer newVertexBuffer(float[] data, int elementsPerVertex) {
    return new VertexBuffer(this, data, elementsPerVertex);
  }

  /**
   * Constructs and returns a new {@link IndexBuffer} filled with {@param data}.
   * The returned {@link IndexBuffer} must only be used with this {@link Renderer}.
   *
   * @param data the index data.
   */
  public IndexBuffer newIndexBuffer(int[] data) {
    return new IndexBuffer(this, data);
  }

  /**
   * Constructs and returns a new {@link Texture}.
   * The returned {@link Texture} must only be used with this {@link Renderer}.
   */
  public Texture newTexture(int target) {
    return new Texture(this, target);
  }

  /**
   * Loads a new {@link Shader} with the shader source file {@param name}.
   * The returned {@link Shader} must only be used with this {@link Renderer}.
   */
  public Shader loadShader(String name) {
    ShaderSource source = ShaderSource.load(name);
    Shader shader = new Shader(this);
    if (!shader.link(source.vertex, source.fragment)) {
      shader.delete();
    }
    return shader;
  }

  /** Clears the viewport with the solid color {@param c} */
  public void clear(Color c) {
    GL11.glClearColor(c.getRed() / 255f, c.getGreen() / 255f, c.getBlue() / 255f, 1f);
    GL11.glClear(GL11.GL_COLOR_BUFFER_BIT);
  }

  /** draws primitives using the vertex data bound to {@param shader}. */
  public void draw(Shader shader, int primitive, int vertexCount) {
    shader.bind();
    GL11.glDrawArrays(primitive, 0, vertexCount);
  }

  /** draws primitives using the vertex data bound to {@param shader} indexed from {@param indices}. */
  public void draw(Shader shader, int primitive, IndexBuffer indices) {
    shader.bind();
    indices.bind();
    GL11.glDrawElements(primitive, indices.count, indices.type, 0);
  }

  /**
   * draws a normalized (-1 to 1) 2D quad transformed by {@param transform} with {@param shader}.
   * The shader is expected to use a {@code uniform mat4 uTransform} for transforming the positions
   * passed in the {@link Constants#POSITION_ATTRIBUTE}.
   */
  public void drawQuad(MatD transform, Shader shader) {
    shader.setUniform("uTransform", transform.toFloatArray());
    shader.setAttribute(Constants.POSITION_ATTRIBUTE, quadVB);
    shader.bind();
    draw(shader, GL11.GL_TRIANGLE_FAN, 4);
  }

  /**
   * draws a 2D quad using the device-independent coordinates with {@param shader}.
   * The shader is expected to use a {@code uniform mat4 uTransform} for transforming the positions
   * passed in the {@link Constants#POSITION_ATTRIBUTE}.
   */
  public void drawQuad(int x, int y, int w, int h, Shader shader) {
    drawQuad(rectTransform(x, y, w, h), shader);
  }

  public void drawSolid(MatD transform, Color color) {
    solidShader.setUniform("uTransform", transform.toFloatArray());
    solidShader.setUniform("uColor", color);
    solidShader.setAttribute(Constants.POSITION_ATTRIBUTE, quadVB);
    solidShader.bind();
    drawQuad(transform, solidShader);
  }

  /** draws a 2D solid color quad using the device-independent coordinates. */
  public void drawSolid(int x, int y, int w, int h, Color color) {
    drawSolid(rectTransform(x, y, w, h), color);
  }

  /** draws a 2D checkered-color quad using the device-independent coordinates. */
  public void drawChecker(MatD transform, Color colorA, Color colorB, int blockSize) {
    checkerShader.setUniform("uTransform", transform.toFloatArray());
    checkerShader.setUniform("uColorA", colorA);
    checkerShader.setUniform("uColorB", colorB);
    checkerShader.setUniform("uCheckerSize", new float[]{blockSize, blockSize});
    checkerShader.setAttribute(Constants.POSITION_ATTRIBUTE, quadVB);
    checkerShader.bind();
    drawQuad(transform, checkerShader);
  }

  /**
   * draws a normalized (-1 to 1) 2D border transformed by {@param transform} with {@param shader}.
   * The border is extruded by {@param width} pixels outward.
   * The shader is expected to use a {@code uniform mat4 uTransform} for transforming the positions
   * passed in the {@link Constants#POSITION_ATTRIBUTE}.
   */
  public void drawBorder(MatD transform, Color color, int width) {
    borderShader.setUniform("uTransform", transform.toFloatArray());
    borderShader.setUniform("uColor", color);
    borderShader.setUniform("uBorderWidth", new float[]{
        2 * width / (float) dipWidth,
        2 * width / (float) dipHeight
    });
    borderShader.setAttribute(Constants.POSITION_ATTRIBUTE, borderVB);
    borderShader.bind();
    draw(borderShader, GL11.GL_TRIANGLES, borderIB);
  }

  /** draws a 2D solid color border quad using the device-independent coordinates. */
  public void drawBorder(int x, int y, int w, int h, Color color, int width) {
    drawBorder(rectTransform(x, y, w, h), color, width);
  }

  /**
   * registers the object with this renderer.
   * The object will be freed when the renderer is destroyed.
   */
  void register(GlObject object) {
    Preconditions.checkState(objects.add(object), "Object was already owned by this renderer");
  }

  /**
   * unregisters the object with this renderer.
   * Should only be called by the object when it is destroyed.
   */
  void unregister(GlObject object) {
    Preconditions.checkState(objects.remove(object), "Object was not owned by this renderer");
  }

  private void buildPrimitives() {
    solidShader = loadShader("solid");
    checkerShader = loadShader("checker");
    borderShader = loadShader("border");

    quadVB = newVertexBuffer(new float[]{-1f, -1f, 1f, -1f, 1f, 1f, -1f, 1f}, 2);

    borderVB = newVertexBuffer(new float[]{
        // x, y, extrude
        -1, -1, 0, /**/ 1, -1, 0, /**/ 1, 1, 0, /**/ -1, 1, 0,
        -1, -1, 1, /**/ 1, -1, 1, /**/ 1, 1, 1, /**/ -1, 1, 1,
    }, 3);

    borderIB = newIndexBuffer(new int[]{
        0, 4, 1, /**/ 1, 4, 5,
        1, 5, 2, /**/ 2, 5, 6,
        2, 6, 3, /**/ 3, 6, 7,
        3, 7, 0, /**/ 0, 7, 4,
    });
  }

  private void deleteOwnedObjects() {
    for (GlObject object : objects.toArray(new GlObject[objects.size()])) {
      object.delete();
    }
    assert(objects.isEmpty());
  }

  private MatD rectTransform(int x, int y, int w, int h) {
    double dx = (x + 0.5 * w) * (2. / dipWidth) - 1;
    double dy = (y + 0.5 * h) * (2. / dipHeight) - 1;
    return MatD
        .translation(dx, dy, 0)
        .scale(w / (double)dipWidth, h / (double)dipHeight, 1);
  }
}
