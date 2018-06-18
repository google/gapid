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

#ifndef CORE_GL_FORMATS_H
#define CORE_GL_FORMATS_H

#include <stdint.h>

namespace core {
namespace gl {

const uint32_t GL_RGB8 = 0x00008051;
const uint32_t GL_RGBA8 = 0x00008058;
const uint32_t GL_RGB565 = 0x00008D62;
const uint32_t GL_DEPTH_COMPONENT16 = 0x000081A5;
const uint32_t GL_DEPTH_COMPONENT24 = 0x000081A6;
const uint32_t GL_DEPTH_COMPONENT32F = 0x00008CAC;
const uint32_t GL_DEPTH32F_STENCIL8 = 0x00008CAD;
const uint32_t GL_DEPTH24_STENCIL8 = 0x000088F0;
const uint32_t GL_STENCIL_INDEX8 = 0x00008D48;

// getColorFormat gets the color buffer format given the number of bits for the
// red, green, blue and alpha channels. Returns true on success, or false if
// there is no format for the given bit combination.
bool getColorFormat(int r, int g, int b, int a, uint32_t& format);

// getColorBits gets the number of bits for the red, green, blue and alpha
// channels for the given format. Returns true on success, or false if the
// format is not recognised.
bool getColorBits(uint32_t format, int& r, int& g, int& b, int& a);

// getDepthStencilFormat gets the depth and stencil buffer formats given the
// number of bits for the depth and stencil channels. Returns true on success,
// or false if there is no format combination for the given bits.
bool getDepthStencilFormat(int d, int s, uint32_t& depth, uint32_t& stencil);

// getDepthBits gets the number of bits for the depth channel for the given
// depth format. Returns true on success, or false if the format is not
// recognised.
bool getDepthBits(uint32_t format, int& d);

// getStencilBits gets the number of bits for the stencil channel for the given
// stencil format. Returns true on success, or false if the format is not
// recognised.
bool getStencilBits(uint32_t format, int& s);

}  // namespace gl
}  // namespace core

#endif  // CORE_GL_FORMATS_H
