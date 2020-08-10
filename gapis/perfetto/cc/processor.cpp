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

#include "gapis/perfetto/service/perfetto.pb.h"
#include "perfetto/trace_processor/trace_processor.h"

#include "processor.h"

namespace p = perfetto;
namespace ptp = perfetto::trace_processor;

processor new_processor() {
  ptp::Config config;
  // TODO(b/154156099): In Android S, AGI should use the time when trace starts
  // instead of the time when all data sources ack the start.
  config.drop_ftrace_data_before =
      ptp::DropFtraceDataBefore::kAllDataSourcesStarted;
  return ptp::TraceProcessor::CreateInstance(config).release();
}

bool parse_data(processor processor, const void* data, size_t size) {
  ptp::TraceProcessor* p = static_cast<ptp::TraceProcessor*>(processor);
  std::unique_ptr<uint8_t[]> buf(new uint8_t[size]);
  memcpy(buf.get(), data, size);
  // TODO: return the error message.
  if (!p->Parse(std::move(buf), size).ok()) {
    return false;
  }
  p->NotifyEndOfFile();
  return true;
}

result execute_query(processor processor, const char* query) {
  ptp::TraceProcessor* p = static_cast<ptp::TraceProcessor*>(processor);
  p::QueryResult raw;

  auto it = p->ExecuteQuery(query);

  for (uint32_t col = 0; col < it.ColumnCount(); col++) {
    auto* descriptor = raw.add_column_descriptors();
    descriptor->set_name(it.GetColumnName(col));
    descriptor->set_type(p::QueryResult::ColumnDesc::UNKNOWN);
    raw.add_columns();
  }

  uint32_t rows = 0;
  for (; it.Next(); rows++) {
    for (uint32_t col = 0; col < it.ColumnCount(); col++) {
      auto* column = raw.mutable_columns(static_cast<int>(col));
      auto* desc = raw.mutable_column_descriptors(static_cast<int>(col));
      auto value = it.Get(col);

      switch (desc->type() << 8 | value.type) {
        // Nulls.
        case p::QueryResult::ColumnDesc::UNKNOWN << 8 | ptp::SqlValue::kNull:
          // Don't yet know the column type. Add a null value for each.
          column->add_long_values(0);
          column->add_double_values(0);
          column->add_string_values("");
          column->add_is_nulls(true);
          break;
        case p::QueryResult::ColumnDesc::LONG << 8 | ptp::SqlValue::kNull:
          column->add_long_values(0);
          column->add_is_nulls(true);
          break;
        case p::QueryResult::ColumnDesc::DOUBLE << 8 | ptp::SqlValue::kNull:
          column->add_double_values(0);
          column->add_is_nulls(true);
          break;
        case p::QueryResult::ColumnDesc::STRING << 8 | ptp::SqlValue::kNull:
          column->add_string_values("");
          column->add_is_nulls(true);
          break;

        // Values matching the type.
        case p::QueryResult::ColumnDesc::UNKNOWN << 8 | ptp::SqlValue::kString:
          desc->set_type(p::QueryResult::ColumnDesc::STRING);
          column->clear_long_values();
          column->clear_double_values();
          // fall-through.
        case p::QueryResult::ColumnDesc::STRING << 8 | ptp::SqlValue::kString:
          column->add_string_values(value.string_value);
          column->add_is_nulls(false);
          break;
        case p::QueryResult::ColumnDesc::UNKNOWN << 8 | ptp::SqlValue::kLong:
          desc->set_type(p::QueryResult::ColumnDesc::LONG);
          column->clear_string_values();
          column->clear_double_values();
          // fall-through.
        case p::QueryResult::ColumnDesc::LONG << 8 | ptp::SqlValue::kLong:
          column->add_long_values(value.long_value);
          column->add_is_nulls(false);
          break;
        case p::QueryResult::ColumnDesc::UNKNOWN << 8 | ptp::SqlValue::kDouble:
          desc->set_type(p::QueryResult::ColumnDesc::DOUBLE);
          column->clear_string_values();
          column->clear_long_values();
          // fall-through.
        case p::QueryResult::ColumnDesc::DOUBLE << 8 | ptp::SqlValue::kDouble:
          column->add_double_values(value.double_value);
          column->add_is_nulls(false);
          break;

        // Values needing conversion.
        case p::QueryResult::ColumnDesc::LONG << 8 | ptp::SqlValue::kDouble:
          // TODO: should we "upgrade" the column to double?
          column->add_long_values(static_cast<int64_t>(value.double_value));
          column->add_is_nulls(false);
          break;
        case p::QueryResult::ColumnDesc::DOUBLE << 8 | ptp::SqlValue::kLong:
          column->add_double_values(static_cast<double>(value.long_value));
          column->add_is_nulls(false);
          break;
        default:
          // ignore mismatched numeric/string values.
          break;
      }
    }
  }

  raw.set_num_records(rows);

  auto status = it.Status();
  if (!status.ok()) {
    raw.set_error(status.message());
  }

  result res;
  res.size = raw.ByteSizeLong();
  res.data = new uint8_t[res.size];
  raw.SerializeWithCachedSizesToArray(res.data);
  return res;
}

void delete_processor(processor processor) { free(processor); }
