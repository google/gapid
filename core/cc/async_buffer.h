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

#ifndef CORE_ASYNC_BUFFER_H
#define CORE_ASYNC_BUFFER_H

#include "semaphore.h"
#include "stream_writer.h"

#include <cstring>
#include <memory>
#include <thread>

namespace core {

// AsyncBuffer is a StreamWriter that buffers writes into a fixed size
// ringbuffer which is streamed out to a downstream StreamWriter on another
// thread.
class AsyncBuffer : public StreamWriter {
 public:
  AsyncBuffer(std::shared_ptr<StreamWriter> out, size_t buffer_size);
  ~AsyncBuffer();

  // create constructs and returns an AsyncBuffer in a shared_ptr.
  static inline std::shared_ptr<AsyncBuffer> create(
      std::shared_ptr<StreamWriter> out, size_t buffer_size);

  // write appends data to the ringbuffer, blocking if it is full.
  // write must always be called from the same thread.
  virtual uint64_t write(const void* data, uint64_t size) override;

  // flush ensures all pending writes have been written downstream before
  // returning. flush must be called on the same thread as the writes.
  void flush();

 private:
  // write_chunk writes a block that is smaller than size_
  uint64_t write_chunk(const void* data, uint64_t size);

  void worker();

  uint8_t* buffer_;    // the ringbuffer
  const size_t size_;  // total size of the buffer
  Semaphore written_;  // number of bytes written (pending) in the ring buffer
  Semaphore free_;     // number of bytes free in the ring buffer
  size_t read_head_;   // read head byte offset in the ringbuffer
  size_t write_head_;  // write head byte offset in the ringbuffer
  std::shared_ptr<StreamWriter> out_;  // the downstream StreamWriter
  std::thread* thread_;
};

std::shared_ptr<AsyncBuffer> AsyncBuffer::create(
    std::shared_ptr<StreamWriter> out, size_t buffer_size) {
  return std::shared_ptr<AsyncBuffer>(new AsyncBuffer(out, buffer_size));
}

}  // namespace core

#endif  // CORE_ASYNC_BUFFER_H
