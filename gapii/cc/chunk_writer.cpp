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

#include "chunk_writer.h"
#include "protocol.h"

#include "core/cc/stream_writer.h"

using namespace gapii::protocol;

namespace {

constexpr size_t kBufferSize = 32 * 1024;

class ChunkWriterImpl : public gapii::ChunkWriter {
 public:
  ChunkWriterImpl(const std::shared_ptr<core::StreamWriter>& writer,
                  bool no_buffer = false);

  ~ChunkWriterImpl();

  // virtual from core::StringWriter
  bool write(std::initializer_list<std::string*> strings) override;
  void flush() override;

 private:
  // returns effective buffer size without reserved space
  size_t getBufferSize() const { return mBuffer.size() - kHeaderSize; }

  std::string mBuffer;

  std::shared_ptr<core::StreamWriter> mWriter;

  bool mStreamGood;

  bool mNoBuffer;
};

ChunkWriterImpl::ChunkWriterImpl(
    const std::shared_ptr<core::StreamWriter>& writer, bool no_buffer)
    // always reserve space for protocol header size at buffer start
    : mBuffer(kHeaderSize, '\0'),
      mWriter(writer),
      mStreamGood(true),
      mNoBuffer(no_buffer) {}

ChunkWriterImpl::~ChunkWriterImpl() {
  if (getBufferSize() > 0u && mStreamGood) {
    flush();
  }
}

bool ChunkWriterImpl::write(std::initializer_list<std::string*> strings) {
  if (mStreamGood) {
    for (auto* s : strings) {
      mBuffer.append(*s);
    }
    if (mNoBuffer || (getBufferSize() >= kBufferSize)) {
      flush();
    }
  }
  return mStreamGood;
}

void ChunkWriterImpl::flush() {
  size_t buf_size = getBufferSize();
  if (buf_size > 0u) {
    // replace reserved space at start of buffer with actual header
    writeHeader(reinterpret_cast<uint8_t*>(&mBuffer.front()),
                MessageType::kData, buf_size);

    // send buffer including header with a single write command
    mStreamGood =
        mWriter->write(mBuffer.data(), mBuffer.size()) == mBuffer.size();

    // continue to reserve protocol header size
    mBuffer.resize(kHeaderSize);
  }
}

}  // anonymous namespace

namespace gapii {

// create returns a shared pointer to a ChunkWriter that writes to
// stream_writer.
ChunkWriter::SPtr ChunkWriter::create(
    const std::shared_ptr<core::StreamWriter>& stream_writer, bool no_buffer) {
  return ChunkWriter::SPtr(new ChunkWriterImpl(stream_writer, no_buffer));
}

}  // namespace gapii
