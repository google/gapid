/*
 * Copyright (C) 2022 Google Inc.
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

#include "spy_serializer.h"

#include "encoder.h"

namespace gapid2 {

encoder_handle spy_serializer::get_encoder(uintptr_t ptr) {
  while (tid_ != std::thread::id()) {
    if (tid_ == std::this_thread::get_id()) {
      break;
    }
    std::this_thread::yield();
  }
  if (!enabled_) {
    return encoder_handle(nullptr);
  }

  encoder* enc = reinterpret_cast<encoder*>(TlsGetValue(encoder_tls_key));
  if (!enc) {
    enc = new encoder();
    TlsSetValue(encoder_tls_key, enc);
  }
  if (!ptr) {
    enc = new encoder();
  }

  return encoder_handle(enc, [this, enc, ptr]() {
    uint64_t data_size = 0;
    for (size_t i = 0; i <= enc->data_offset; ++i) {
      data_size += enc->data_[i].size - enc->data_[i].left;
    }
    if (!data_size) {
      if (!ptr) {
        delete enc;
      }
      return;
    }
    char dat[sizeof(data_size)];
    memcpy(dat, &data_size, sizeof(data_size));
    call_mutex.lock();
    out_file.write(dat, sizeof(dat));

    for (size_t i = 0; i <= enc->data_offset; ++i) {
      out_file.write(enc->data_[i].data,
                     enc->data_[i].size - enc->data_[i].left);
      GAPID2_ASSERT(!out_file.bad(), "Out file is bad, invalid write?");
    }
    enc->reset();
    call_mutex.unlock();
    if (!ptr) {
      delete enc;
    }
  });
}

encoder_handle spy_serializer::get_locked_encoder(uintptr_t) {
  while (tid_ != std::thread::id()) {
    if (tid_ == std::this_thread::get_id()) {
      break;
    }
    std::this_thread::yield();
  }
  if (!enabled_) {
    return encoder_handle(nullptr);
  }

  encoder* enc = reinterpret_cast<encoder*>(TlsGetValue(encoder_tls_key));
  if (!enc) {
    enc = new encoder();
    TlsSetValue(encoder_tls_key, enc);
  }
  // Well this is just insane :D 
  // The nvidia driver calls DXGI (which might be DXVK) which calls vulkan, which calls gapii
  // which calls this function... which is already in flight. This screws everything up.
  // Solution? I guess we dont serialize re-entrant calls :(
  if (enc->current_->left != enc->current_->size) {
    return encoder_handle(nullptr);
  }

  call_mutex.lock();
  return encoder_handle(enc, [this, enc]() {
    uint64_t data_size = 0;
    for (size_t i = 0; i <= enc->data_offset; ++i) {
      data_size += enc->data_[i].size - enc->data_[i].left;
    }
    if (!data_size) {
      call_mutex.unlock();
      return;
    }
    char dat[sizeof(data_size)];
    memcpy(dat, &data_size, sizeof(data_size));
    out_file.write(dat, sizeof(dat));

    for (size_t i = 0; i <= enc->data_offset; ++i) {
      out_file.write(enc->data_[i].data,
                     enc->data_[i].size - enc->data_[i].left);
    }
    enc->reset();
    call_mutex.unlock();
  });
}

void spy_serializer::enable() {
  tid_ = std::thread::id();
  enabled_ = true;
}

void spy_serializer::enable_with_mec() {
  tid_ = std::this_thread::get_id();
  enabled_ = true;
}

void spy_serializer::disable() {
  out_file.flush();
  enabled_ = false;
}

}  // namespace gapid2