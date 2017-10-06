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

#ifndef CORE_STRING_WRITER_H
#define CORE_STRING_WRITER_H

#include <memory>
#include <string>

namespace core {
class StreamWriter;

// StringWriter is a pure virtual class used to write strings to a StreamWriter.
class StringWriter {
public:
    typedef std::shared_ptr<StringWriter> SPtr;

    // write attempts to write the string 'data' to the underlying stream,
    // returning false upon failure.  'data' may be in an unknown state past
    // this call, as implementations of this interface may use move semantics
    // as a memory optimization.
    virtual bool write(std::string& data) = 0;

    // flush flushes out all of the pending in the steam
    virtual void flush() = 0;

protected:
    virtual ~StringWriter() {}
};

}  // namespace core

#endif  // CORE_STRING_WRITER_H
