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

#include "wgl.h"
#include "gapii/cc/gles_types.h"

#include <windows.h>

namespace gapii {
namespace wgl {

void getFramebufferInfo(void* hdcUntyped, FramebufferInfo& info) {
  const ::HDC hdc = reinterpret_cast<::HDC>(hdcUntyped);
  const HWND hWnd = WindowFromDC(hdc);
  RECT rect;
  GetClientRect(hWnd, &rect);
  info.width = rect.right - rect.left;
  info.height = rect.bottom - rect.top;

  PIXELFORMATDESCRIPTOR pfd;
  DescribePixelFormat(hdc, GetPixelFormat(hdc), sizeof(pfd), &pfd);

  int r = pfd.cRedBits;
  int g = pfd.cGreenBits;
  int b = pfd.cBlueBits;
  int a = pfd.cAlphaBits;

  if (r == 8 && g == 8 && b == 8 && a == 8) {
    info.colorFormat = GLenum::GL_RGBA8;
  } else if (r == 4 && g == 4 && b == 4 && a == 4) {
    info.colorFormat = GLenum::GL_RGBA4;
  } else if (r == 5 && g == 5 && b == 5 && a == 1) {
    info.colorFormat = GLenum::GL_RGB5_A1;
  } else if (r == 5 && g == 6 && b == 5 && a == 0) {
    info.colorFormat = GLenum::GL_RGB565;
  } else {
    info.colorFormat = GLenum::GL_RGBA8;
  }

  // No options for these yet.
  info.stencilFormat = GLenum::GL_DEPTH24_STENCIL8;
  info.depthFormat = GLenum::GL_DEPTH24_STENCIL8;
}

}  // namespace wgl
}  // namespace gapii
