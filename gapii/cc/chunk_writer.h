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

#ifndef GAPII_CHUNK_STRING_WRITER_H
#define GAPII_CHUNK_STRING_WRITER_H

#include "core/cc/string_writer.h"

namespace gapii {

// ChunkWriter is used to write chunk strings to a core::StreamWriter.
class ChunkWriter : public core::StringWriter {
public:
    static SPtr create(const std::shared_ptr<core::StreamWriter>& stream_writer);

protected:
    ~ChunkWriter() = default;
};

}  // namespace gapii

#endif  // GAPII_CHUNK_STRING_WRITER_H
