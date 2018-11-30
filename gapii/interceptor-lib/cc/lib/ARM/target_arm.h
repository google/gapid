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

#ifndef INTERCEPTOR_ARM_TARGET_ARM_H_
#define INTERCEPTOR_ARM_TARGET_ARM_H_

#include "target.h"

namespace interceptor {

class TargetARM : public Target {
 public:
  CodeGenerator *GetCodeGenerator(void *address,
                                  size_t start_alignment) override;

  Disassembler *CreateDisassembler(void *address) override;

  size_t GetCodeAlignment() const override { return 4; }

  void *GetLoadAddress(void *addr) override;

  std::vector<TrampolineConfig> GetTrampolineConfigs(
      uintptr_t start_address) const override;

  Error EmitTrampoline(const TrampolineConfig &config, CodeGenerator &codegen,
                       void *target) override;

  Error RewriteInstruction(const llvm::MCInst &inst, CodeGenerator &codegen,
                           void *data, size_t offset,
                           bool &possible_end_of_function) override;

  void *FixupCallbackFunction(void *old_function, void *new_function) override;

 private:
  enum TrampolineType {
    FULL_TRAMPOLINE = 0,  // Full trampoline with an absolute jump
  };
};

}  // end of namespace interceptor

#endif  // INTERCEPTOR_ARM_TARGET_ARM_H_
