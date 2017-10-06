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

#include <core/cc/stream_writer.h>

#include <google/protobuf/io/coded_stream.h>

using ::google::protobuf::io::CodedOutputStream;

namespace {

constexpr size_t kBufferSize = 32*1024;

class ChunkWriterImpl : public gapii::ChunkWriter {
public:
    ChunkWriterImpl(const std::shared_ptr<core::StreamWriter>& writer);

    ~ChunkWriterImpl();

    virtual bool write(std::string& s) override;
    virtual void flush() override;

private:

    std::string mBuffer;

    std::shared_ptr<core::StreamWriter> mWriter;

    bool mStreamGood;
};

ChunkWriterImpl::ChunkWriterImpl(const std::shared_ptr<core::StreamWriter>& writer)
        : mWriter(writer)
        , mStreamGood(true) {
}

ChunkWriterImpl::~ChunkWriterImpl() {
    if (mBuffer.size() && mStreamGood) {
        flush();
    }
}

bool ChunkWriterImpl::write(std::string& s) {
    if (mStreamGood) {
        uint8_t size_buf[16];

        auto size = s.size();
        auto size_count = CodedOutputStream::WriteVarint64ToArray(size, &size_buf[0]) - &size_buf[0];

        mBuffer.append(reinterpret_cast<char*>(&size_buf[0]), size_count);
        mBuffer.append(s);

        if(mBuffer.size() >= kBufferSize) {
            flush();
        }
    }

    return mStreamGood;
}

void ChunkWriterImpl::flush() {
    mStreamGood = mWriter->write(mBuffer.data(), mBuffer.size());
    mBuffer.clear();
}

} // anonymous namespace

namespace gapii {

// create returns a shared pointer to a ChunkWriter that writes to stream_writer.
ChunkWriter::SPtr ChunkWriter::create(const std::shared_ptr<core::StreamWriter>& stream_writer) {
    return ChunkWriter::SPtr(new ChunkWriterImpl(stream_writer));
}

} // namespace gapii
