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

#ifndef GAPID_CORE_OS_DEVICEINFO_GL_LITE
#define GAPID_CORE_OS_DEVICEINFO_GL_LITE

typedef unsigned int GLenum;
typedef unsigned char GLubyte;
typedef unsigned int GLuint;
typedef int GLint;

const GLint GL_UNIFORM_BUFFER_OFFSET_ALIGNMENT = 0x8A34;

const GLint GL_NO_ERROR = 0x0000;
const GLint GL_VENDOR = 0x1F00;
const GLint GL_RENDERER = 0x1F01;
const GLint GL_VERSION = 0x1F02;
const GLint GL_EXTENSIONS = 0x1F03;
const GLint GL_MINOR_VERSION = 0x821C;
const GLint GL_MAJOR_VERSION = 0x821B;
const GLint GL_NUM_EXTENSIONS = 0x821D;
const GLint GL_MAX_TRANSFORM_FEEDBACK_INTERLEAVED_COMPONENTS = 0x8C8A;
const GLint GL_MAX_TRANSFORM_FEEDBACK_SEPARATE_ATTRIBS = 0x8C8B;

typedef void (*PFNGLGETINTEGERV)(GLenum param, GLint* values);
typedef GLenum (*PFNGLGETERROR)();
typedef GLubyte* (*PFNGLGETSTRING)(GLenum param);
typedef GLubyte* (*PFNGLGETSTRINGI)(GLenum name, GLuint index);

#endif  // GAPID_CORE_OS_DEVICEINFO_GL_LITE
