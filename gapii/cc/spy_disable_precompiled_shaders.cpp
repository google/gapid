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

#include "gapii/cc/call_observer.h"
#include "gapii/cc/spy.h"

#include "gapis/api/gfxtrace.pb.h"

#include "core/cc/id.h"

// This file contains a number of GLES method 'overrides' to optionally lie to
// the application about the driver not supporting precompiled shaders or
// programs.

namespace {

#define NELEM(x) (sizeof(x) / sizeof(x[0]))

const char* kProgramBinaryExtensions[] = {
    "OES_get_program_binary",
    "ARB_get_program_binary",
};

const char* kProgramBinaryReplacements[] = {
    // These *must* match the length of each string in kProgramBinaryExtensions
    "__GAPID_PCS_DISABLED__",
    "__GAPID_PCS_DISABLED__",
};

static_assert(
    NELEM(kProgramBinaryExtensions) == NELEM(kProgramBinaryReplacements),
    "length of kProgramBinaryExtensions must match kProgramBinaryReplacements");

// HACK: Workaround for devices that do not check the error status after calling
// glProgramBinary() or glProgramBinaryOES(). As the error is not checked, this
// can cause logic later on in the application to fail, sometimes leading to
// application termination.
// See https://github.com/google/gapid/issues/1456#issuecomment-349611106 for
// more information.
const core::Id kProgramHashesForNoError[] = {
    // https://github.com/google/gapid/issues/1456
    // 0xe14cc04bd723f9c2c46eeef948b08a379f090235
    {{0xe1, 0x4c, 0xc0, 0x4b, 0xd7, 0x23, 0xf9, 0xc2, 0xc4, 0x6e,
      0xee, 0xf9, 0x48, 0xb0, 0x8a, 0x37, 0x9f, 0x09, 0x02, 0x35}},

    // 0xc231a3a4b597f45244a4745fecdcba918bb8eacc
    {{0xc2, 0x31, 0xa3, 0xa4, 0xb5, 0x97, 0xf4, 0x52, 0x44, 0xa4,
      0x74, 0x5f, 0xec, 0xdc, 0xba, 0x91, 0x8b, 0xb8, 0xea, 0xcc}},

    // 0x55626b9bc73964f52fd5bcf6710659df97997d83
    {{0x55, 0x62, 0x6b, 0x9b, 0xc7, 0x39, 0x64, 0xf5, 0x2f, 0xd5,
      0xbc, 0xf6, 0x71, 0x06, 0x59, 0xdf, 0x97, 0x99, 0x7d, 0x83}},

    // https://github.com/google/gapid/issues/1525
    // 0xc6b9efad92959f4af5f6fb67a21d94b22f746838
    {{0xc6, 0xb9, 0xef, 0xad, 0x92, 0x95, 0x9f, 0x4a, 0xf5, 0xf6,
      0xfb, 0x67, 0xa2, 0x1d, 0x94, 0xb2, 0x2f, 0x74, 0x68, 0x38}},
};

bool shouldErrorForProgram(const core::Id& id) {
  for (size_t i = 0; i < NELEM(kProgramHashesForNoError); i++) {
    if (id == kProgramHashesForNoError[i]) {
      GAPID_WARNING("Not setting error for program with ID (blacklisted): %s",
                    id.string().c_str());
      return false;
    }
  }
  GAPID_INFO("Program ID: %s", id.string().c_str());
  return true;
}

}  // anonymous namespace

using namespace gapii::GLenum;

namespace gapii {

void Spy::glProgramBinary(CallObserver* observer, uint32_t program,
                          uint32_t binary_format, const void* binary,
                          int32_t binary_size) {
  if (mDisablePrecompiledShaders) {
    GAPID_WARNING("glProgramBinary(%" PRIu32 ", 0x%X, %p, %" PRId32
                  ") "
                  "called when precompiled shaders are disabled",
                  program, binary_format, binary, binary_size);

    // GL_INVALID_ENUM is generated if binaryformat is not a supported format
    // returned in GL_SHADER_BINARY_FORMATS.
    auto id = core::Id::Hash(binary, binary_size);
    if (shouldErrorForProgram(id)) {
      setFakeGlError(observer, GL_INVALID_ENUM);
    }

    observer->enter(cmd::glProgramBinary{observer->getCurrentThread(), program,
                                         binary_format, binary, binary_size});

    observer->read(binary, binary_size);
    observer->observePending();
    api::CmdCall call;
    observer->encode_message(&call);
    observer->exit();
  } else {
    GlesSpy::glProgramBinary(observer, program, binary_format, binary,
                             binary_size);
  }
}

void Spy::glProgramBinaryOES(CallObserver* observer, uint32_t program,
                             uint32_t binary_format, const void* binary,
                             int32_t binary_size) {
  if (mDisablePrecompiledShaders) {
    GAPID_WARNING("glProgramBinaryOES(%" PRIu32 ", 0x%X, %p, %" PRId32
                  ") "
                  "called when precompiled shaders are disabled",
                  program, binary_format, binary, binary_size);

    // GL_INVALID_ENUM is generated if binaryformat is not a supported format
    // returned in GL_SHADER_BINARY_FORMATS.
    auto id = core::Id::Hash(binary, binary_size);
    if (shouldErrorForProgram(id)) {
      setFakeGlError(observer, GL_INVALID_ENUM);
    }

    observer->enter(cmd::glProgramBinaryOES{observer->getCurrentThread(),
                                            program, binary_format, binary,
                                            binary_size});

    observer->read(binary, binary_size);
    observer->observePending();

    api::CmdCall call;
    observer->encode_message(&call);
    observer->exit();
  } else {
    GlesSpy::glProgramBinaryOES(observer, program, binary_format, binary,
                                binary_size);
  }
}

void Spy::glShaderBinary(CallObserver* observer, int32_t count,
                         const uint32_t* shaders, uint32_t binary_format,
                         const void* binary, int32_t binary_size) {
  if (mDisablePrecompiledShaders) {
    GAPID_WARNING("glShaderBinary(%" PRId32 ", %p, 0x%X, %p, %" PRId32
                  ") "
                  "called when precompiled shaders are disabled",
                  count, shaders, binary_format, binary, binary_size);

    // GL_INVALID_ENUM is generated if binaryFormat is not a value recognized by
    // the implementation.
    setFakeGlError(observer, GL_INVALID_ENUM);

    observer->enter(cmd::glShaderBinary{observer->getCurrentThread(), count,
                                        shaders, binary_format, binary,
                                        binary_size});

    observer->read(slice(shaders, (uint64_t)((GLsizei)(0)), (uint64_t)(count)));
    observer->read(
        slice(binary, (uint64_t)((GLsizei)(0)), (uint64_t)(binary_size)));
    observer->observePending();

    api::CmdCall call;
    observer->encode_message(&call);
    observer->exit();
  } else {
    GlesSpy::glShaderBinary(observer, count, shaders, binary_format, binary,
                            binary_size);
  }
}

void Spy::glGetInteger64v(CallObserver* observer, uint32_t param,
                          int64_t* values) {
  if (mDisablePrecompiledShaders && (param == GL_NUM_SHADER_BINARY_FORMATS ||
                                     param == GL_NUM_PROGRAM_BINARY_FORMATS)) {
    values[0] = 0;

    observer->enter(
        cmd::glGetInteger64v{observer->getCurrentThread(), param, values});

    api::CmdCall call;
    observer->encode_message(&call);

    observer->write(slice(values, 0, 1));
    observer->observePending();
    observer->exit();
  } else {
    GlesSpy::glGetInteger64v(observer, param, values);
  }
}

void Spy::glGetIntegerv(CallObserver* observer, uint32_t param,
                        int32_t* values) {
  if (mDisablePrecompiledShaders && (param == GL_NUM_SHADER_BINARY_FORMATS ||
                                     param == GL_NUM_PROGRAM_BINARY_FORMATS)) {
    values[0] = 0;

    observer->enter(
        cmd::glGetIntegerv{observer->getCurrentThread(), param, values});

    api::CmdCall call;
    observer->encode_message(&call);

    observer->write(slice(values, 0, 1));
    observer->observePending();
    observer->exit();
  } else {
    GlesSpy::glGetIntegerv(observer, param, values);
  }
}

const GLubyte* Spy::glGetString(CallObserver* observer, uint32_t name) {
  if (mDisablePrecompiledShaders && name == GL_EXTENSIONS) {
    if (auto exts = reinterpret_cast<const char*>(
            GlesSpy::mImports.glGetString(name))) {
      std::string list = reinterpret_cast<const char*>(exts);
      for (size_t i = 0; i < NELEM(kProgramBinaryExtensions); i++) {
        size_t start = list.find(kProgramBinaryExtensions[i]);
        if (start != std::string::npos) {
          static std::string copy = list;
          copy.replace(start, strlen(kProgramBinaryExtensions[i]),
                       kProgramBinaryReplacements[i]);
          // TODO: write command.
          return reinterpret_cast<GLubyte*>(const_cast<char*>(copy.c_str()));
        }
      }
    }
  }
  return GlesSpy::glGetString(observer, name);
}

const GLubyte* Spy::glGetStringi(CallObserver* observer, uint32_t name,
                                 GLuint index) {
  if (mDisablePrecompiledShaders && (name == GL_EXTENSIONS)) {
    const char* extension = reinterpret_cast<const char*>(
        GlesSpy::mImports.glGetStringi(name, index));
    for (size_t i = 0; i < NELEM(kProgramBinaryExtensions); i++) {
      if (strcmp(extension, kProgramBinaryExtensions[i]) == 0) {
        // TODO: write command.
        return reinterpret_cast<GLubyte*>(
            const_cast<char*>(kProgramBinaryReplacements[i]));
      }
    }
  }
  return GlesSpy::glGetStringi(observer, name, index);
}

}  // namespace gapii
