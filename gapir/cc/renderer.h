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

#ifndef GAPIR_RENDERER_H
#define GAPIR_RENDERER_H

#include "gfx_api.h"

namespace gapir {

// Renderer is an interface to an off-screen rendering context. Constructing a
// renderer will perform any necessary hidden window construction and minimal
// event handling for the target platform.
class Renderer {
 public:
  class Listener {
   public:
    virtual void onDebugMessage(uint32_t severity, uint8_t api_index,
                                const char* msg) = 0;
    virtual ~Listener() {}
  };

  inline Renderer();

  // Destroys the renderer and any associated off-screen windows.
  virtual ~Renderer() {}

  // Returns the renderer's API.
  virtual Api* api() = 0;

  template <typename T>
  inline T* getApi();

  inline void setListener(Listener* listener);
  inline Listener* getListener() const;

  virtual bool isValid() = 0;

 private:
  Listener* mListener;
};

inline Renderer::Renderer() : mListener(nullptr) {}

template <typename T>
inline T* Renderer::getApi() {
  if (Api* api = this->api()) {
    // Raw-pointer comparison is safe as the ID is static.
    if (api->id() == T::ID) {
      return static_cast<T*>(api);
    }
  }
  return nullptr;
}

inline void Renderer::setListener(Listener* listener) { mListener = listener; }
inline Renderer::Listener* Renderer::getListener() const { return mListener; }

}  // namespace gapir

#endif  // GAPIR_RENDERER_H
