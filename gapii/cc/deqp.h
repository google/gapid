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

#ifndef GAPII_DEQP_H
#define GAPII_DEQP_H

#include "core_spy.h"
#include "gles_spy.h"
#include "return_handler.h"

#include "core/cc/null_writer.h"
#include "core/cc/null_encoder.h"

#include <memory>

namespace gapii {

// GlesNull implements a GAPI wrapper for the dEQP null driver.
class GlesNull : public CoreSpy, public GlesSpy {
 public:
  GlesNull();
  uint32_t glGetError(CallObserver* observer);
  void wrapGetIntegerv(uint32_t param, GLint* values);
  virtual void onThreadSwitched(CallObserver* observer,
                                uint64_t threadID) override;
  void Import(void**);
  std::shared_ptr<ReturnHandler> getReturnHandler();

 private:
  typedef void (*PFNGLGETINTEGERV)(uint32_t param, GLint* values);
  PFNGLGETINTEGERV mImportedGetIntegerv;
  std::shared_ptr<ReturnHandler> mReturnHandler;
};

}  // namespace gapii

#endif  // GAPII_DEQP_H
