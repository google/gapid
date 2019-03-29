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

#include "gl_lite.h"
#include "query.h"

#include "core/cc/assert.h"
#include "core/cc/get_gles_proc_address.h"
#include "core/cc/log.h"

#include <sstream>

namespace query {

const char* safe_string(void* x) {
  return (x != nullptr) ? reinterpret_cast<const char*>(x) : "";
}

void glDriver(device::OpenGLDriver* driver) {
  GLint major_version = 2;
  GLint minor_version = 0;
  GLint uniformbufferalignment = 1;
  GLint maxtransformfeedbackseparateattribs = 0;
  GLint maxtransformfeedbackinterleavedcomponents = 0;

  auto glGetIntegerv = reinterpret_cast<PFNGLGETINTEGERV>(
      core::GetGlesProcAddress("glGetIntegerv"));
  auto glGetError =
      reinterpret_cast<PFNGLGETERROR>(core::GetGlesProcAddress("glGetError"));
  auto glGetString =
      reinterpret_cast<PFNGLGETSTRING>(core::GetGlesProcAddress("glGetString"));
  auto glGetStringi = reinterpret_cast<PFNGLGETSTRINGI>(
      core::GetGlesProcAddress("glGetStringi"));

  GAPID_ASSERT(glGetError != nullptr);
  GAPID_ASSERT(glGetString != nullptr);

  glGetError();  // Clear error state.
  glGetIntegerv(GL_MAJOR_VERSION, &major_version);
  glGetIntegerv(GL_MINOR_VERSION, &minor_version);
  if (glGetError() != GL_NO_ERROR) {
    // GL_MAJOR_VERSION/GL_MINOR_VERSION were introduced in GLES 3.0,
    // so if the commands returned error we assume it is GLES 2.0.
    major_version = 2;
    minor_version = 0;
  }

  if (major_version >= 3) {
    GAPID_ASSERT(glGetIntegerv != nullptr);
    GAPID_ASSERT(glGetStringi != nullptr);

    glGetIntegerv(GL_UNIFORM_BUFFER_OFFSET_ALIGNMENT, &uniformbufferalignment);
    glGetIntegerv(GL_MAX_TRANSFORM_FEEDBACK_SEPARATE_ATTRIBS,
                  &maxtransformfeedbackseparateattribs);
    glGetIntegerv(GL_MAX_TRANSFORM_FEEDBACK_INTERLEAVED_COMPONENTS,
                  &maxtransformfeedbackinterleavedcomponents);

    int32_t c = 0;
    glGetIntegerv(GL_NUM_EXTENSIONS, &c);
    for (int32_t i = 0; i < c; i++) {
      driver->add_extensions(safe_string(glGetStringi(GL_EXTENSIONS, i)));
    }
  } else {
    std::string extensions = safe_string(glGetString(GL_EXTENSIONS));
    if (glGetError() == GL_NO_ERROR) {
      std::istringstream iss(extensions);
      std::string extension;
      while (std::getline(iss, extension, ' ')) {
        driver->add_extensions(extension);
      }
    }
  }

  driver->set_renderer(safe_string(glGetString(GL_RENDERER)));
  driver->set_vendor(safe_string(glGetString(GL_VENDOR)));
  driver->set_version(safe_string(glGetString(GL_VERSION)));
  driver->set_uniform_buffer_alignment(uniformbufferalignment);
  driver->set_max_transform_feedback_separate_attribs(
      maxtransformfeedbackseparateattribs);
  driver->set_max_transform_feedback_interleaved_components(
      maxtransformfeedbackinterleavedcomponents);

  glDriverPlatform(driver);
}

}  // namespace query
