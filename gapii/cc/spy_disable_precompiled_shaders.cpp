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

#include "gapii/cc/spy.h"
#include "gapii/cc/call_observer.h"

#include "gapis/api/gfxtrace.pb.h"

#include "core/cc/id.h"

// This file contains a number of GLES method 'overrides' to optionally lie to the
// application about the driver not supporting precompiled shaders or programs.

namespace {

const char* kOESGetProgramBinary  = "OES_get_program_binary";
const char* kReplacementExtension = "__GAPID_PCS_DISABLED__"; // Must be the same length as kOESGetProgramBinary

#define NELEM(x) (sizeof(x) / sizeof(x[0]))

// HACK: Workaround for devices that do not check the error status after calling
// glProgramBinary() or glProgramBinaryOES(). As the error is not checked, this
// can cause logic later on in the application to fail, sometimes leading to
// application termination.
// See https://github.com/google/gapid/issues/1456#issuecomment-349611106 for
// more information.
const core::Id kProgramHashesForNoError[] = {
    // 0xe14cc04bd723f9c2c46eeef948b08a379f090235
    {0xe1, 0x4c, 0xc0, 0x4b, 0xd7, 0x23, 0xf9, 0xc2, 0xc4, 0x6e,
     0xee, 0xf9, 0x48, 0xb0, 0x8a, 0x37, 0x9f, 0x09, 0x02, 0x35},

    // 0xc231a3a4b597f45244a4745fecdcba918bb8eacc
    {0xc2, 0x31, 0xa3, 0xa4, 0xb5, 0x97, 0xf4, 0x52, 0x44, 0xa4,
     0x74, 0x5f, 0xec, 0xdc, 0xba, 0x91, 0x8b, 0xb8, 0xea, 0xcc},

    // 0x55626b9bc73964f52fd5bcf6710659df97997d83
    {0x55, 0x62, 0x6b, 0x9b, 0xc7, 0x39, 0x64, 0xf5, 0x2f, 0xd5,
     0xbc, 0xf6, 0x71, 0x06, 0x59, 0xdf, 0x97, 0x99, 0x7d, 0x83},
};

bool shouldErrorForProgram(const core::Id& id) {
    for (int i = 0; i < NELEM(kProgramHashesForNoError); i++) {
         if (id == kProgramHashesForNoError[i]) {
            GAPID_WARNING("Not setting error for program with ID (blacklisted): %s", id.string().c_str());
            return false;
         }
    }
    GAPID_INFO("Program ID: %s", id.string().c_str());
    return true;
}

}  // anonymous namespace

using namespace gapii::GLenum;

namespace gapii {

void Spy::glProgramBinary(CallObserver* observer, uint32_t program, uint32_t binary_format, void* binary,
                          int32_t binary_size) {
    if (mDisablePrecompiledShaders) {
        GAPID_WARNING("glProgramBinary(%" PRIu32 ", 0x%X, %p, %" PRId32 ") "
                "called when precompiled shaders are disabled",
                program, binary_format, binary, binary_size);

        // GL_INVALID_ENUM is generated if binaryformat is not a supported format returned in
        // GL_SHADER_BINARY_FORMATS.
        auto id = core::Id::Hash(binary, binary_size);
        if (shouldErrorForProgram(id)) {
            setFakeGlError(observer, GL_INVALID_ENUM);
        }

        observer->encode(cmd::glProgramBinary{
            observer->getCurrentThread(), program, binary_format, binary, binary_size
        });

        observer->read(binary, binary_size);
        observer->observePending();

        observer->encodeAndDelete(new api::CmdCall);
    } else {
        GlesSpy::glProgramBinary(observer, program, binary_format, binary, binary_size);
    }
}

void Spy::glProgramBinaryOES(CallObserver* observer, uint32_t program, uint32_t binary_format, void* binary,
                             int32_t binary_size) {
    if (mDisablePrecompiledShaders) {
        GAPID_WARNING("glProgramBinaryOES(%" PRIu32 ", 0x%X, %p, %" PRId32 ") "
                "called when precompiled shaders are disabled",
                program, binary_format, binary, binary_size);

        // GL_INVALID_ENUM is generated if binaryformat is not a supported format returned in
        // GL_SHADER_BINARY_FORMATS.
        auto id = core::Id::Hash(binary, binary_size);
        if (shouldErrorForProgram(id)) {
            setFakeGlError(observer, GL_INVALID_ENUM);
        }

        observer->encode(cmd::glProgramBinaryOES{
            observer->getCurrentThread(), program, binary_format, binary, binary_size
        });

        observer->read(binary, binary_size);
        observer->observePending();

        observer->encodeAndDelete(new api::CmdCall);
    } else {
        GlesSpy::glProgramBinaryOES(observer, program, binary_format, binary, binary_size);
    }
}

void Spy::glShaderBinary(CallObserver* observer, int32_t count, uint32_t* shaders, uint32_t binary_format, void* binary,
                         int32_t binary_size) {
    if (mDisablePrecompiledShaders) {
        GAPID_WARNING("glShaderBinary(%" PRId32 ", %p, 0x%X, %p, %" PRId32 ") "
                "called when precompiled shaders are disabled",
                count, shaders, binary_format, binary,  binary_size);

        // GL_INVALID_ENUM is generated if binaryFormat is not a value recognized by the implementation.
        setFakeGlError(observer, GL_INVALID_ENUM);

        observer->encode(cmd::glShaderBinary{
            observer->getCurrentThread(), count, shaders, binary_format, binary, binary_size
        });

        observer->read(slice(shaders, (uint64_t)((GLsizei)(0)), (uint64_t)(count)));
        observer->read(slice(binary, (uint64_t)((GLsizei)(0)), (uint64_t)(binary_size)));
        observer->observePending();

        observer->encodeAndDelete(new api::CmdCall);
    } else {
        GlesSpy::glShaderBinary(observer, count, shaders, binary_format, binary, binary_size);
    }
}

void Spy::glGetInteger64v(CallObserver* observer, uint32_t param, int64_t* values) {
    if (mDisablePrecompiledShaders &&
        (param == GL_NUM_SHADER_BINARY_FORMATS || param == GL_NUM_PROGRAM_BINARY_FORMATS)) {
        values[0] = 0;

        observer->encode(cmd::glGetInteger64v{
            observer->getCurrentThread(), param, values
        });

        observer->encodeAndDelete(new api::CmdCall);

        observer->write(slice(values, 0, 1));
        observer->observePending();
    } else {
        GlesSpy::glGetInteger64v(observer, param, values);
    }
}

void Spy::glGetIntegerv(CallObserver* observer, uint32_t param, int32_t* values) {
    if (mDisablePrecompiledShaders &&
        (param == GL_NUM_SHADER_BINARY_FORMATS || param == GL_NUM_PROGRAM_BINARY_FORMATS)) {
        values[0] = 0;

        observer->encode(cmd::glGetIntegerv{
            observer->getCurrentThread(), param, values
        });

        observer->encodeAndDelete(new api::CmdCall);

        observer->write(slice(values, 0, 1));
        observer->observePending();
    } else {
        GlesSpy::glGetIntegerv(observer, param, values);
    }
}

GLubyte* Spy::glGetString(CallObserver* observer, uint32_t name) {
    if (mDisablePrecompiledShaders && name == GL_EXTENSIONS) {
        std::string list = reinterpret_cast<const char*>(GlesSpy::mImports.glGetString(name));
        size_t start = list.find(kOESGetProgramBinary);
        if (start != std::string::npos) {
            static std::string copy = list;
            copy.replace(start, strlen(kOESGetProgramBinary), kReplacementExtension);
            // TODO: write command.
            return reinterpret_cast<GLubyte*>(const_cast<char*>(copy.c_str()));
        }
    }
    return GlesSpy::glGetString(observer, name);
}

GLubyte* Spy::glGetStringi(CallObserver* observer, uint32_t name, GLuint index) {
    if (mDisablePrecompiledShaders && (name == GL_EXTENSIONS)) {
        const char* extension = reinterpret_cast<const char*>(GlesSpy::mImports.glGetStringi(name, index));
        if (strcmp(extension, kOESGetProgramBinary) == 0) {
            // TODO: write command.
            return reinterpret_cast<GLubyte*>(const_cast<char*>(kReplacementExtension));
        }
    }
    return GlesSpy::glGetStringi(observer, name, index);
}


} // namespace gapii
