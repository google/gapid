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

#ifndef GAPIR_RESOURCE_PROVIDER_H
#define GAPIR_RESOURCE_PROVIDER_H

#include "resource.h"

#include <stdint.h>

namespace gapir {

class ReplayConnection;

class ResourceProvider {
public:
    virtual ~ResourceProvider() {}

    // Loads count resources from the provider and writes them, in-order, to target.
    // If the net size of all the resources exceeds size, then false is returned.
    virtual bool get(const Resource* resources, size_t count, ReplayConnection* conn,
                     void* target, size_t targetSize) = 0;

    // Prefetches the resources for resource providers where prefetching is available.
    // temp is a temporary buffer of size tempSize that can be used by prefetch.
    virtual void prefetch(const Resource* resources, size_t count, ReplayConnection* conn, 
    											void* temp, size_t tempSize) = 0;
};

}  // namespace gapir

#endif  // GAPIR_RESOURCE_PROVIDER_H
