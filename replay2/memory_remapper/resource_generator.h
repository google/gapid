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

#ifndef REPLAY2_MEMORY_REMAPPER_RESOURCE_GENERATOR_H
#define REPLAY2_MEMORY_REMAPPER_RESOURCE_GENERATOR_H

#include "replay_address.h"

#include <cstddef>
#include <memory>

namespace agi {
namespace replay2 {

class ResourceGenerator {
   public:
    virtual ~ResourceGenerator() {}

    virtual size_t length() const = 0;
    virtual void generate(ReplayAddress replayAddress) = 0;
};
typedef std::shared_ptr<ResourceGenerator> ResourceGeneratorPtr;

class NullResourceGenerator : public ResourceGenerator {
   public:
    NullResourceGenerator(size_t length) : length_(length) {}
    virtual ~NullResourceGenerator() {}

    size_t length() const override { return length_; }

    void generate(ReplayAddress replayAddress) override {}

   private:
    size_t length_;
};

}  // namespace replay2
}  // namespace agi

#endif
