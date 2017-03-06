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

#include "gapii/cc/gles_spy.h"
#include "gapii/cc/call_observer.h"

namespace gapii {

template<typename T> T inline min(T a, T b) { return (a < b) ? a : b; }
template<typename T> T inline max(T a, T b) { return (a > b) ? a : b; }

// Externs not implemented in GAPII.
void GlesSpy::mapMemory(CallObserver*, Slice<uint8_t>) {}
void GlesSpy::unmapMemory(CallObserver*, Slice<uint8_t>) {}
MsgID GlesSpy::newMsg(CallObserver*, uint32_t, const char*) { return 0; }
void GlesSpy::addTag(CallObserver*, uint32_t, const char*) {}

u32Limits GlesSpy::IndexLimits(CallObserver*, uint8_t* indices, uint32_t indices_type, uint32_t offset, uint32_t count) {
    uint32_t low = ~(uint32_t)0;
    uint32_t high = 0;
    switch (indices_type) {
        case gapii::GLenum::GL_UNSIGNED_BYTE: {
            const uint8_t* p = reinterpret_cast<const uint8_t*>(&indices[offset]);
            for (uint32_t i = 0; i < count; i++) {
                low = min<uint32_t>(low, p[i]);
                high = max<uint32_t>(high, p[i]);
            }
            break;
        }
        case gapii::GLenum::GL_UNSIGNED_SHORT: {
            const uint16_t* p = reinterpret_cast<const uint16_t*>(&indices[offset]);
            for (uint32_t i = 0; i < count; i++) {
                low = min<uint32_t>(low, p[i]);
                high = max<uint32_t>(high, p[i]);
            }
            break;
        }
        case gapii::GLenum::GL_UNSIGNED_INT: {
            const uint32_t* p = reinterpret_cast<const uint32_t*>(&indices[offset]);
            for (uint32_t i = 0; i < count; i++) {
                low = min<uint32_t>(low, p[i]);
                high = max<uint32_t>(high, p[i]);
            }
            break;
        }
    }
    return u32Limits(low, high);
}

void GlesSpy::onGlError(CallObserver* observer, GLenum_Error err) {
    const char* current_cmd_name = observer->getCurrentCommandName();
    switch (err) {
        case GLenum::GL_INVALID_ENUM:
            GAPID_WARNING("Error calling %s: GL_INVALID_ENUM", current_cmd_name);
            break;
        case GLenum::GL_INVALID_VALUE:
            GAPID_WARNING("Error calling %s: GL_INVALID_VALUE", current_cmd_name);
            break;
        case GLenum::GL_INVALID_OPERATION:
            GAPID_WARNING("Error calling %s: GL_INVALID_OPERATION", current_cmd_name);
            break;
        case GLenum::GL_STACK_OVERFLOW:
            GAPID_WARNING("Error calling %s: GL_STACK_OVERFLOW", current_cmd_name);
            break;
        case GLenum::GL_STACK_UNDERFLOW:
            GAPID_WARNING("Error calling %s: GL_STACK_UNDERFLOW", current_cmd_name);
            break;
        case GLenum::GL_OUT_OF_MEMORY:
            GAPID_WARNING("Error calling %s: GL_OUT_OF_MEMORY", current_cmd_name);
            break;
        case GLenum::GL_INVALID_FRAMEBUFFER_OPERATION:
            GAPID_WARNING("Error calling %s: GL_INVALID_FRAMEBUFFER_OPERATION", current_cmd_name);
            break;
        case GLenum::GL_CONTEXT_LOST:
            GAPID_WARNING("Error calling %s: GL_CONTEXT_LOST", current_cmd_name);
            break;
        default:
            GAPID_WARNING("Error calling %s: %d", current_cmd_name, err);
            break;
    }

    // Set error only if has not been previously set
    if (observer->getError() == GLenum::GL_NO_ERROR) {
        observer->setError(err);
    }
}

}
