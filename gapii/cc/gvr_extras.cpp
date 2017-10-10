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

namespace gapii {

bool GvrSpy::observeFramebuffer(CallObserver* observer, uint32_t* w, uint32_t* h, std::vector<uint8_t>* data) {
    auto spy = static_cast<Spy*>(this);

    auto contextIt = spy->Contexts.find(observer->getCurrentThread());
    if (contextIt == spy->Contexts.end() || contextIt->second == nullptr) {
        GAPID_WARNING("No GLES context bound to thread");
        return false;
    }
    auto context = contextIt->second;

    constexpr int frameIndex = 0; // TODO: other indices.
    auto framebufferId = mImports.gvr_frame_get_framebuffer_object(mLastSubmittedFrame, frameIndex);

    GAPID_INFO("frame=%p, framebufferId=%d", mLastSubmittedFrame, framebufferId);

    auto framebufferIt = context->mObjects.mFramebuffers.find(framebufferId);
    if (framebufferIt == context->mObjects.mFramebuffers.end()) {
        GAPID_WARNING("Framebuffer %d not found", framebufferId);
        return false;
    }
    auto framebuffer = framebufferIt->second;

    if (!spy->GlesSpy::getFramebufferAttachmentSize(observer, framebuffer.get(), w, h)) {
        GAPID_WARNING("Could not get the framebuffer size");
        return false;
    }

    auto gles = spy->GlesSpy::imports();

    // Get current framebuffer state.
    GLint prevFramebufferId = 0;
    GLint prevReadBuffer = 0;
    gles.glGetIntegerv(GLenum::GL_READ_FRAMEBUFFER_BINDING, &prevFramebufferId);
    gles.glGetIntegerv(GLenum::GL_READ_BUFFER, &prevReadBuffer);

    // Bind submitted framebuffer for reading.
    gles.glBindFramebuffer(GLenum::GL_READ_FRAMEBUFFER, framebufferId);
    gles.glReadBuffer(GLenum::GL_COLOR_ATTACHMENT0);

    // Do the read.
    data->resize((*w) * (*h) * 4);
    gles.glReadPixels(0, 0, int32_t(*w), int32_t(*h),
            GLenum::GL_RGBA, GLenum::GL_UNSIGNED_BYTE, data->data());

    // Restore previous state.
    gles.glBindFramebuffer(GLenum::GL_READ_FRAMEBUFFER, prevFramebufferId);
    gles.glReadBuffer(prevReadBuffer);
    return true;
}

}  // namespace gapii