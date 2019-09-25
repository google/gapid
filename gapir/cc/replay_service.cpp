/*
 * Copyright (C) 2018 Google Inc.
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

#include "replay_service.h"

#include <grpc++/grpc++.h>
#include <memory>

#include "core/cc/log.h"
#include "gapir/replay_service/service.grpc.pb.h"
#include "gapis/service/severity/severity.pb.h"

namespace gapir {

// Posts member methods

ReplayService::Posts::~Posts() = default;

bool ReplayService::Posts::append(uint64_t id, const void* data, size_t size) {
  if (mProtoPostData == nullptr) {
    return false;
  }
  auto* piece = mProtoPostData->add_post_data_pieces();
  piece->set_id(id);
  piece->set_data(data, size);
  return true;
}

replay_service::PostData* ReplayService::Posts::release_to_proto() {
  auto ptr = mProtoPostData.release();
  mProtoPostData = nullptr;
  return ptr;
}

size_t ReplayService::Posts::piece_count() const {
  return mProtoPostData->post_data_pieces_size();
}

size_t ReplayService::Posts::piece_size(int index) const {
  return mProtoPostData->post_data_pieces(index).data().size();
}

const void* ReplayService::Posts::piece_data(int index) const {
  return mProtoPostData->post_data_pieces(index).data().data();
}

uint64_t ReplayService::Posts::piece_id(int index) const {
  return mProtoPostData->post_data_pieces(index).id();
}

ReplayService::Posts::Posts()
    : mProtoPostData(new replay_service::PostData()) {}

// FenceReady

ReplayService::FenceReady::FenceReady(
    std::unique_ptr<replay_service::FenceReady> protoFenceReady)
    : mProtoFenceReady(std::move(protoFenceReady)) {}

ReplayService::FenceReady::~FenceReady() = default;

uint32_t ReplayService::FenceReady::id() const {
  return mProtoFenceReady->id();
}

// Payload member methods

ReplayService::Payload::Payload(
    std::unique_ptr<replay_service::Payload> protoPayload)
    : mProtoPayload(std::move(protoPayload)) {}

ReplayService::Payload::~Payload() = default;

uint32_t ReplayService::Payload::stack_size() const {
  return mProtoPayload->stack_size();
}

uint32_t ReplayService::Payload::volatile_memory_size() const {
  return mProtoPayload->volatile_memory_size();
}

size_t ReplayService::Payload::constants_size() const {
  return mProtoPayload->constants().size();
}

const void* ReplayService::Payload::constants_data() const {
  return mProtoPayload->constants().data();
}

size_t ReplayService::Payload::resource_info_count() const {
  return mProtoPayload->resources_size();
}

const std::string ReplayService::Payload::resource_id(int index) const {
  return mProtoPayload->resources(index).id();
}

uint32_t ReplayService::Payload::resource_size(int index) const {
  return mProtoPayload->resources(index).size();
}

size_t ReplayService::Payload::opcodes_size() const {
  return mProtoPayload->opcodes().size();
}

const void* ReplayService::Payload::opcodes_data() const {
  return mProtoPayload->opcodes().data();
}

// Resources member methods

ReplayService::Resources::Resources(
    std::unique_ptr<replay_service::Resources> protoResources)
    : mProtoResources(std::move(protoResources)) {}

ReplayService::Resources::~Resources() = default;

size_t ReplayService::Resources::size() const {
  return mProtoResources->data().size();
}

const void* ReplayService::Resources::data() const {
  return mProtoResources->data().data();
}

}  // namespace gapir
