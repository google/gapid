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

#include <jni.h>
#include <GL/glx.h>

#include <stdio.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * Native JNI helper function to create an OpenGL 3.2 Core context.
 * This is done in native code to catch the X11 error, when creating
 * the context, to prevent it from taking down the whole process.
 */
JNIEXPORT jlong JNICALL Java_com_google_gapid_glcanvas_GlCanvas_createContext0(
    JNIEnv* env, jlong clazz, Display* display, GLXFBConfig config) {
  PFNGLXCREATECONTEXTATTRIBSARBPROC glXCreateContextAttribsARB =
      (PFNGLXCREATECONTEXTATTRIBSARBPROC)glXGetProcAddress(
        (const GLubyte*)"glXCreateContextAttribsARB");
  if (glXCreateContextAttribsARB == nullptr) {
    // This shouldn't really happen, as we check this Java side.
    return 0;
  }

  const int attr[] = {
      GLX_RENDER_TYPE, GLX_RGBA_TYPE,
      GLX_CONTEXT_MAJOR_VERSION_ARB, 3,
      GLX_CONTEXT_MINOR_VERSION_ARB, 2,
      GLX_CONTEXT_PROFILE_MASK_ARB, GLX_CONTEXT_CORE_PROFILE_BIT_ARB,
      None,
  };
  auto oldHandler = XSetErrorHandler([](Display*, XErrorEvent*)->int{ return 0; });
  auto context = glXCreateContextAttribsARB(display, config, 0, true, attr);
  XSetErrorHandler(oldHandler);

  return (jlong)context;
}

#ifdef __cplusplus
}
#endif
