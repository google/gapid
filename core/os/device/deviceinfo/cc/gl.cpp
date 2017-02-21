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

#include "query.h"

#include "core/cc/target.h"

#if TARGET_OS == GAPID_OS_OSX
#   import <OpenGL/gl3.h>
#   define HAS_MAJOR_VERSION_3 1
#elif TARGET_OS == GAPID_OS_ANDROID
#   include <GLES2/gl2.h>
#   define HAS_MAJOR_VERSION_3 0
#elif TARGET_OS == GAPID_OS_LINUX
#   define GL_GLEXT_PROTOTYPES 1
#   include <GL/gl.h>
#   include <GL/glext.h>
#   define HAS_MAJOR_VERSION_3 1
#elif TARGET_OS == GAPID_OS_WINDOWS
#   define WINGDIAPI
#   define APIENTRY
#   include <GL/gl.h>
#endif

#include <sstream>

namespace query {

void glDriver(device::OpenGLDriver* driver) {
    GLint major_version = 2;
    GLint minor_version = 0;
    GLint uniformbufferalignment = 1;

#if HAS_MAJOR_VERSION_3
    glGetIntegerv(GL_UNIFORM_BUFFER_OFFSET_ALIGNMENT, &uniformbufferalignment);

    glGetError();  // Clear error state.
    glGetIntegerv(GL_MAJOR_VERSION, &major_version);
    glGetIntegerv(GL_MINOR_VERSION, &minor_version);
    if (glGetError() != GL_NO_ERROR) {
      // GL_MAJOR_VERSION/GL_MINOR_VERSION were introduced in GLES 3.0,
      // so if the commands returned error we assume it is GLES 2.0.
      major_version = 2;
      minor_version = 0;
    }

    if (major_version > 3) {
        int32_t c;
        glGetIntegerv(GL_NUM_EXTENSIONS, &c);
        for (int32_t i = 0; i < c; i++) {
            if (glGetError() == GL_NO_ERROR) {
                driver->add_extensions(reinterpret_cast<const char*>(glGetStringi(GL_EXTENSIONS, i)));
            }
        }
    } else
#endif  // HAS_MAJOR_VERSION_3
    {
        std::string extensions = reinterpret_cast<const char*>(glGetString(GL_EXTENSIONS));
        if (glGetError() == GL_NO_ERROR) {
            std::istringstream iss(extensions);
            std::string extension;
            while (std::getline(iss, extension, ' ')) {
                driver->add_extensions(extension);
            }
        }
    }

    driver->set_renderer((const char*)(glGetString(GL_RENDERER)));
    driver->set_vendor((const char*)(glGetString(GL_VENDOR)));
    driver->set_version((const char*)(glGetString(GL_VERSION)));
    driver->set_uniformbufferalignment(uniformbufferalignment);
}

}  // namespace query