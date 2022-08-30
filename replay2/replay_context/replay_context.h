// Copyright (C) 2022 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "replay2/core_utils/non_copyable.h"
#include "replay2/handle_remapper/handle_remapper.h"
#include "replay2/memory_remapper/memory_remapper.h"

#include <string>

namespace agi {
namespace replay2 {

class ReplayContext : public NonCopyable {
   public:
    ReplayContext(const std::string& replayIdentifier) : replayIdentifier_(replayIdentifier) {}

    const std::string& getReplayIdentifier() const { return replayIdentifier_; }

    HandleRemapper& getHandleRemapper() { return handleRemapper_; }
    MemoryRemapper& getMemoryRemapper() { return memoryRemapper_; }

   private:
    std::string replayIdentifier_;

    HandleRemapper handleRemapper_;
    MemoryRemapper memoryRemapper_;
};

}  // namespace replay2
}  // namespace agi
