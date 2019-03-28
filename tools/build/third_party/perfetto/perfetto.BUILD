# Copyright (C) 2019 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@gapid//tools/build:rules.bzl", "cc_copts", "copy_tree")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    embed = [
        ":config_go_proto",
        ":raw_query_go_proto",
    ],
    importpath = "perfetto_pb",
    visibility = ["//visibility:public"],
)

cc_library(
    name = "trace_processor",
    srcs = [
        "src/base/string_splitter.cc",
        "src/base/string_utils.cc",
        "src/base/time.cc",
        "src/protozero/message.cc",
        "src/protozero/message_handle.cc",
        "src/protozero/proto_decoder.cc",
        "src/protozero/proto_field_descriptor.cc",
        "src/protozero/scattered_heap_buffer.cc",
        "src/protozero/scattered_stream_null_delegate.cc",
        "src/protozero/scattered_stream_writer.cc",
        "src/trace_processor/android_logs_table.cc",
        "src/trace_processor/args_table.cc",
        "src/trace_processor/args_tracker.cc",
        "src/trace_processor/clock_tracker.cc",
        "src/trace_processor/counter_definitions_table.cc",
        "src/trace_processor/counter_values_table.cc",
        "src/trace_processor/event_tracker.cc",
        "src/trace_processor/filtered_row_index.cc",
        "src/trace_processor/ftrace_descriptors.cc",
        "src/trace_processor/ftrace_utils.cc",
        "src/trace_processor/instants_table.cc",
        "src/trace_processor/json_trace_parser_stub.cc",
        "src/trace_processor/process_table.cc",
        "src/trace_processor/process_tracker.cc",
        "src/trace_processor/proto_trace_parser.cc",
        "src/trace_processor/proto_trace_tokenizer.cc",
        "src/trace_processor/query_constraints.cc",
        "src/trace_processor/raw_table.cc",
        "src/trace_processor/row_iterators.cc",
        "src/trace_processor/sched_slice_table.cc",
        "src/trace_processor/slice_table.cc",
        "src/trace_processor/slice_tracker.cc",
        "src/trace_processor/span_join_operator_table.cc",
        "src/trace_processor/sql_stats_table.cc",
        "src/trace_processor/sqlite3_str_split.cc",
        "src/trace_processor/stats_table.cc",
        "src/trace_processor/storage_columns.cc",
        "src/trace_processor/storage_schema.cc",
        "src/trace_processor/storage_table.cc",
        "src/trace_processor/string_table.cc",
        "src/trace_processor/table.cc",
        "src/trace_processor/thread_table.cc",
        "src/trace_processor/trace_processor.cc",
        "src/trace_processor/trace_processor_context.cc",
        "src/trace_processor/trace_processor_impl.cc",
        "src/trace_processor/trace_sorter.cc",
        "src/trace_processor/trace_storage.cc",
        "src/trace_processor/virtual_destructors.cc",
        "src/trace_processor/window_operator_table.cc",
    ] + glob([
        "include/**/*.h",
        "src/trace_processor/*.h",
    ]),
    hdrs = glob(["include/**/*.h"]),
    copts = cc_copts() + [
        "-Iexternal/perfetto/sqlite",
        # Always build in optimized mode.
        "-O2",
        "-DNDEBUG",
    ] + select({
        "@gapid//tools/build:windows": ["-D__STDC_FORMAT_MACROS"],
        "//conditions:default": [],
    }),
    strip_include_prefix = "include",
    visibility = ["//visibility:public"],
    deps = [
        ":trace_cc_proto",
        ":trace_processor_cc_proto",
        "//sqlite",
    ],
)

proto_library(
    name = "config_proto",
    srcs = ["perfetto/config/perfetto_config.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "config_go_proto",
    importpath = "perfetto_pb",
    proto = ":config_proto",
    visibility = ["//visibility:public"],
)

java_proto_library(
    name = "config_java_proto",
    visibility = ["//visibility:public"],
    deps = [":config_proto"],
)

proto_library(
    name = "raw_query_proto",
    srcs = ["perfetto/trace_processor/raw_query.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "raw_query_go_proto",
    importpath = "perfetto_pb",
    proto = ":raw_query_proto",
    visibility = ["//visibility:public"],
)

proto_library(
    name = "trace_proto",
    srcs = [
        "perfetto/common/android_log_constants.proto",
        "perfetto/common/commit_data_request.proto",
        "perfetto/common/observable_events.proto",
        "perfetto/common/sys_stats_counters.proto",
        "perfetto/common/trace_stats.proto",
        "perfetto/config/android/android_log_config.proto",
        "perfetto/config/chrome/chrome_config.proto",
        "perfetto/config/data_source_config.proto",
        "perfetto/config/data_source_descriptor.proto",
        "perfetto/config/ftrace/ftrace_config.proto",
        "perfetto/config/inode_file/inode_file_config.proto",
        "perfetto/config/power/android_power_config.proto",
        "perfetto/config/process_stats/process_stats_config.proto",
        "perfetto/config/profiling/heapprofd_config.proto",
        "perfetto/config/sys_stats/sys_stats_config.proto",
        "perfetto/config/test_config.proto",
        "perfetto/config/trace_config.proto",
        "perfetto/trace/android/android_log.proto",
        "perfetto/trace/chrome/chrome_trace_event.proto",
        "perfetto/trace/clock_snapshot.proto",
        "perfetto/trace/filesystem/inode_file_map.proto",
        "perfetto/trace/ftrace/binder.proto",
        "perfetto/trace/ftrace/block.proto",
        "perfetto/trace/ftrace/cgroup.proto",
        "perfetto/trace/ftrace/clk.proto",
        "perfetto/trace/ftrace/compaction.proto",
        "perfetto/trace/ftrace/ext4.proto",
        "perfetto/trace/ftrace/f2fs.proto",
        "perfetto/trace/ftrace/fence.proto",
        "perfetto/trace/ftrace/filemap.proto",
        "perfetto/trace/ftrace/ftrace.proto",
        "perfetto/trace/ftrace/ftrace_event.proto",
        "perfetto/trace/ftrace/ftrace_event_bundle.proto",
        "perfetto/trace/ftrace/ftrace_stats.proto",
        "perfetto/trace/ftrace/generic.proto",
        "perfetto/trace/ftrace/i2c.proto",
        "perfetto/trace/ftrace/ipi.proto",
        "perfetto/trace/ftrace/irq.proto",
        "perfetto/trace/ftrace/kmem.proto",
        "perfetto/trace/ftrace/lowmemorykiller.proto",
        "perfetto/trace/ftrace/mdss.proto",
        "perfetto/trace/ftrace/mm_event.proto",
        "perfetto/trace/ftrace/oom.proto",
        "perfetto/trace/ftrace/power.proto",
        "perfetto/trace/ftrace/raw_syscalls.proto",
        "perfetto/trace/ftrace/regulator.proto",
        "perfetto/trace/ftrace/sched.proto",
        "perfetto/trace/ftrace/signal.proto",
        "perfetto/trace/ftrace/sync.proto",
        "perfetto/trace/ftrace/task.proto",
        "perfetto/trace/ftrace/test_bundle_wrapper.proto",
        "perfetto/trace/ftrace/vmscan.proto",
        "perfetto/trace/ftrace/workqueue.proto",
        "perfetto/trace/interned_data/interned_data.proto",
        "perfetto/trace/power/battery_counters.proto",
        "perfetto/trace/power/power_rails.proto",
        "perfetto/trace/profiling/profile_packet.proto",
        "perfetto/trace/ps/process_stats.proto",
        "perfetto/trace/ps/process_tree.proto",
        "perfetto/trace/sys_stats/sys_stats.proto",
        "perfetto/trace/system_info.proto",
        "perfetto/trace/test_event.proto",
        "perfetto/trace/trace.proto",
        "perfetto/trace/trace_packet.proto",
        "perfetto/trace/track_event/debug_annotation.proto",
        "perfetto/trace/track_event/process_descriptor.proto",
        "perfetto/trace/track_event/task_execution.proto",
        "perfetto/trace/track_event/thread_descriptor.proto",
        "perfetto/trace/track_event/track_event.proto",
    ],
)

cc_proto_library(
    name = "trace_cc_proto",
    deps = ["//:trace_proto"],
)

proto_library(
    name = "trace_processor_proto",
    srcs = [
        "perfetto/trace_processor/raw_query.proto",
        "perfetto/trace_processor/sched.proto",
        "perfetto/trace_processor/trace_processor.proto",
    ],
)

cc_proto_library(
    name = "trace_processor_cc_proto",
    deps = ["//:trace_processor_proto"],
)
