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
#include "gapii/cc/gles_types.h"
#include "gapii/cc/spy.h"
#include "gapii/cc/state_serializer.h"

#include "gapis/api/gles/gles_pb/extras.pb.h"

namespace {
using namespace gapii;
using namespace gapii::GLenum;
using namespace gapii::GLbitfield;
using namespace gapii::EGLenum;

struct ImageData {
  std::unique_ptr<std::vector<uint8_t>> data;
  uint32_t width;
  uint32_t height;
  GLint dataFormat;
  GLint dataType;
};

class TempObject {
 public:
  TempObject(uint64_t id, const std::function<void()>& deleteId) : mId(id), mDeleteId(deleteId) {}
  uint64_t id() { return mId; }
  ~TempObject() { mDeleteId(); }
 private:
  uint64_t mId;
  std::function<void()> mDeleteId;
};

TempObject CreateAndBindFramebuffer(const GlesImports& imports, uint32_t target) {
  GLuint fb = 0;
  imports.glGenFramebuffers(1, &fb);
  imports.glBindFramebuffer(target, fb);
  return TempObject(fb, [=]{
    GLuint id = fb;
    imports.glDeleteFramebuffers(1, &id);
  });
}

TempObject CreateAndBindTexture2D(const GlesImports& imports, GLint w, GLint h, uint32_t sizedFormat) {
  GLuint tex = 0;
  imports.glGenTextures(1, &tex);
  imports.glBindTexture(GL_TEXTURE_2D, tex);
  imports.glTexStorage2D(GL_TEXTURE_2D, 1, sizedFormat, w, h);
  return TempObject(tex, [=]{
    GLuint id = tex;
    imports.glDeleteTextures(1, &id);
  });
}

TempObject CreateAndBindTextureExternal(const GlesImports& imports, EGLImageKHR handle) {
  GLuint tex = 0;
  imports.glGenTextures(1, &tex);
  imports.glBindTexture(GL_TEXTURE_EXTERNAL_OES, tex);
  imports.glEGLImageTargetTexture2DOES(GL_TEXTURE_EXTERNAL_OES, handle);
  imports.glTexParameteri(GL_TEXTURE_EXTERNAL_OES, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
  imports.glTexParameteri(GL_TEXTURE_EXTERNAL_OES, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
  return TempObject(tex, [=]{
    GLuint id = tex;
    imports.glDeleteTextures(1, &id);
  });
}

// Creates temporary GL context which shares objects with the given application context.
// This makes it easier to do a lot of work without worrying about corrupting the state.
// For example, calling glGetError would be otherwise technically invalid without hacks.
// As the moment this context is short lived but long term we might consider caching it.
TempObject CreateAndBindContext(const GlesImports& imports, EGLContext sharedContext) {
  // Save old state.
  auto display = imports.eglGetCurrentDisplay();
  auto draw = imports.eglGetCurrentSurface(EGL_DRAW);
  auto read = imports.eglGetCurrentSurface(EGL_READ);
  auto oldCtx = imports.eglGetCurrentContext();

  // Find an EGL config.
  EGLConfig cfg;
  EGLint cfgAttribs[] = { EGL_RENDERABLE_TYPE, EGL_OPENGL_ES2_BIT, EGL_NONE };
  int one = 1;
  if (imports.eglChooseConfig(display, cfgAttribs, &cfg, 1, &one) == EGL_FALSE || one != 1) {
    GAPID_FATAL("MEC: Failed to choose EGL config");
  }

  // Create an EGL context.
  EGLContext ctx;
  EGLint ctxAttribs[] = { EGL_CONTEXT_CLIENT_VERSION, 2, EGL_NONE };
  if ((ctx = imports.eglCreateContext(display, cfg, sharedContext, ctxAttribs)) == nullptr) {
    GAPID_FATAL("MEC: Failed to create EGL context");
  }

  // Create an EGL surface.
  EGLSurface surface;
  EGLint surfaceAttribs[] = { EGL_WIDTH, 16, EGL_HEIGHT, 16, EGL_NONE };
  if ((surface = imports.eglCreatePbufferSurface(display, cfg, surfaceAttribs)) == nullptr) {
    GAPID_FATAL("MEC: Failed to create EGL surface");
  }

  // Bind the EGL context.
  if (imports.eglMakeCurrent(display, surface, surface, ctx) == EGL_FALSE) {
    GAPID_FATAL("MEC: Failed to bind new EGL context");
  }

  // Setup desirable default state for reading data.
	imports.glPixelStorei(GL_PACK_ALIGNMENT, 1);
	imports.glPixelStorei(GL_PACK_ROW_LENGTH, 0);
	imports.glPixelStorei(GL_PACK_SKIP_PIXELS, 0);
	imports.glPixelStorei(GL_PACK_SKIP_ROWS, 0);

  return TempObject(reinterpret_cast<uint64_t>(ctx), [=]{
    if (imports.eglMakeCurrent(display, draw, read, oldCtx) == EGL_FALSE) {
      GAPID_FATAL("MEC: Failed to restore old EGL context");
    }
    imports.eglDestroySurface(display, surface);
    imports.eglDestroyContext(display, ctx);
  });
}

void DrawTexturedQuad(const GlesImports& imports, uint32_t textureTarget, GLsizei w, GLsizei h) {
  GLint err;

  std::string vsSource;
  vsSource += "precision highp float;\n";
  vsSource += "attribute vec2 position;\n";
  vsSource += "varying vec2 texcoord;\n";
  vsSource += "void main() {\n";
  vsSource += "  gl_Position = vec4(position, 0.5, 1.0);\n";
  vsSource += "  texcoord = position * vec2(0.5) + vec2(0.5);\n";
  vsSource += "}\n";

  std::string fsSource;
  fsSource += "#extension GL_OES_EGL_image_external : require\n";
  fsSource += "precision highp float;\n";
  if (textureTarget == GL_TEXTURE_EXTERNAL_OES) {
    fsSource += "uniform samplerExternalOES tex;\n";
  } else {
    fsSource += "uniform sampler2D tex;\n";
  }
  fsSource += "varying vec2 texcoord;\n";
  fsSource += "void main() {\n";
  fsSource += "  gl_FragColor = texture2D(tex, texcoord);\n";
  fsSource += "}\n";

  auto prog = imports.glCreateProgram();

  auto vs = imports.glCreateShader(GL_VERTEX_SHADER);
  char* vsSources[] = { const_cast<char*>(vsSource.data()) };
  imports.glShaderSource(vs, 1, vsSources, nullptr);
  imports.glCompileShader(vs);
  imports.glAttachShader(prog, vs);

  auto fs = imports.glCreateShader(GL_FRAGMENT_SHADER);
  char* fsSources[] = { const_cast<char*>(fsSource.data()) };
  imports.glShaderSource(fs, 1, fsSources, nullptr);
  imports.glCompileShader(fs);
  imports.glAttachShader(prog, fs);

  imports.glBindAttribLocation(prog, 0, "position");
  imports.glLinkProgram(prog);

  if ((err = imports.glGetError()) != 0) {
    GAPID_FATAL("MEC: Failed to create program: 0x%X", err);
  }

  GLint linkStatus = 0;
  imports.glGetProgramiv(prog, GL_LINK_STATUS, &linkStatus);
  if (linkStatus == 0) {
      char log[1024];
      imports.glGetProgramInfoLog(prog, sizeof(log), nullptr, log);
    GAPID_FATAL("MEC: Failed to compile program:\n%s", log);
  }

  if ((err = imports.glCheckFramebufferStatus(GL_DRAW_FRAMEBUFFER)) != GL_FRAMEBUFFER_COMPLETE) {
    GAPID_FATAL("MEC: Draw framebuffer incomplete: 0x%X", err);
  }

  imports.glDisable(GL_CULL_FACE);
  imports.glDisable(GL_DEPTH_TEST);
  imports.glViewport(0, 0, w, h);
  imports.glClearColor(0.0, 0.0, 0.0, 0.0);
  imports.glClear(GLbitfield::GL_COLOR_BUFFER_BIT);
  imports.glUseProgram(prog);
  GLfloat vb[] = {
      -1.0f, +1.0f,  // 2--4
      -1.0f, -1.0f,  // |\ |
      +1.0f, +1.0f,  // | \|
      +1.0f, -1.0f,  // 1--3
  };
  imports.glEnableVertexAttribArray(0);
  imports.glVertexAttribPointer(0, 2, GL_FLOAT, 0, 0, vb);
  imports.glDrawArrays(GL_TRIANGLE_STRIP, 0, 4);
  if ((err = imports.glGetError()) != 0) {
    GAPID_FATAL("MEC: Failed to draw quad: 0x%X", err);
  }
}

ImageData ReadPixels(const GlesImports& imports, GLsizei w, GLsizei h) {
  ImageData img;
  uint32_t err;

  if ((err = imports.glCheckFramebufferStatus(GL_FRAMEBUFFER)) != GL_FRAMEBUFFER_COMPLETE) {
    GAPID_FATAL("MEC: ReadPixels: Framebuffer incomplete: 0x%X", err);
  }

  // Ask the driver what is the ideal format/type for reading the pixels.
  imports.glGetIntegerv(GL_IMPLEMENTATION_COLOR_READ_FORMAT, &img.dataFormat);
  imports.glGetIntegerv(GL_IMPLEMENTATION_COLOR_READ_TYPE, &img.dataType);
  if ((err = imports.glGetError()) != 0) {
    GAPID_FATAL("MEC: ReadPixels: Failed to get data format/type: 0x%X", err);
  }

  auto spy = Spy::get();
  auto observer = spy->enter("subUncompressedImageSize", GlesSpy::kApiIndex);
  auto size = spy->subUncompressedImageSize(observer, []{}, w, h, img.dataFormat, img.dataType);
  spy->exit();

  img.width = w;
  img.height = h;
  img.data.reset(new std::vector<uint8_t>(size));
  imports.glReadnPixels(0, 0, w, h, img.dataFormat, img.dataType, img.data->size(), img.data->data());
  if ((err = imports.glGetError()) != 0) {
    GAPID_FATAL("MEC: Failed to read pixels: 0x%X", err);
  }

  return img;
}

ImageData ReadTexture(const GlesImports& imports, uint32_t kind, GLint texID, GLint level, GLint layer,
                      GLsizei w, GLsizei h, uint32_t dataFormat) {
  switch (dataFormat) {
    case GL_STENCIL:
      GAPID_WARNING("MEC: Reading of stencil data is not yet supported"); // TODO
      break;
    case GL_DEPTH_STENCIL:
      GAPID_WARNING("MEC: Reading of stencil data is not yet supported"); // TODO
      // Fall through to depth
    case GL_DEPTH_COMPONENT:
      if (kind != GL_TEXTURE_2D) {
        // TODO: Copy the layer/level to temporary 2D texture and then use the path below.
        GAPID_WARNING("MEC: Reading of depth data for target %i is not yet supported", kind); // TODO
      } else {
        // Convert depth texture to colour texture using shader code
        auto drawFb = CreateAndBindFramebuffer(imports, GL_DRAW_FRAMEBUFFER);
        auto tmpTex = CreateAndBindTexture2D(imports, w, h, GL_R32F);
        imports.glFramebufferTexture(GL_DRAW_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, tmpTex.id(), 0);
        imports.glBindTexture(GL_TEXTURE_2D, texID);
        GLint oldCompMode = 0;
        imports.glGetTexParameteriv(GL_TEXTURE_2D, GL_TEXTURE_COMPARE_MODE, &oldCompMode);
        imports.glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_COMPARE_MODE, GL_NONE);
        DrawTexturedQuad(imports, GL_TEXTURE_2D, w, h);
        imports.glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_COMPARE_MODE, oldCompMode);
        return ReadTexture(imports, GL_TEXTURE_2D, tmpTex.id(), 0, 0, w, h, GL_R);
      }
      break;
    default: {
      auto readFb = CreateAndBindFramebuffer(imports, GL_READ_FRAMEBUFFER);
      if (kind == GL_TEXTURE_CUBE_MAP) {
        uint32_t face = GL_TEXTURE_CUBE_MAP_POSITIVE_X + (layer%6);
        imports.glFramebufferTexture2D(GL_READ_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, face, texID, level);
      } else if (layer == 0) {
        imports.glFramebufferTexture(GL_READ_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, texID, level);
      } else {
        imports.glFramebufferTextureLayer(GL_READ_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, texID, level, layer);
      }
      return ReadPixels(imports, w, h);
    }
  }
  return ImageData{};
}

ImageData ReadRenderbuffer(const GlesImports& imports, Renderbuffer* rb) {
  auto img = rb->mImage;
  auto w = img->mWidth;
  auto h = img->mHeight;
  auto dataFmt = rb->mImage->mDataFormat;
  uint32_t attachment = GL_COLOR_ATTACHMENT0;
  switch (dataFmt) {
    case GL_DEPTH_COMPONENT: attachment = GL_DEPTH_ATTACHMENT; break;
    case GL_DEPTH_STENCIL:   attachment = GL_DEPTH_STENCIL_ATTACHMENT; break;
    case GL_STENCIL:         attachment = GL_STENCIL_ATTACHMENT; break;
  }
  if (attachment == GL_COLOR_ATTACHMENT0) {
    auto readFb = CreateAndBindFramebuffer(imports, GL_READ_FRAMEBUFFER);
    imports.glFramebufferRenderbuffer(GL_READ_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_RENDERBUFFER, rb->mID);
    return ReadPixels(imports, w, h);
  } else {
    // Copy the renderbuffer data to temporary texture and then use the texture reading path.
    auto readFb = CreateAndBindFramebuffer(imports, GL_READ_FRAMEBUFFER);
    auto drawFb = CreateAndBindFramebuffer(imports, GL_DRAW_FRAMEBUFFER);
    auto tmpTex = CreateAndBindTexture2D(imports, w, h, img->mSizedFormat);
    imports.glFramebufferRenderbuffer(GL_READ_FRAMEBUFFER, attachment, GL_RENDERBUFFER, rb->mID);
    imports.glFramebufferTexture(GL_DRAW_FRAMEBUFFER, attachment, tmpTex.id(), 0);
    uint32_t mask = GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT | GL_STENCIL_BUFFER_BIT;
    imports.glBlitFramebuffer(0, 0, w, h, 0, 0, w, h, mask, GL_NEAREST);
    return ReadTexture(imports, GL_TEXTURE_2D, tmpTex.id(), 0, 0, w, h, dataFmt);
  }
}

ImageData ReadExternal(const GlesImports& imports, EGLImageKHR handle, GLsizei w, GLsizei h) {
  auto extTex = CreateAndBindTextureExternal(imports, handle);
  auto tmpTex = CreateAndBindTexture2D(imports, w, h, GL_RGBA8);
  auto fb = CreateAndBindFramebuffer(imports, GL_FRAMEBUFFER);
  imports.glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, tmpTex.id(), 0);
  DrawTexturedQuad(imports, GL_TEXTURE_EXTERNAL_OES, w, h);
  return ReadPixels(imports, w, h);
}

void SerializeAndUpdate(StateSerializer* serializer, gapil::Ref<gapii::Image> current, const ImageData& read) {
  current->mData = serializer->encodeBuffer<uint8_t>(read.data->size(),
    [serializer,&read](memory::Observation* obs) {
      serializer->sendData(obs, false, read.data->data(), read.data->size());
    });
  current->mDataFormat = read.dataFormat;
  current->mDataType = read.dataType;
}

}  // anonymous namespace

namespace gapii {
using namespace GLenum;
using namespace GLbitfield;
using namespace EGLenum;

void GlesSpy::GetEGLImageData(CallObserver* observer, EGLImageKHR handle, GLsizei width, GLsizei height) {
  if (!should_trace(kApiIndex)) {
    return;
  }

  GAPID_DEBUG("Get EGLImage data: 0x%p x%xx%x", handle, width, height);
  auto tmpCtx = CreateAndBindContext(mImports, nullptr);

  auto img = ReadExternal(mImports, handle, width, height);

  if (!img.data->empty()) {
    auto resIndex = sendResource(kApiIndex, img.data->data(), img.data->size());
    auto extra = new gles_pb::EGLImageData();
    extra->set_resindex(resIndex);
    extra->set_size(img.data->size());
    extra->set_width(width);
    extra->set_height(height);
    extra->set_format(img.dataFormat);
    extra->set_type(img.dataType);
    observer->encodeAndDelete(extra);
  }
}

void GlesSpy::serializeGPUBuffers(StateSerializer* serializer) {
  // Ensure we process shared objects only once.
  std::unordered_set<void*> seen;
  auto once = [&](void* ptr) { return seen.emplace(ptr).second; };

  for (auto& it : mState.EGLContexts) {
    auto handle = it.first;
    auto ctx = it.second;
    auto tmpCtx = CreateAndBindContext(mImports, handle);
    if (once(&ctx->mObjects.mRenderbuffers)) {
      for (auto& it : ctx->mObjects.mRenderbuffers) {
        auto rb = it.second;
        auto img = rb->mImage;
        if (img != nullptr) {
          auto newImg = ReadRenderbuffer(mImports, rb.get());
          SerializeAndUpdate(serializer, img, newImg);
        }
      }
    }
    if (once(&ctx->mObjects.mTextures)) {
      for (auto& it : ctx->mObjects.mTextures) {
        auto tex = it.second;
        auto eglImage = tex->mEGLImage.get();
        if (eglImage != nullptr) {
          if (once(eglImage)) {
            for (auto& it : eglImage->mImages) {
              auto img = it.second;
              auto newImg = ReadExternal(mImports, eglImage->mID, img->mWidth, img->mHeight);
              SerializeAndUpdate(serializer, img, newImg);
            }
          }
        } else {
          for (auto it : tex->mLevels) {
            auto level = it.first;
            for (auto it2 : it.second.mLayers) {
              auto layer = it2.first;
              auto img = it2.second;
              if (img->mSamples != 0) {
                GAPID_WARNING("MEC: Reading of multisample textures is not yet supported"); // TODO
                continue;
              }
              auto newImg = ReadTexture(mImports, tex->mKind, tex->mID, level, layer,
                                        img->mWidth, img->mHeight, img->mDataFormat);
              SerializeAndUpdate(serializer, img, newImg);
            }
          }
        }
      }
    }
    if (once(&ctx->mObjects.mBuffers)) {
      for (auto& it : ctx->mObjects.mBuffers) {
        auto buffer = it.second;
        if (buffer->mMapped) {
          // TODO: Implement - it is fairly difficult to access already mapped buffer.
          // We can get the mapped pointer but it might not be mapping the whole buffer.
          // We can not unmap and remap since the application has the existing pointer,
          // and we can copy the buffer since copy is not allowed for mapped buffers.
          // Proposed solution: change glMapBuffer* to always map the whole buffer,
          // and return pointer inside that buffer to the user.
          GAPID_WARNING("MEC: Can not read mapped buffer")
          continue;
        }
        size_t size = buffer->mSize;
        if (size == 0) {
          continue;
        }
        GAPID_ASSERT(buffer->mData.pool() != nullptr);
        GAPID_ASSERT(buffer->mData.size() == size);
        const uint32_t target = GL_ARRAY_BUFFER;
        mImports.glBindBuffer(target, buffer->mID);
        void* data = mImports.glMapBufferRange(target, 0, size, GL_MAP_READ_BIT);
        buffer->mData = serializer->encodeBuffer<uint8_t>(size,
          [=](memory::Observation* obs) {
            serializer->sendData(obs, false, data, size);
        });
        mImports.glUnmapBuffer(target);
      }
    }
  }
}

}  // namespace gapii
