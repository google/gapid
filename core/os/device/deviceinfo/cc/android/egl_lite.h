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

#ifndef GAPID_CORE_OS_DEVICEINFO_ANDROID_EGL_LITE
#define GAPID_CORE_OS_DEVICEINFO_ANDROID_EGL_LITE

typedef int EGLBoolean;
typedef int EGLint;
typedef unsigned int EGLenum;
typedef void* EGLConfig;
typedef void* EGLContext;
typedef void* EGLDisplay;
typedef void* EGLNativeDisplayType;
typedef void* EGLSurface;

const EGLint EGL_PBUFFER_BIT = 0x0001;
const EGLint EGL_OPENGL_ES2_BIT = 0x0004;

const EGLint EGL_TRUE = 0x0001;
const EGLint EGL_SUCCESS = 0x3000;
const EGLint EGL_BUFFER_SIZE = 0x3020;
const EGLint EGL_ALPHA_SIZE = 0x3021;
const EGLint EGL_BLUE_SIZE = 0x3022;
const EGLint EGL_GREEN_SIZE = 0x3023;
const EGLint EGL_RED_SIZE = 0x3024;
const EGLint EGL_DEPTH_SIZE = 0x3025;
const EGLint EGL_STENCIL_SIZE = 0x3026;
const EGLint EGL_CONFIG_ID = 0x3028;
const EGLint EGL_SURFACE_TYPE = 0x3033;
const EGLint EGL_NONE = 0x3038;
const EGLint EGL_RENDERABLE_TYPE = 0x3040;
const EGLint EGL_EXTENSIONS = 0x3055;
const EGLint EGL_HEIGHT = 0x3056;
const EGLint EGL_WIDTH = 0x3057;
const EGLint EGL_SWAP_BEHAVIOR = 0x3093;
const EGLint EGL_BUFFER_PRESERVED = 0x3094;
const EGLint EGL_CONTEXT_CLIENT_VERSION = 0x3098;
const EGLint EGL_OPENGL_ES_API = 0x30A0;

const EGLNativeDisplayType EGL_DEFAULT_DISPLAY = nullptr;
const EGLContext EGL_NO_CONTEXT = 0;
const EGLDisplay EGL_NO_DISPLAY = 0;
const EGLSurface EGL_NO_SURFACE = 0;

typedef EGLBoolean (*PFNEGLBINDAPI)(EGLenum api);
typedef EGLBoolean (*PFNEGLCHOOSECONFIG)(EGLDisplay display,
                                         const EGLint* attrib_list,
                                         EGLConfig* configs, EGLint config_size,
                                         EGLint* num_config);
typedef EGLBoolean (*PFNEGLDESTROYCONTEXT)(EGLDisplay display,
                                           EGLContext context);
typedef EGLBoolean (*PFNEGLDESTROYSURFACE)(EGLDisplay display,
                                           EGLSurface surface);
typedef EGLBoolean (*PFNEGLINITIALIZE)(EGLDisplay dpy, EGLint* major,
                                       EGLint* minor);
typedef EGLBoolean (*PFNEGLMAKECURRENT)(EGLDisplay display, EGLSurface draw,
                                        EGLSurface read, EGLContext context);
typedef EGLBoolean (*PFNEGLTERMINATE)(EGLDisplay display);
typedef EGLContext (*PFNEGLCREATECONTEXT)(EGLDisplay display, EGLConfig config,
                                          EGLContext share_context,
                                          const EGLint* attrib_list);
typedef EGLDisplay (*PFNEGLGETDISPLAY)(EGLNativeDisplayType native_display);
typedef EGLint (*PFNEGLGETERROR)();
typedef EGLSurface (*PFNEGLCREATEPBUFFERSURFACE)(EGLDisplay display,
                                                 EGLConfig config,
                                                 const EGLint* attrib_list);
typedef const char* (*PFNEGLQUERYSTRING)(EGLDisplay display, EGLint name);
typedef EGLBoolean (*PFNEGLRELEASETHREAD)();

#endif  // GAPID_CORE_OS_DEVICEINFO_ANDROID_EGL_LITE
