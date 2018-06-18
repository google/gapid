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

#ifndef GAPIR_RESOURCE_H
#define GAPIR_RESOURCE_H

#include <stdint.h>

#include <string>

namespace gapir {

typedef std::string ResourceId;

// Resource represent a requestable blob of data from the server.
class Resource {
 public:
  inline Resource();
  inline Resource(const Resource& other);
  inline Resource(ResourceId id, uint32_t size);
  inline bool operator==(const Resource& other) const;

  ResourceId id;  // The resource identifier.
  uint32_t size;  // The resource size in bytes.
};

inline Resource::Resource() {}
inline Resource::Resource(const Resource& other)
    : id(other.id), size(other.size) {}
inline Resource::Resource(ResourceId id_, uint32_t size_)
    : id(id_), size(size_) {}
inline bool Resource::operator==(const Resource& other) const {
  return id == other.id && size == other.size;
}

}  // namespace gapir

#endif  // GAPIR_RESOURCE_H
