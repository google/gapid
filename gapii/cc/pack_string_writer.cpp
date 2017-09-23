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


#include "pack_string_writer.h"

#include <core/cc/stream_writer.h>

#include <google/protobuf/io/coded_stream.h>

using ::google::protobuf::io::CodedOutputStream;

namespace {

constexpr size_t kBufferSize = 32*1024;

class PackStringWriterImpl : public gapii::PackStringWriter {
public:
    PackStringWriterImpl(const std::shared_ptr<core::StreamWriter>& writer);

    ~PackStringWriterImpl();

    virtual bool write(std::string& s) override;

    virtual std::shared_ptr<core::StreamWriter> getStream() override;

private:
    void flush();

    std::string mBuffer;

    std::shared_ptr<core::StreamWriter> mWriter;

    bool mStreamGood;
};

PackStringWriterImpl::PackStringWriterImpl(const std::shared_ptr<core::StreamWriter>& writer)
        : mWriter(writer)
        , mStreamGood(true) {
}

PackStringWriterImpl::~PackStringWriterImpl() {
    if (mBuffer.size() && mStreamGood) {
        flush();
    }
}

bool PackStringWriterImpl::write(std::string& s) {
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

std::shared_ptr<core::StreamWriter> PackStringWriterImpl::getStream() {
    return mWriter;
}

void PackStringWriterImpl::flush() {
    mStreamGood = mWriter->write(mBuffer.data(), mBuffer.size());
    mBuffer.clear();
}

} // anonymous namespace

namespace gapii {

// create returns a PackStringWriter::SPtr that writes to stream_writer.
PackStringWriter::SPtr PackStringWriter::create(const std::shared_ptr<core::StreamWriter>& stream_writer) {
    return PackStringWriter::SPtr(new PackStringWriterImpl(stream_writer));
}

} // namespace gapii
