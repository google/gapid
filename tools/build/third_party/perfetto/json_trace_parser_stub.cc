/*
 * Copyright (C) 2019 Google Inc.
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

// We don't need no stinkin' JSON.
// We don't need no Chrome support.
// No extra symbols in the namespace.
// Hey, cgo, leave GAPID alone!
// All in all it's just another dependency.

#include "src/trace_processor/json_trace_parser.h"


namespace perfetto {
namespace trace_processor {

JsonTraceParser::JsonTraceParser(TraceProcessorContext* ctx) : context_(ctx) {
  (void)context_;
  (void)offset_;
}

JsonTraceParser::~JsonTraceParser() {
}

bool JsonTraceParser::Parse(std::unique_ptr<uint8_t[]>, size_t) {
  return false;
}

}  // namespace trace_processor
}
