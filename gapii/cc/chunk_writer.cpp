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

#include <google/protobuf/io/coded_stream.h>

#include "core/cc/stream_writer.h"

using ::google::protobuf::io::CodedOutputStream;

namespace {

constexpr size_t kBufferSize = 32 * 1024;

class ChunkWriterImpl : public gapii::ChunkWriter {
 public:
  ChunkWriterImpl(const std::shared_ptr<core::StreamWriter>& writer,
                  bool paranoid = false);

  ~ChunkWriterImpl();

  virtual bool write(std::string& s) override;
  virtual void flush() override;

 private:
  std::string mBuffer;

  std::shared_ptr<core::StreamWriter> mWriter;

  bool mStreamGood;

  bool mNoBuffer;
};

ChunkWriterImpl::ChunkWriterImpl(
    const std::shared_ptr<core::StreamWriter>& writer, bool no_buffer)
    : mWriter(writer), mStreamGood(true), mNoBuffer(no_buffer) {}

ChunkWriterImpl::~ChunkWriterImpl() {
  if (mBuffer.size() && mStreamGood) {
    flush();
  }
}

bool ChunkWriterImpl::write(std::string& s) {
  if (mStreamGood) {
    mBuffer.append(s);

    if (mNoBuffer || (mBuffer.size() >= kBufferSize)) {
      flush();
    }
  }

  return mStreamGood;
}

void ChunkWriterImpl::flush() {
  size_t bufferSize = mBuffer.size();
  mStreamGood = mWriter->write(mBuffer.data(), mBuffer.size()) == bufferSize;
  mBuffer.clear();
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
