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

#ifndef GAPIR_GLES_RENDERER_H
#define GAPIR_GLES_RENDERER_H

#include "renderer.h"

namespace gapir {

// The GLES renderer implementation, which creates a OpenGL / GLES rendering
// context.
class GlesRenderer : public Renderer {
 public:
  // Backbuffer describes a backbuffer dimensions and format.
  struct Backbuffer {
    struct Format {
      inline Format();
      inline Format(uint32_t c, uint32_t d, uint32_t s);
      uint32_t color;
      uint32_t depth;
      uint32_t stencil;
    };

    inline Backbuffer();
    inline Backbuffer(int w, int h, uint32_t c, uint32_t d, uint32_t s);

    int width;
    int height;
    Format format;
  };

  // Construct and return an offscreen renderer.
  static GlesRenderer* create(GlesRenderer* sharedContext);

  // Returns the renderer's API.
  virtual Api* api() = 0;

  // Changes the back-buffer dimensions and format.
  virtual void setBackbuffer(Backbuffer backbuffer) = 0;

  // Makes the current renderer active.
  virtual void bind(bool resetViewportScissor) = 0;

  // Makes the current renderer inactive.
  virtual void unbind() = 0;

  // Returns the name of the renderer's created graphics context.
  virtual const char* name() = 0;

  // Returns the list of extensions that the renderer's graphics context
  // supports.
  virtual const char* extensions() = 0;

  // Returns the name of the vendor that has implemented the renderer's graphics
  // context.
  virtual const char* vendor() = 0;

  // Returns the version of the renderer's graphics context.
  virtual const char* version() = 0;

  virtual bool isValid() { return true; }

  // Creates an external image backed by the given texture.
  virtual void* createExternalImage(uint32_t texture) { return nullptr; }

  // Perform a call that acts as a frame delimiter, typically swapBuffers
  virtual bool frameDelimiter() { return true; }
};

inline GlesRenderer::Backbuffer::Format::Format()
    : color(0), depth(0), stencil(0) {}

inline GlesRenderer::Backbuffer::Format::Format(uint32_t c, uint32_t d,
                                                uint32_t s)
    : color(c), depth(d), stencil(s) {}

inline bool operator==(const GlesRenderer::Backbuffer::Format& lhs,
                       const GlesRenderer::Backbuffer::Format& rhs) {
  return lhs.color == rhs.color && lhs.depth == rhs.depth &&
         lhs.stencil == rhs.stencil;
}

inline GlesRenderer::Backbuffer::Backbuffer() : width(0), height(0) {}

inline GlesRenderer::Backbuffer::Backbuffer(int w, int h, uint32_t c,
                                            uint32_t d, uint32_t s)
    : width(w), height(h), format(c, d, s) {}

inline bool operator==(const GlesRenderer::Backbuffer& lhs,
                       const GlesRenderer::Backbuffer& rhs) {
  return lhs.width == rhs.width && lhs.height == rhs.height &&
         lhs.format == rhs.format;
}

}  // namespace gapir

#endif  // GAPIR_GLES_RENDERER_H
