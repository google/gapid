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

#include "formats.h"

#include "core/cc/log.h"

namespace core {
namespace gl {

bool getColorFormat(int r, int g, int b, int a, uint32_t& format) {
  if (r == 8 && g == 8 && b == 8 && a == 8) {
    format = GL_RGBA8;
    return true;
  } else if (r == 8 && g == 8 && b == 8 && a == 0) {
    format = GL_RGB8;
    return true;
  } else if (r == 5 && g == 6 && b == 5 && a == 0) {
    format = GL_RGB565;
    return true;
  }
  return false;  // Not a recognised combination.
}

bool getColorBits(uint32_t format, int& r, int& g, int& b, int& a) {
  switch (format) {
    case GL_RGBA8:
      r = 8;
      g = 8;
      b = 8;
      a = 8;
      return true;
    case GL_RGB8:
      r = 8;
      g = 8;
      b = 8;
      a = 0;
      return true;
    case GL_RGB565:
      r = 5;
      g = 6;
      b = 5;
      a = 0;
      return true;
  }
  return false;  // Not a recognised combination.
}

// See:
// https://www.khronos.org/opengles/sdk/docs/man3/docbook4/xhtml/glRenderbufferStorage.xml
bool getDepthStencilFormat(int d, int s, uint32_t& depth, uint32_t& stencil) {
  depth = 0;
  stencil = 0;

  if (d == 0 && s == 0) {
    return true;  // No depth, no stencil.
  }

  switch (s) {
    case 0:
      switch (d) {
        case 16:
          depth = GL_DEPTH_COMPONENT16;
          return true;
        case 24:
          depth = GL_DEPTH_COMPONENT24;
          return true;
        case 32:
          depth = GL_DEPTH_COMPONENT32F;
          return true;
      }
      break;
    case 8:
      switch (d) {
        case 0:
          stencil = GL_STENCIL_INDEX8;
          return true;
        case 24:
          depth = GL_DEPTH24_STENCIL8;
          stencil = GL_DEPTH24_STENCIL8;
          return true;
        case 32:
          depth = GL_DEPTH32F_STENCIL8;
          stencil = GL_DEPTH32F_STENCIL8;
          return true;
      }
      break;
  }
  return false;  // Not a recognised combination.
}

bool getDepthBits(uint32_t format, int& d) {
  d = 0;
  switch (format) {
    case 0:
      return true;
    case GL_DEPTH_COMPONENT16:
      d = 16;
      return true;
    case GL_DEPTH_COMPONENT24:
    case GL_DEPTH24_STENCIL8:
      d = 24;
      return true;
    case GL_DEPTH_COMPONENT32F:
    case GL_DEPTH32F_STENCIL8:
      d = 32;
      return true;
  }
  return false;  // Not a recognised combination.
}

bool getStencilBits(uint32_t format, int& s) {
  s = 0;
  switch (format) {
    case 0:
      return true;
    case GL_STENCIL_INDEX8:
    case GL_DEPTH24_STENCIL8:
    case GL_DEPTH32F_STENCIL8:
      s = 8;
      return true;
  }
  return false;  // Not a recognised combination.
}

}  // namespace gl
}  // namespace core
