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

#include <google/protobuf/text_format.h>
#include <string.h>

#include "perfetto/trace_processor/raw_query.pb.h"
#include "perfetto/trace_processor/trace_processor.h"

#include "processor.h"

namespace ptp = perfetto::trace_processor;
namespace pp = perfetto::protos;

// Callback back into go.
extern "C" void on_query_complete(int id, void* data, long unsigned int size);

processor new_processor() {
  ptp::Config config;
  return ptp::TraceProcessor::CreateInstance(config).release();
}

bool parse_data(processor processor, const void* data, size_t size) {
  ptp::TraceProcessor* p = static_cast<ptp::TraceProcessor*>(processor);
  std::unique_ptr<uint8_t[]> buf(new uint8_t[size]);
  memcpy(buf.get(), data, size);
  if (!p->Parse(std::move(buf), size)) {
    return false;
  }
  p->NotifyEndOfFile();
  return true;
}

void execute_query(processor processor, int id, const char* query) {
  ptp::TraceProcessor* p = static_cast<ptp::TraceProcessor*>(processor);
  pp::RawQueryArgs args;
  args.set_sql_query(query);
  p->ExecuteQuery(args, [id](const pp::RawQueryResult& r) {
    std::string data;
    r.SerializeToString(&data);
    on_query_complete(
        id, const_cast<void*>(reinterpret_cast<const void*>(data.data())),
        data.size());
  });
}

void delete_processor(processor processor) { free(processor); }
