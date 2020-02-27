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

#include <sstream>

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
  GLint sizedFormat;
  GLint dataFormat;
  GLint dataType;
};

class TempObject {
 public:
  TempObject(uint64_t id, const std::function<void()>& deleteId)
      : mId(id), mDeleteId(deleteId) {}

  TempObject(TempObject&& other) {
    mId = other.mId;
    mDeleteId = other.mDeleteId;
    other.mDeleteId = nullptr;
  }

  ~TempObject() {
    if (mDeleteId) {
      mDeleteId();
    }
  }

  uint64_t id() { return mId; }

 private:
  TempObject(const TempObject&) = delete;
  TempObject& operator=(const TempObject&) = delete;

  uint64_t mId;
  std::function<void()> mDeleteId;
};

struct swizzle {
  GLint r, g, b, a;
};

struct index {
  GLint level, layer;
};

class Sampler {
 public:
  enum ESVersion { ES20, ES30, ES31 };
  Sampler() {}

  virtual ESVersion getESVersion() const { return ES20; }

  virtual std::string getExtensions() const = 0;
  virtual std::string getUniform() const = 0;
  virtual std::string getFragmentPreamble() const { return ""; }
  virtual std::string getSamplingExpression() const = 0;
};

class Sampler2D : public Sampler {
 public:
  Sampler2D() {}

  virtual std::string getExtensions() const { return ""; }

  virtual std::string getUniform() const { return "uniform sampler2D tex;"; }

  virtual std::string getSamplingExpression() const {
    return "texture2D(tex, texcoord)";
  }

  static const Sampler& get() {
    static const Sampler2D instance;
    return instance;
  }
};

class Sampler2DArray : public Sampler {
 public:
  Sampler2DArray(int layer) : mLayer(layer) {}

  virtual ESVersion getESVersion() const { return ES30; }

  virtual std::string getExtensions() const { return ""; }

  virtual std::string getUniform() const {
    return "uniform sampler2DArray tex;";
  }

  virtual std::string getSamplingExpression() const {
    std::ostringstream r;
    r << "texture(tex, vec3(texcoord, " << mLayer << ".0))";
    return r.str();
  }

 private:
  GLint mLayer;
};

class SamplerExternal : public Sampler {
 public:
  SamplerExternal() {}

  virtual std::string getExtensions() const {
    return "#extension GL_OES_EGL_image_external : require\n";
  }

  virtual std::string getUniform() const {
    return "uniform samplerExternalOES tex;";
  }

  virtual std::string getSamplingExpression() const {
    return "texture2D(tex, texcoord)";
  }

  static const Sampler& get() {
    static const SamplerExternal instance;
    return instance;
  }
};

class Sampler3D : public Sampler {
 public:
  Sampler3D(float z) : mZ(z) {}

  virtual std::string getExtensions() const {
    return "#extension GL_OES_texture_3D : require\n";
  }

  virtual std::string getUniform() const { return "uniform sampler3D tex;"; }

  virtual std::string getSamplingExpression() const {
    std::ostringstream r;
    r << "texture3D(tex, vec3(texcoord, " << mZ << "))";
    return r.str();
  }

 private:
  float mZ;
};

class SamplerCube : public Sampler {
 public:
  SamplerCube(GLint face) : mFace(face) {}

  virtual std::string getExtensions() const { return ""; }

  virtual std::string getUniform() const { return "uniform samplerCube tex;"; }

  virtual std::string getFragmentPreamble() const {
    // uv.xy is texcoord expanded to [-1, 1].
    // uv.zw is -uv.xy.
    return "vec4 uv = vec4(-1.0 + 2.0 * texcoord, 1.0 - 2.0 * texcoord);";
  }

  virtual std::string getSamplingExpression() const {
    switch (mFace) {
      case GL_TEXTURE_CUBE_MAP_POSITIVE_X:
        return "textureCube(tex, vec3(1.0, uv.wz))";
      case GL_TEXTURE_CUBE_MAP_NEGATIVE_X:
        return "textureCube(tex, vec3(-1.0, uv.wx))";
      case GL_TEXTURE_CUBE_MAP_POSITIVE_Y:
        return "textureCube(tex, vec3(uv.x, 1.0, uv.y))";
      case GL_TEXTURE_CUBE_MAP_NEGATIVE_Y:
        return "textureCube(tex, vec3(uv.x, -1.0, uv.w))";
      case GL_TEXTURE_CUBE_MAP_POSITIVE_Z:
        return "textureCube(tex, vec3(uv.xw, 1.0))";
      case GL_TEXTURE_CUBE_MAP_NEGATIVE_Z:
        return "textureCube(tex, vec3(uv.zw, -1.0))";
      default:
        GAPID_FATAL("MEC: unexpected cube face: 0x%x", mFace);
        return "";
    }
  }

 private:
  GLint mFace;
};

class SamplerCubeArray : public Sampler {
 public:
  SamplerCubeArray(int layer, GLint face) : mLayer(layer), mFace(face) {}

  virtual ESVersion getESVersion() const { return ES31; }

  virtual std::string getExtensions() const {
    return "#extension GL_EXT_texture_cube_map_array : require\n";
  }

  virtual std::string getUniform() const {
    return "uniform samplerCubeArray tex;";
  }

  virtual std::string getFragmentPreamble() const {
    // uv.xy is texcoord expanded to [-1, 1].
    // uv.zw is -uv.xy.
    return "vec4 uv = vec4(-1.0 + 2.0 * texcoord, 1.0 - 2.0 * texcoord);";
  }

  virtual std::string getSamplingExpression() const {
    std::ostringstream r;
    r << "texture(tex, vec4(";

    switch (mFace) {
      case GL_TEXTURE_CUBE_MAP_POSITIVE_X:
        r << "1.0, uv.wz";
        break;
      case GL_TEXTURE_CUBE_MAP_NEGATIVE_X:
        r << "-1.0, uv.wx";
        break;
      case GL_TEXTURE_CUBE_MAP_POSITIVE_Y:
        r << "uv.x, 1.0, uv.y";
        break;
      case GL_TEXTURE_CUBE_MAP_NEGATIVE_Y:
        r << "uv.x, -1.0, uv.w";
        break;
      case GL_TEXTURE_CUBE_MAP_POSITIVE_Z:
        r << "uv.xw, 1.0";
        break;
      case GL_TEXTURE_CUBE_MAP_NEGATIVE_Z:
        r << "uv.xw, 1.0";
        break;
      default:
        GAPID_FATAL("MEC: unexpected cube face: 0x%x", mFace);
        return "";
    }

    r << ", " << mLayer << ".0))";
    return r.str();
  }

 private:
  int mLayer;
  GLint mFace;
};

struct texture_state {
  swizzle swiz;
  GLint compMode;
  GLint minFilter, magFilter;
  GLint baseLevel, maxLevel;
  GLfloat minLod, maxLod;
};

struct texture {
  uint64_t id;
  uint32_t kind;
  GLsizei w, h, d;

  void bind(const GlesImports& i) const { i.glBindTexture(kind, id); }

  // Gets the current state of the texture.
  texture_state getState(const GlesImports& i) const {
    texture_state r;
    i.glGetTexParameteriv(kind, GL_TEXTURE_SWIZZLE_R, &r.swiz.r);
    i.glGetTexParameteriv(kind, GL_TEXTURE_SWIZZLE_G, &r.swiz.g);
    i.glGetTexParameteriv(kind, GL_TEXTURE_SWIZZLE_B, &r.swiz.b);
    i.glGetTexParameteriv(kind, GL_TEXTURE_SWIZZLE_A, &r.swiz.a);
    i.glGetTexParameteriv(kind, GL_TEXTURE_COMPARE_MODE, &r.compMode);
    i.glGetTexParameteriv(kind, GL_TEXTURE_MIN_FILTER, &r.minFilter);
    i.glGetTexParameteriv(kind, GL_TEXTURE_MAG_FILTER, &r.magFilter);
    i.glGetTexParameteriv(kind, GL_TEXTURE_BASE_LEVEL, &r.baseLevel);
    i.glGetTexParameteriv(kind, GL_TEXTURE_MAX_LEVEL, &r.maxLevel);
    i.glGetTexParameterfv(kind, GL_TEXTURE_MIN_LOD, &r.minLod);
    i.glGetTexParameterfv(kind, GL_TEXTURE_MAX_LOD, &r.maxLod);
    return r;
  }

  // Modifes the texture's state so it can be savely used to read from. Make
  // sure to call setState() after, with a previously saved state.
  void prepareToRead(const GlesImports& i, GLint level, swizzle swizzle) const {
    i.glTexParameteri(kind, GL_TEXTURE_SWIZZLE_R, swizzle.r);
    i.glTexParameteri(kind, GL_TEXTURE_SWIZZLE_G, swizzle.g);
    i.glTexParameteri(kind, GL_TEXTURE_SWIZZLE_B, swizzle.b);
    i.glTexParameteri(kind, GL_TEXTURE_SWIZZLE_A, swizzle.a);
    i.glTexParameteri(kind, GL_TEXTURE_COMPARE_MODE, GL_NONE);
    i.glTexParameteri(kind, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
    i.glTexParameteri(kind, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
    i.glTexParameteri(kind, GL_TEXTURE_BASE_LEVEL, level);
    i.glTexParameteri(kind, GL_TEXTURE_MAX_LEVEL, level);
    i.glTexParameterf(kind, GL_TEXTURE_MIN_LOD, -1000.0f);
    i.glTexParameterf(kind, GL_TEXTURE_MAX_LOD, 1000.0f);
  }

  // Modifies the texture's state to match the given state.
  void setState(const GlesImports& i, const texture_state& state) const {
    i.glTexParameteri(kind, GL_TEXTURE_SWIZZLE_R, state.swiz.r);
    i.glTexParameteri(kind, GL_TEXTURE_SWIZZLE_G, state.swiz.g);
    i.glTexParameteri(kind, GL_TEXTURE_SWIZZLE_B, state.swiz.b);
    i.glTexParameteri(kind, GL_TEXTURE_SWIZZLE_A, state.swiz.a);
    i.glTexParameteri(kind, GL_TEXTURE_COMPARE_MODE, state.compMode);
    i.glTexParameteri(kind, GL_TEXTURE_MIN_FILTER, state.minFilter);
    i.glTexParameteri(kind, GL_TEXTURE_MAG_FILTER, state.magFilter);
    i.glTexParameteri(kind, GL_TEXTURE_BASE_LEVEL, state.baseLevel);
    i.glTexParameteri(kind, GL_TEXTURE_MAX_LEVEL, state.maxLevel);
    i.glTexParameterf(kind, GL_TEXTURE_MIN_LOD, state.minLod);
    i.glTexParameterf(kind, GL_TEXTURE_MAX_LOD, state.maxLod);
  }
};

class Reader {
 public:
  Reader(const GlesImports& imports) : imports(imports) {}

  TempObject CreateAndBindFramebuffer(uint32_t target);
  TempObject CreateAndBindTexture2D(GLint w, GLint h, uint32_t sizedFormat);
  TempObject CreateAndBindTextureExternal(EGLImageKHR handle);
  TempObject CreateAndBindContext(EGLContext sharedContext, EGLint version);

  ImageData ReadTexture(const texture& tex, index index, uint32_t format);
  ImageData ReadRenderbuffer(Renderbuffer* rb);
  ImageData ReadExternal(EGLImageKHR handle, GLsizei w, GLsizei h);

 private:
  ImageData ReadTextureViaDrawQuad(const texture& tex, index index,
                                   uint32_t format, const char* name,
                                   swizzle swizzle);
  ImageData ReadTextureViaDrawQuad(const Sampler& sampler, const texture& tex,
                                   index index, uint32_t format,
                                   swizzle swizzle);

  inline ImageData ReadTextureViaDrawQuad(const texture& tex, index index,
                                          uint32_t format, const char* name,
                                          uint32_t originalFormat,
                                          GLint rSwizzle) {
    ImageData result = ReadTextureViaDrawQuad(
        tex, index, format, name, {rSwizzle, GL_ZERO, GL_ZERO, GL_ONE});
    // Restore original format, so it doesn't show up as GL_RED in the UI.
    result.dataFormat = originalFormat;
    return result;
  }

  inline ImageData ReadTextureViaDrawQuad(const texture& tex, index index,
                                          uint32_t format, const char* name,
                                          uint32_t originalFormat,
                                          GLint rSwizzle, GLint gSwizzle) {
    ImageData result = ReadTextureViaDrawQuad(
        tex, index, format, name, {rSwizzle, gSwizzle, GL_ZERO, GL_ONE});
    // Restore original format, so it doesn't show up as GL_RG in the UI.
    result.dataFormat = originalFormat;
    return result;
  }

  inline ImageData ReadCompressedTexture(const texture& tex, index index,
                                         uint32_t format, swizzle swizzle) {
    ImageData result =
        ReadTextureViaDrawQuad(tex, index, format, "compressed", swizzle);
    // Override the internal format to the uncompressed format of the data.
    result.sizedFormat = format;
    return result;
  }

  void DrawTexturedQuad(const Sampler& sampler, GLsizei w, GLsizei h);
  ImageData ReadPixels(GLsizei w, GLsizei h);

  const GlesImports& imports;
};

TempObject Reader::CreateAndBindFramebuffer(uint32_t target) {
  GLuint fb = 0;
  imports.glGenFramebuffers(1, &fb);
  imports.glBindFramebuffer(target, fb);
  return TempObject(fb, [=] {
    GLuint id = fb;
    imports.glDeleteFramebuffers(1, &id);
  });
}

TempObject Reader::CreateAndBindTexture2D(GLint w, GLint h, uint32_t format) {
  GLuint tex = 0;
  imports.glGenTextures(1, &tex);
  imports.glBindTexture(GL_TEXTURE_2D, tex);
  imports.glTexStorage2D(GL_TEXTURE_2D, 1, format, w, h);
  return TempObject(tex, [=] {
    GLuint id = tex;
    imports.glDeleteTextures(1, &id);
  });
}

TempObject Reader::CreateAndBindTextureExternal(EGLImageKHR handle) {
  GLuint tex = 0;
  imports.glGenTextures(1, &tex);
  imports.glBindTexture(GL_TEXTURE_EXTERNAL_OES, tex);
  imports.glEGLImageTargetTexture2DOES(GL_TEXTURE_EXTERNAL_OES, handle);
  imports.glTexParameteri(GL_TEXTURE_EXTERNAL_OES, GL_TEXTURE_MIN_FILTER,
                          GL_NEAREST);
  imports.glTexParameteri(GL_TEXTURE_EXTERNAL_OES, GL_TEXTURE_MAG_FILTER,
                          GL_NEAREST);
  return TempObject(tex, [=] {
    GLuint id = tex;
    imports.glDeleteTextures(1, &id);
  });
}

#define EGL_WARNING(FORMAT, ...) \
  GAPID_WARNING(FORMAT ": 0x%x", ##__VA_ARGS__, imports.eglGetError())
#define EGL_FATAL(FORMAT, ...) \
  GAPID_FATAL(FORMAT ": 0x%x", ##__VA_ARGS__, imports.eglGetError())

#define CHECK_GL_ERROR(FORMAT, ...)      \
  do {                                   \
    GLint err = imports.glGetError();    \
    if (err) {                           \
      GAPID_FATAL(FORMAT ": 0x%x", err); \
    }                                    \
  } while (false)

#define CHECK_FB_COMPLETE(FB, FORMAT, ...)               \
  do {                                                   \
    GLint status = imports.glCheckFramebufferStatus(FB); \
    if (status != GL_FRAMEBUFFER_COMPLETE) {             \
      GAPID_FATAL(FORMAT ": 0x%x", status);              \
    }                                                    \
  } while (false)

// Attempts to create a context sharing the given context, by querying the
// share context for its EGLConfig ID.
EGLContext CreateEglContextForSharedContext(const GlesImports& imports,
                                            EGLDisplay dsp, EGLContext share,
                                            EGLint* ctxAtr) {
  EGLint id = 42;
  if (!imports.eglQueryContext(dsp, share, EGL_CONFIG_ID, &id)) {
    EGL_WARNING("MEC: Failed to query the config ID of the context");
    return nullptr;
  }

  EGLint cfgAtr[] = {EGL_CONFIG_ID, id, EGL_NONE};
  EGLConfig cfg;
  int one = 1;
  if (!imports.eglChooseConfig(dsp, cfgAtr, &cfg, 1, &one) || one != 1) {
    EGL_WARNING("MEC: Failed to choose EGLConfig by id %d", id);
    return nullptr;
  }

  EGLContext ctx;
  if ((ctx = imports.eglCreateContext(dsp, cfg, share, ctxAtr)) == nullptr) {
    EGL_WARNING("MEC: Failed to create EGL context");
  }
  return ctx;
}

// Creates temporary GL context which shares objects with the given application
// context. This makes it easier to do a lot of work without worrying about
// corrupting the state. For example, calling glGetError would be otherwise
// technically invalid without hacks.
TempObject Reader::CreateAndBindContext(EGLContext share, EGLint version) {
  // Save old state.
  auto dsp = imports.eglGetCurrentDisplay();
  auto draw = imports.eglGetCurrentSurface(EGL_DRAW);
  auto read = imports.eglGetCurrentSurface(EGL_READ);
  auto oldCtx = imports.eglGetCurrentContext();

  // Find an EGL config.
  EGLConfig cfg;
  EGLint cfgAtr[] = {EGL_RENDERABLE_TYPE, EGL_OPENGL_ES2_BIT, EGL_NONE};
  int one = 1;
  if (!imports.eglChooseConfig(dsp, cfgAtr, &cfg, 1, &one) || one != 1) {
    EGL_FATAL("MEC: Failed to choose EGLConfig");
  }

  // Create an EGL context.
  EGLContext ctx;
  EGLint ctxAtr[] = {EGL_CONTEXT_CLIENT_VERSION, version, EGL_NONE};
  if ((ctx = imports.eglCreateContext(dsp, cfg, share, ctxAtr)) == nullptr) {
    EGLint error = imports.eglGetError();
    if (share == nullptr || error != EGL_BAD_MATCH) {
      GAPID_WARNING("MEC: Failed to create EGL context: 0x%x", error);
    } else {
      GAPID_WARNING("MEC: BAD_MATCH creating shared context. Querying config.");
      ctx = CreateEglContextForSharedContext(imports, dsp, share, ctxAtr);
    }
  }

  if (ctx == nullptr) {
    return TempObject(reinterpret_cast<uint64_t>(ctx), [=] {
      if (!imports.eglMakeCurrent(dsp, draw, read, oldCtx)) {
        EGL_FATAL("MEC: Failed to restore old EGL context");
      }
    });
  }

  // Create an EGL surface.
  EGLSurface surf;
  EGLint surfAtr[] = {EGL_WIDTH, 16, EGL_HEIGHT, 16, EGL_NONE};
  if ((surf = imports.eglCreatePbufferSurface(dsp, cfg, surfAtr)) == nullptr) {
    EGL_FATAL("MEC: Failed to create EGL surface");
  }

  // Bind the EGL context.
  if (!imports.eglMakeCurrent(dsp, surf, surf, ctx)) {
    EGL_FATAL("MEC: Failed to bind new EGL context");
  }

  // Setup desirable default state for reading data.
  imports.glPixelStorei(GL_PACK_ALIGNMENT, 1);
  imports.glPixelStorei(GL_PACK_ROW_LENGTH, 0);
  imports.glPixelStorei(GL_PACK_SKIP_PIXELS, 0);
  imports.glPixelStorei(GL_PACK_SKIP_ROWS, 0);

  return TempObject(reinterpret_cast<uint64_t>(ctx), [=] {
    if (!imports.eglMakeCurrent(dsp, draw, read, oldCtx)) {
      EGL_FATAL("MEC: Failed to restore old EGL context");
    }
    imports.eglDestroySurface(dsp, surf);
    imports.eglDestroyContext(dsp, ctx);
  });
}

#define GLSL_VERSION(version) \
  (version == Sampler::ES30   \
       ? "#version 300 es\n"  \
       : version == Sampler::ES31 ? "#version 310 es\n" : "")

void Reader::DrawTexturedQuad(const Sampler& sampler, GLsizei w, GLsizei h) {
  CHECK_GL_ERROR("MEC: Entered DrawTexturedQuad in error state");
  CHECK_FB_COMPLETE(GL_DRAW_FRAMEBUFFER, "MEC: Draw framebuffer incomplete");

  Sampler::ESVersion version = sampler.getESVersion();
  std::string vsSource;
  if (version != Sampler::ES20) {
    vsSource += GLSL_VERSION(version);
    vsSource += "#define attribute in\n";
    vsSource += "#define varying out\n";
  }
  vsSource += "precision highp float;\n";
  vsSource += "attribute vec2 position;\n";
  vsSource += "varying vec2 texcoord;\n";
  vsSource += "void main() {\n";
  vsSource += "  gl_Position = vec4(position, 0.5, 1.0);\n";
  vsSource += "  texcoord = position * vec2(0.5) + vec2(0.5);\n";
  vsSource += "}\n";

  std::string fsSource;
  fsSource += GLSL_VERSION(version);
  fsSource += sampler.getExtensions();
  fsSource += "precision highp float;\n";
  if (version != Sampler::ES20) {
    fsSource += "#define varying in\n";
    fsSource += "out vec4 fragColor;\n";
  } else {
    fsSource += "#define fragColor gl_FragColor\n";
  }
  fsSource += sampler.getUniform();
  fsSource += "varying vec2 texcoord;\n";
  fsSource += "void main() {\n";
  fsSource += sampler.getFragmentPreamble();
  fsSource += "  fragColor = " + sampler.getSamplingExpression() + ";\n";
  fsSource += "}\n";

  auto prog = imports.glCreateProgram();

  auto vs = imports.glCreateShader(GL_VERTEX_SHADER);
  char* vsSources[] = {const_cast<char*>(vsSource.data())};
  imports.glShaderSource(vs, 1, vsSources, nullptr);
  imports.glCompileShader(vs);
  imports.glAttachShader(prog, vs);

  auto fs = imports.glCreateShader(GL_FRAGMENT_SHADER);
  char* fsSources[] = {const_cast<char*>(fsSource.data())};
  imports.glShaderSource(fs, 1, fsSources, nullptr);
  imports.glCompileShader(fs);
  imports.glAttachShader(prog, fs);

  imports.glBindAttribLocation(prog, 0, "position");
  imports.glLinkProgram(prog);
  CHECK_GL_ERROR("MEC: Failed to create program");

  GLint linkStatus = 0;
  imports.glGetProgramiv(prog, GL_LINK_STATUS, &linkStatus);
  if (linkStatus == 0) {
    char log[1024];
    imports.glGetProgramInfoLog(prog, sizeof(log), nullptr, log);
    GAPID_ERROR("Vertex Shader:\n%s\n\nFragment Shader:\n%s", vsSource.c_str(),
                fsSource.c_str());
    GAPID_FATAL("MEC: Failed to compile program:\n%s", log);
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
  CHECK_GL_ERROR("MEC: Failed to draw quad");

  imports.glDeleteShader(vs);
  imports.glDeleteShader(fs);
  imports.glDeleteProgram(prog);
}

ImageData Reader::ReadPixels(GLsizei w, GLsizei h) {
  CHECK_FB_COMPLETE(GL_READ_FRAMEBUFFER, "ReadPixels: Framebuffer incomplete");

  ImageData img;

  // Ask the driver what is the ideal format/type for reading the pixels.
  imports.glGetIntegerv(GL_IMPLEMENTATION_COLOR_READ_FORMAT, &img.dataFormat);
  imports.glGetIntegerv(GL_IMPLEMENTATION_COLOR_READ_TYPE, &img.dataType);
  CHECK_GL_ERROR("ReadPixels: Failed to get data format/type");
  GAPID_DEBUG("ReadPixels: fmt 0x%x type 0x%x", img.dataFormat, img.dataType);

  auto spy = Spy::get();
  auto observer = spy->enter("subUncompressedImageSize", GlesSpy::kApiIndex);
  auto size = spy->subUncompressedImageSize(observer, [] {}, w, h,
                                            img.dataFormat, img.dataType);
  spy->exit();

  img.sizedFormat = GL_NONE;
  img.width = w;
  img.height = h;
  img.data.reset(new std::vector<uint8_t>(size));
  GAPID_DEBUG("ReadPixels: Reading %dx%d at %d bytes", w, h, size);
  if (imports.glReadnPixels) {
    imports.glReadnPixels(0, 0, w, h, img.dataFormat, img.dataType,
                          img.data->size(), img.data->data());
  } else {
    imports.glReadPixels(0, 0, w, h, img.dataFormat, img.dataType,
                         img.data->data());
  }
  CHECK_GL_ERROR("ReadPixels: Failed to read pixels");
  return img;
}

#define CUBE_FACE(layer) (GL_TEXTURE_CUBE_MAP_POSITIVE_X + (layer))

ImageData Reader::ReadTextureViaDrawQuad(const texture& tex, index idx,
                                         uint32_t format, const char* name,
                                         swizzle swizzle) {
  switch (tex.kind) {
    case GL_TEXTURE_2D:
      return ReadTextureViaDrawQuad(Sampler2D::get(), tex, idx, format,
                                    swizzle);
    case GL_TEXTURE_2D_ARRAY: {
      Sampler2DArray sampler(idx.layer);
      return ReadTextureViaDrawQuad(sampler, tex, idx, format, swizzle);
    }
    case GL_TEXTURE_3D: {
      Sampler3D sampler(1.0f / (2.0f * tex.d) + (float)idx.layer / tex.d);
      return ReadTextureViaDrawQuad(sampler, tex, idx, format, swizzle);
    }
    case GL_TEXTURE_CUBE_MAP: {
      SamplerCube sampler(CUBE_FACE(idx.layer));
      return ReadTextureViaDrawQuad(sampler, tex, idx, format, swizzle);
    }
    case GL_TEXTURE_CUBE_MAP_ARRAY: {
      SamplerCubeArray sampler(idx.layer / 6, CUBE_FACE(idx.layer % 6));
      return ReadTextureViaDrawQuad(sampler, tex, idx, format, swizzle);
    }
    default:
      // TODO: Copy the layer/level to temporary 2D texture.
      GAPID_WARNING("MEC: Reading of %s data, target 0x%x is not yet supported",
                    name, tex.kind);
      return ImageData{};
  }
}

ImageData Reader::ReadTexture(const texture& tex, index idx, uint32_t format) {
  GAPID_DEBUG("MEC: Reading texture %" PRIu64 " kind 0x%x %dx%d format 0x%x",
              tex.id, tex.kind, tex.w, tex.h, format);
  switch (format) {
    /* depth and stencil */
    case GL_STENCIL_INDEX8:
      GAPID_WARNING("MEC: Reading of stencil data is not yet supported");
      break;
    case GL_DEPTH24_STENCIL8:
    case GL_DEPTH32F_STENCIL8:
      GAPID_WARNING("MEC: Reading of depth stencil data is not yet supported");
      break;
    case GL_DEPTH_COMPONENT16:
    case GL_DEPTH_COMPONENT24:
    case GL_DEPTH_COMPONENT32:
    case GL_DEPTH_COMPONENT32F: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_R32F, "depth",
                                           GL_DEPTH_COMPONENT, GL_RED);
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_DEPTH_COMPONENT32F;
      return r;
    }
    /* alpha and luminance */
    case GL_ALPHA8_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_R8, "alpha", GL_ALPHA,
                                    GL_ALPHA);
    case GL_ALPHA16F_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_R16F_EXT, "alpha", GL_ALPHA,
                                    GL_ALPHA);
    case GL_ALPHA32F_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_R32F, "alpha", GL_ALPHA,
                                    GL_ALPHA);
    case GL_LUMINANCE8_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_R8, "luminance", GL_LUMINANCE,
                                    GL_RED);
    case GL_LUMINANCE16F_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_R16F_EXT, "luminance",
                                    GL_LUMINANCE, GL_RED);
    case GL_LUMINANCE32F_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_R32F, "luminance",
                                    GL_LUMINANCE, GL_RED);
    case GL_LUMINANCE8_ALPHA8_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_RG8, "luminance alpha",
                                    GL_LUMINANCE_ALPHA, GL_RED, GL_ALPHA);
    case GL_LUMINANCE_ALPHA16F_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_RG16F_EXT, "luminance alpha",
                                    GL_LUMINANCE_ALPHA, GL_RED, GL_ALPHA);
    case GL_LUMINANCE_ALPHA32F_EXT:
      return ReadTextureViaDrawQuad(tex, idx, GL_RG32F, "luminance alpha",
                                    GL_LUMINANCE_ALPHA, GL_RED, GL_ALPHA);
    /* RGB */
    case GL_RGB32F: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA32F, "RGB32F",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA32F;
      return r;
    }
    case GL_RGB16F: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA16F, "RGB16F",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA16F;
      return r;
    }
    case GL_RGB9_E5: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA16F, "RGB9_E5",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA16F;
      return r;
    }
    case GL_RGB8I: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA8I, "RGB8I",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA8I;
      return r;
    }
    case GL_RGB8UI: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA8UI, "RGB8UI",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA8UI;
      return r;
    }
    case GL_RGB16I: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA16I, "RGB16I",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA16I;
      return r;
    }
    case GL_RGB16UI: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA16UI, "RGB16UI",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA16UI;
      return r;
    }
    case GL_RGB32I: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA16I, "RGB32I",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA32I;
      return r;
    }
    case GL_RGB32UI: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA16UI, "RGB32UI",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA32UI;
      return r;
    }
    /* SRGB */
    case GL_SRGB8: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_SRGB8_ALPHA8, "srgb",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_SRGB8_ALPHA8;
      return r;
    }
    /* signed normalized */
    // TODO: Draw to a regular R/RG/RGB/RGBA 8bit texture with a shader that
    // will map [-1, 1] to the correct values.
    case GL_R8_SNORM: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_R16F, "R8_SNORM",
                                           {GL_RED, GL_ZERO, GL_ZERO, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_R16F;
      return r;
    }
    case GL_RG8_SNORM: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RG16F, "RG8_SNORM",
                                           {GL_RED, GL_GREEN, GL_ZERO, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RG16F;
      return r;
    }
    case GL_RGB8_SNORM: {
      ImageData r = ReadTextureViaDrawQuad(tex, idx, GL_RGBA16F, "RGB8_SNORM",
                                           {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA16F;
      return r;
    }
    case GL_RGBA8_SNORM: {
      ImageData r =
          ReadTextureViaDrawQuad(tex, idx, GL_RGBA16F, "RGBGBA8_SNORM",
                                 {GL_RED, GL_GREEN, GL_BLUE, GL_ALPHA});
      // Override the internal format to the format of the data.
      r.sizedFormat = GL_RGBA16F;
      return r;
    }
    /* compressed 8bit RGB */
    case GL_COMPRESSED_RGB8_ETC2:
    case GL_COMPRESSED_RGB_S3TC_DXT1_EXT:
    case GL_ATC_RGB_AMD:
    case GL_ETC1_RGB8_OES:
      return ReadCompressedTexture(tex, idx, GL_RGBA8,
                                   {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
    /* compressed 8bit RGBA */
    case GL_COMPRESSED_RGBA_ASTC_4x4:
    case GL_COMPRESSED_RGBA_ASTC_5x4:
    case GL_COMPRESSED_RGBA_ASTC_5x5:
    case GL_COMPRESSED_RGBA_ASTC_6x5:
    case GL_COMPRESSED_RGBA_ASTC_6x6:
    case GL_COMPRESSED_RGBA_ASTC_8x5:
    case GL_COMPRESSED_RGBA_ASTC_8x6:
    case GL_COMPRESSED_RGBA_ASTC_8x8:
    case GL_COMPRESSED_RGBA_ASTC_10x5:
    case GL_COMPRESSED_RGBA_ASTC_10x6:
    case GL_COMPRESSED_RGBA_ASTC_10x8:
    case GL_COMPRESSED_RGBA_ASTC_10x10:
    case GL_COMPRESSED_RGBA_ASTC_12x10:
    case GL_COMPRESSED_RGBA_ASTC_12x12:
    case GL_COMPRESSED_RGBA8_ETC2_EAC:
    case GL_COMPRESSED_RGB8_PUNCHTHROUGH_ALPHA1_ETC2:
    case GL_COMPRESSED_RGBA_S3TC_DXT1_EXT:
    case GL_COMPRESSED_RGBA_S3TC_DXT3_EXT:
    case GL_COMPRESSED_RGBA_S3TC_DXT5_EXT:
    case GL_ATC_RGBA_EXPLICIT_ALPHA_AMD:
    case GL_ATC_RGBA_INTERPOLATED_ALPHA_AMD:
      return ReadCompressedTexture(tex, idx, GL_RGBA8,
                                   {GL_RED, GL_GREEN, GL_BLUE, GL_ALPHA});
    /* compressed 8bit SRGB */
    case GL_COMPRESSED_SRGB8_ETC2:
      // GL_SRGB8 (i.e. without alpha) is not color renderable.
      return ReadCompressedTexture(tex, idx, GL_SRGB8_ALPHA8,
                                   {GL_RED, GL_GREEN, GL_BLUE, GL_ONE});
    /* compressed 8bit SRGBA */
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_4x4:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x4:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_5x5:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x5:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_6x6:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x5:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x6:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_8x8:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x5:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x6:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x8:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_10x10:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x10:
    case GL_COMPRESSED_SRGB8_ALPHA8_ASTC_12x12:
    case GL_COMPRESSED_SRGB8_ALPHA8_ETC2_EAC:
    case GL_COMPRESSED_SRGB8_PUNCHTHROUGH_ALPHA1_ETC2:
      return ReadCompressedTexture(tex, idx, GL_SRGB8_ALPHA8,
                                   {GL_RED, GL_GREEN, GL_BLUE, GL_ALPHA});
    /* compressed 11bit R - Half floats have 11bit mantissa. */
    case GL_COMPRESSED_R11_EAC:
    case GL_COMPRESSED_SIGNED_R11_EAC:
      return ReadCompressedTexture(tex, idx, GL_R16F,
                                   {GL_RED, GL_ZERO, GL_ZERO, GL_ONE});
    /* compressed 11 bit RG - Half floats have 11bit mantissa. */
    case GL_COMPRESSED_RG11_EAC:
    case GL_COMPRESSED_SIGNED_RG11_EAC:
      return ReadCompressedTexture(tex, idx, GL_RG16F,
                                   {GL_RED, GL_GREEN, GL_ZERO, GL_ONE});
    /* formats that can be used as render targets */
    default: {
      // Need to use GL_FRAMEBUFFER (GL_READ_FRAMEBUFFER fails) due to
      // driver bug b/115574126.
      auto readFb = CreateAndBindFramebuffer(GL_FRAMEBUFFER);
      if (tex.kind == GL_TEXTURE_CUBE_MAP) {
        uint32_t face = GL_TEXTURE_CUBE_MAP_POSITIVE_X + (idx.layer % 6);
        imports.glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0,
                                       face, tex.id, idx.level);
      } else if (idx.layer == 0) {
        imports.glFramebufferTexture(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0,
                                     tex.id, idx.level);
      } else {
        imports.glFramebufferTextureLayer(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0,
                                          tex.id, idx.level, idx.layer);
      }
      return ReadPixels(tex.w, tex.h);
    }
  }
  return ImageData{};
}

ImageData Reader::ReadTextureViaDrawQuad(const Sampler& sampler,
                                         const texture& tex, index idx,
                                         uint32_t format, swizzle swizzle) {
  GAPID_DEBUG("MEC: Drawing quad to format 0x%x", format);
  CHECK_GL_ERROR("MEC: Entered ReadTextureViaDrawQuad in error state");

  auto drawFb = CreateAndBindFramebuffer(GL_DRAW_FRAMEBUFFER);
  auto tmpTex = CreateAndBindTexture2D(tex.w, tex.h, format);
  imports.glFramebufferTexture(GL_DRAW_FRAMEBUFFER, GL_COLOR_ATTACHMENT0,
                               tmpTex.id(), 0);
  tex.bind(imports);
  CHECK_GL_ERROR("MEC: Failed to create framebuffer");

  auto savedState = tex.getState(imports);
  CHECK_GL_ERROR("MEC: Failed querying texture state");

  tex.prepareToRead(imports, idx.level, swizzle);
  CHECK_GL_ERROR("MEC: Failed setting texture state");

  DrawTexturedQuad(sampler, tex.w, tex.h);

  tex.setState(imports, savedState);
  CHECK_GL_ERROR("MEC: Failed restoring texture state", err);

  texture res = {
      .id = tmpTex.id(),
      .kind = GL_TEXTURE_2D,
      .w = tex.w,
      .h = tex.h,
  };
  return ReadTexture(res, {0, 0}, format);
}

ImageData Reader::ReadRenderbuffer(Renderbuffer* rb) {
  auto img = rb->mImage;
  auto w = img->mWidth;
  auto h = img->mHeight;
  auto format = img->mSizedFormat;
  uint32_t attach = GL_COLOR_ATTACHMENT0;
  switch (img->mDataFormat) {
    case GL_DEPTH_COMPONENT:
      attach = GL_DEPTH_ATTACHMENT;
      break;
    case GL_DEPTH_STENCIL:
      attach = GL_DEPTH_STENCIL_ATTACHMENT;
      break;
    case GL_STENCIL_INDEX:
      attach = GL_STENCIL_ATTACHMENT;
      break;
  }
  GAPID_DEBUG(
      "MEC: Reading renderbuffer %d format 0x%x type 0x%x sized 0x%x at 0x%x",
      rb->mID, img->mDataFormat, img->mDataType, format, attach);
  if (attach == GL_COLOR_ATTACHMENT0) {
    // Need to use GL_FRAMEBUFFER (GL_READ_FRAMEBUFFER fails) due to
    // driver bug b/115574126.
    auto readFb = CreateAndBindFramebuffer(GL_FRAMEBUFFER);
    imports.glFramebufferRenderbuffer(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0,
                                      GL_RENDERBUFFER, rb->mID);
    return ReadPixels(w, h);
  } else {
    // Copy the renderbuffer data to temporary texture and then use the texture
    // reading path.
    auto readFb = CreateAndBindFramebuffer(GL_READ_FRAMEBUFFER);
    auto drawFb = CreateAndBindFramebuffer(GL_DRAW_FRAMEBUFFER);
    auto tmpTex = CreateAndBindTexture2D(w, h, format);
    imports.glFramebufferRenderbuffer(GL_READ_FRAMEBUFFER, attach,
                                      GL_RENDERBUFFER, rb->mID);
    imports.glFramebufferTexture(GL_DRAW_FRAMEBUFFER, attach, tmpTex.id(), 0);
    uint32_t mask =
        GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT | GL_STENCIL_BUFFER_BIT;
    imports.glBlitFramebuffer(0, 0, w, h, 0, 0, w, h, mask, GL_NEAREST);

    texture res = {
        .id = tmpTex.id(),
        .kind = GL_TEXTURE_2D,
        .w = w,
        .h = h,
    };
    return ReadTexture(res, {0, 0}, format);
  }
}

ImageData Reader::ReadExternal(EGLImageKHR handle, GLsizei w, GLsizei h) {
  GAPID_DEBUG("MEC: Reading external texture 0x%p", handle);
  auto extTex = CreateAndBindTextureExternal(handle);
  auto tmpTex = CreateAndBindTexture2D(w, h, GL_RGBA8);
  auto fb = CreateAndBindFramebuffer(GL_FRAMEBUFFER);
  imports.glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0,
                                 GL_TEXTURE_2D, tmpTex.id(), 0);
  DrawTexturedQuad(SamplerExternal::get(), w, h);
  return ReadPixels(w, h);
}

void SerializeAndUpdate(StateSerializer* serializer,
                        gapil::Ref<gapii::Image> current,
                        const ImageData& read) {
  if (read.data) {
    serializer->encodeBuffer<uint8_t>(
        read.data->size(), &current->mData,
        [serializer, &read](memory::Observation* obs) {
          serializer->sendData(obs, false, read.data->data(),
                               read.data->size());
        });
    current->mDataFormat = read.dataFormat;
    current->mDataType = read.dataType;
    if (read.sizedFormat) {
      current->mSizedFormat = read.sizedFormat;
    }
  }
}

void SerializeRenderBuffer(Reader& r, StateSerializer* serializer,
                           gapil::Ref<gapii::Renderbuffer> rb) {
  auto img = rb->mImage;
  if (img != nullptr) {
    auto newImg = r.ReadRenderbuffer(rb.get());
    SerializeAndUpdate(serializer, img, newImg);
  }
}

void SerializeTexture(Reader& r, StateSerializer* serializer,
                      gapil::Ref<gapii::Texture> tex) {
  texture out{
      .id = tex->mID,
      .kind = tex->mKind,
  };
  for (auto it : tex->mLevels) {
    auto level = it.first;
    out.d = it.second.mLayers.count();
    for (auto it2 : it.second.mLayers) {
      auto layer = it2.first;
      auto img = it2.second;
      if (img->mSamples != 0) {
        GAPID_WARNING("MEC: Reading multisample textures is not yet supported");
        continue;
      }
      out.w = img->mWidth;
      out.h = img->mHeight;
      auto newImg = r.ReadTexture(out, {level, layer}, img->mSizedFormat);
      SerializeAndUpdate(serializer, img, newImg);
    }
  }
}

}  // anonymous namespace

namespace gapii {
using namespace GLenum;
using namespace GLbitfield;
using namespace EGLenum;

void GlesSpy::GetEGLImageData(CallObserver* observer, EGLImageKHR handle,
                              GLsizei width, GLsizei height) {
  if (!should_trace(kApiIndex)) {
    return;
  }
  Reader read(mImports);

  GAPID_DEBUG("MEC: Get EGLImage data: 0x%p x%xx%x", handle, width, height);
  auto tmpCtx = read.CreateAndBindContext(nullptr, 2);
  if (tmpCtx.id() == EGL_NO_CONTEXT) {
    return;
  }

  auto img = read.ReadExternal(handle, width, height);

  if (!img.data->empty()) {
    auto resIndex = sendResource(kApiIndex, img.data->data(), img.data->size());
    gles_pb::EGLImageData extra;
    extra.set_res_index(resIndex);
    extra.set_size(img.data->size());
    extra.set_width(width);
    extra.set_height(height);
    extra.set_format(img.dataFormat);
    extra.set_type(img.dataType);
    observer->encode_message(&extra);
  }
}

void GlesSpy::serializeGPUBuffers(StateSerializer* serializer) {
  // Ensure we process shared objects only once.
  std::unordered_set<const void*> seen;
  auto once = [&](const void* ptr) { return seen.emplace(ptr).second; };

  Reader r(mImports);
  for (auto& it : mState.EGLContexts) {
    auto handle = it.first;
    auto ctx = it.second;
    if (ctx->mOther.mDestroyed) {
      continue;
    }
    GAPID_DEBUG("MEC: processing context %d thread %s", ctx->mIdentifier,
                ctx->mOther.mThreadName.c_str());

    auto tmpCtx = r.CreateAndBindContext(handle, 3);
    if (tmpCtx.id() == EGL_NO_CONTEXT) {
      continue;
    }

    if (once(ctx->mObjects.mRenderbuffers.instance_ptr())) {
      for (auto& it : ctx->mObjects.mRenderbuffers) {
        SerializeRenderBuffer(r, serializer, it.second);
      }
    }
    if (once(ctx->mObjects.mTextures.instance_ptr())) {
      for (auto& it : ctx->mObjects.mTextures) {
        auto ext = it.second->mEGLImage.get();
        if (ext != nullptr) {
          if (once(ext)) {
            for (auto& it : ext->mImages) {
              auto img = it.second;
              auto newImg = r.ReadExternal(ext->mID, img->mWidth, img->mHeight);
              SerializeAndUpdate(serializer, img, newImg);
            }
          }
        } else {
          SerializeTexture(r, serializer, it.second);
        }
      }
    }
    /* TODO: read buffers from GPU. Currently disabled due to buffer data
             being required by draw calls. Need to be able to determine,
             which buffers have been written to by the GPU.
    if (once(ctx->mObjects.mBuffers.instance_ptr())) {
      for (auto& it : ctx->mObjects.mBuffers) {
        auto buffer = it.second;
        size_t size = buffer->mSize;
        if (size == 0) {
          continue;
        }
        GAPID_DEBUG("MEC: Reading buffer %d size %zu", buffer->mID, size);

        void* data;
        const uint32_t target = GL_ARRAY_BUFFER;
        if (buffer->mMapped) {
          if (buffer->mMapOffset != 0 ||
              static_cast<size_t>(buffer->mMapLength) != size) {
            // TODO: Implement - We can not unmap and remap since the
            // application has the existing pointer, and we can't copy the
            // buffer since copy is not allowed for mapped buffers.
            // Proposed solution: change glMapBuffer* to always map the whole
            // buffer, and return pointer inside that buffer to the user.
            GAPID_WARNING("MEC: Can not read partially mapped buffer")
            continue;
          }
          GAPID_DEBUG("MEC: buffer is application mapped");
          data = buffer->mMapPointer;
        } else {
          mImports.glBindBuffer(target, buffer->mID);
          data = mImports.glMapBufferRange(target, 0, size, GL_MAP_READ_BIT);
        }
        buffer->mData = serializer->encodeBuffer<uint8_t>(
            size, [=](memory::Observation* obs) {
              serializer->sendData(obs, false, data, size);
            });
        if (!buffer->mMapped) {
          mImports.glUnmapBuffer(target);
        }
      }
    }
    */
  }

  GAPID_INFO("MEC: done");
}

}  // namespace gapii
