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

#ifndef CORE_GL_VERSIONS_H
#define CORE_GL_VERSIONS_H

namespace core {
namespace gl {

// Version represents a single major.minor version of an OpenGL context.
struct Version {
  int major;
  int minor;
};

// sVersionSearchOrder is an preference-ordered list of OpenGL context versions
// that should be searched in order to pick the most recent version of OpenGL.
// The driver is always free to return a newer version, however some
// implementations will return the precise version requested, so if we request
// 3.2, we would get 3.2 even if a new version is available.
static const Version sVersionSearchOrder[] = {
    // clang-format off
    {4, 5}, // Compatible with OpenGL ES 3.1
    {4, 4},
    {4, 3}, // Compatible with OpenGL ES 3.0
    {4, 2},
    {4, 1}, // Compatible with OpenGL ES 2.0
    {4, 0},
    {3, 3},
    {3, 2}, // Introduces core profile
    // clang-format on
};

}  // namespace gl
}  // namespace core

#endif  // CORE_GL_VERSIONS_H
