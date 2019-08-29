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

load("@gapid//tools/build:rules.bzl", "cc_copts", "cc_stripped_binary")
load("@gapid//tools/build/third_party/perfetto:protozero.bzl", "cc_protozero_library")
load("@gapid//tools/build/third_party/perfetto:ipc.bzl", "cc_ipc_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

_COPTS_BASE = cc_copts() + [
    # Always build in optimized mode.
    "-O2",
    "-DNDEBUG",
] + select({
    "@gapid//tools/build:windows": ["-D__STDC_FORMAT_MACROS"],
    "//conditions:default": [],
})

_COPTS = _COPTS_BASE + [
    "-DPERFETTO_BUILD_WITH_EMBEDDER",
]

###################### Public Libraries and Protos ######################

cc_library(
    name = "trace_processor",
    srcs = [
        "src/trace_processor/android_logs_table.cc",
        "src/trace_processor/args_table.cc",
        "src/trace_processor/args_tracker.cc",
        "src/trace_processor/clock_tracker.cc",
        "src/trace_processor/counter_definitions_table.cc",
        "src/trace_processor/counter_values_table.cc",
        "src/trace_processor/event_tracker.cc",
        "src/trace_processor/filtered_row_index.cc",
        "src/trace_processor/forwarding_trace_parser.cc",
        "src/trace_processor/ftrace_descriptors.cc",
        "src/trace_processor/ftrace_utils.cc",
        "src/trace_processor/fuchsia_provider_view.cc",
        "src/trace_processor/fuchsia_trace_parser.cc",
        "src/trace_processor/fuchsia_trace_tokenizer.cc",
        "src/trace_processor/fuchsia_trace_utils.cc",
        "src/trace_processor/graphics_frame_event_parser.cc",
        "src/trace_processor/gzip_trace_parser.cc",
        "src/trace_processor/heap_profile_allocation_table.cc",
        "src/trace_processor/heap_profile_tracker.cc",
        "src/trace_processor/instants_table.cc",
        "src/trace_processor/metrics/descriptors.cc",
        "src/trace_processor/metrics/metrics.cc",
        "src/trace_processor/metadata_table.cc",
        "src/trace_processor/process_table.cc",
        "src/trace_processor/process_tracker.cc",
        "src/trace_processor/proto_trace_parser.cc",
        "src/trace_processor/proto_trace_tokenizer.cc",
        "src/trace_processor/raw_table.cc",
        "src/trace_processor/row_iterators.cc",
        "src/trace_processor/sched_slice_table.cc",
        "src/trace_processor/slice_table.cc",
        "src/trace_processor/slice_tracker.cc",
        "src/trace_processor/span_join_operator_table.cc",
        "src/trace_processor/sql_stats_table.cc",
        "src/trace_processor/sqlite/query_constraints.cc",
        "src/trace_processor/sqlite/sqlite3_str_split.cc",
        "src/trace_processor/sqlite/sqlite_table.cc",
        "src/trace_processor/stack_profile_callsite_table.cc",
        "src/trace_processor/stack_profile_frame_table.cc",
        "src/trace_processor/stack_profile_mapping_table.cc",
        "src/trace_processor/stack_profile_tracker.cc",
        "src/trace_processor/stats_table.cc",
        "src/trace_processor/storage_columns.cc",
        "src/trace_processor/storage_schema.cc",
        "src/trace_processor/storage_table.cc",
        "src/trace_processor/string_pool.cc",
        "src/trace_processor/syscall_tracker.cc",
        "src/trace_processor/systrace_parser.cc",
        "src/trace_processor/systrace_trace_parser.cc",
        "src/trace_processor/thread_table.cc",
        "src/trace_processor/trace_processor.cc",
        "src/trace_processor/trace_processor_context.cc",
        "src/trace_processor/trace_processor_impl.cc",
        "src/trace_processor/trace_processor_shell.cc",
        "src/trace_processor/trace_sorter.cc",
        "src/trace_processor/trace_storage.cc",
        "src/trace_processor/track_table.cc",
        "src/trace_processor/virtual_destructors.cc",
        "src/trace_processor/virtual_track_tracker.cc",
        "src/trace_processor/window_operator_table.cc",
    ] + glob([
        "src/trace_processor/**/*.h",
    ]) + [
        ":sql_metrics_h",
    ],
    copts = _COPTS,
    visibility = ["//visibility:public"],
    deps = [
        ":base",
        ":common_zero_proto",
        ":config_zero_proto",
        ":metrics_zero_proto",
        ":protozero",
        ":trace_processor_zero_proto",
        ":trace_zero_proto",
        "//third_party/sqlite",
        "@com_google_protobuf//:protobuf",
        "@net_zlib//:z",
    ],
)

proto_library(
    name = "config_combined_proto",
    srcs = ["perfetto/config/perfetto_config.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "config_go_proto",
    importpath = "perfetto/config",
    proto = ":config_combined_proto",
    visibility = ["//visibility:public"],
)

java_proto_library(
    name = "config_java_proto",
    visibility = ["//visibility:public"],
    deps = [":config_combined_proto"],
)

###################### Internal Libraries ######################

cc_library(
    name = "public_headers",
    hdrs = glob(["include/**/*.h"]),
    strip_include_prefix = "include",
)

cc_library(
    name = "base",
    srcs = [
        "src/base/paged_memory.cc",
        "src/base/string_splitter.cc",
        "src/base/string_utils.cc",
        "src/base/string_view.cc",
        "src/base/thread_checker.cc",
        "src/base/time.cc",
        "src/base/uuid.cc",
        "src/base/virtual_destructors.cc",
        "src/base/waitable_event.cc",
    ] + select({
        "@gapid//tools/build:windows": [],
        "@gapid//tools/build:darwin": [],
        "@gapid//tools/build:linux": [
            "src/base/event_fd.cc",
            "src/base/file_utils.cc",
            "src/base/metatrace.cc",
            "src/base/pipe.cc",
            "src/base/temp_file.cc",
            "src/base/unix_socket.cc",
            "src/base/unix_task_runner.cc",
            "src/android_internal/lazy_library_loader.cc",
        ],
        # Android
        "//conditions:default": [
            "src/base/android_task_runner.cc",
            "src/base/pipe.cc",
            "src/base/temp_file.cc",
            "src/base/unix_socket.cc",
            "src/android_internal/lazy_library_loader.cc",
        ],
    }) + glob([
        "src/android_internal/*.h",
    ]),
    copts = _COPTS,
    deps = [
        ":public_headers",
    ],
)

cc_library(
    name = "base_ipc",
    srcs = [
        "src/ipc/buffered_frame_deserializer.cc",
        "src/ipc/client_impl.cc",
        "src/ipc/deferred.cc",
        "src/ipc/host_impl.cc",
        "src/ipc/service_proxy.cc",
        "src/ipc/virtual_destructors.cc",
    ] + glob([
        "src/ipc/**/*.h",
    ]),
    copts = _COPTS,
    deps = [
        ":base",
        ":wire_protocol_cc_proto",
    ],
)

cc_library(
    name = "protozero",
    srcs = [
        "src/protozero/message.cc",
        "src/protozero/message_handle.cc",
        "src/protozero/proto_decoder.cc",
        "src/protozero/scattered_heap_buffer.cc",
        "src/protozero/scattered_stream_null_delegate.cc",
        "src/protozero/scattered_stream_writer.cc",
    ],
    copts = _COPTS,
    deps = [
        ":base",
    ],
)

cc_library(
    name = "tracing_core",
    srcs = [
        "src/tracing/core/chrome_config.cc",
        "src/tracing/core/commit_data_request.cc",
        "src/tracing/core/data_source_config.cc",
        "src/tracing/core/data_source_descriptor.cc",
        "src/tracing/core/id_allocator.cc",
        "src/tracing/core/metatrace_writer.cc",
        "src/tracing/core/null_trace_writer.cc",
        "src/tracing/core/observable_events.cc",
        "src/tracing/core/packet_stream_validator.cc",
        "src/tracing/core/shared_memory_abi.cc",
        "src/tracing/core/shared_memory_arbiter_impl.cc",
        "src/tracing/core/sliced_protobuf_input_stream.cc",
        "src/tracing/core/startup_trace_writer.cc",
        "src/tracing/core/startup_trace_writer_registry.cc",
        "src/tracing/core/test_config.cc",
        "src/tracing/core/trace_buffer.cc",
        "src/tracing/core/trace_config.cc",
        "src/tracing/core/trace_packet.cc",
        "src/tracing/core/trace_stats.cc",
        "src/tracing/core/trace_writer_impl.cc",
        "src/tracing/core/tracing_service_impl.cc",
        "src/tracing/core/tracing_service_state.cc",
        "src/tracing/core/virtual_destructors.cc",
        "src/tracing/trace_writer_base.cc",
    ] + glob([
        "src/tracing/core/**/*.h",
    ]),
    copts = _COPTS,
    deps = [
        ":base",
        ":common_cc_proto",
        ":config_cc_proto",
        ":protozero",
        ":trace_cc_proto",
        ":trace_zero_proto",
    ],
)

cc_library(
    name = "tracing_ipc",
    srcs = [
        "src/tracing/ipc/consumer/consumer_ipc_client_impl.cc",
        "src/tracing/ipc/default_socket.cc",
        "src/tracing/ipc/posix_shared_memory.cc",
        "src/tracing/ipc/producer/producer_ipc_client_impl.cc",
        "src/tracing/ipc/service/consumer_ipc_service.cc",
        "src/tracing/ipc/service/producer_ipc_service.cc",
        "src/tracing/ipc/service/service_ipc_host_impl.cc",
    ] + glob([
        "src/tracing/ipc/**/*.h",
    ]),
    copts = _COPTS,
    deps = [
        ":base",
        ":ipc_cc_ipc",
        ":ipc_cc_proto",
        ":tracing_core",
    ],
)

genrule(
    name = "sql_metrics_h",
    srcs = [
        "src/trace_processor/metrics/trace_metadata.sql",
        "src/trace_processor/metrics/android/android_batt.sql",
        "src/trace_processor/metrics/android/android_cpu.sql",
        "src/trace_processor/metrics/android/android_cpu_agg.sql",
        "src/trace_processor/metrics/android/android_ion.sql",
        "src/trace_processor/metrics/android/android_lmk.sql",
        "src/trace_processor/metrics/android/android_mem.sql",
        "src/trace_processor/metrics/android/android_mem_unagg.sql",
        "src/trace_processor/metrics/android/android_package_list.sql",
        "src/trace_processor/metrics/android/android_powrails.sql",
        "src/trace_processor/metrics/android/android_process_growth.sql",
        "src/trace_processor/metrics/android/android_startup.sql",
        "src/trace_processor/metrics/android/android_startup_cpu.sql",
        "src/trace_processor/metrics/android/android_startup_launches.sql",
        "src/trace_processor/metrics/android/android_task_state.sql",
        "src/trace_processor/metrics/android/heap_profile_callsite_stats.sql",
        "src/trace_processor/metrics/android/mem_stats_priority_breakdown.sql",
        "src/trace_processor/metrics/android/process_mem.sql",
        "src/trace_processor/metrics/android/process_unagg_mem_view.sql",
        "src/trace_processor/metrics/android/span_view_stats.sql",
        "src/trace_processor/metrics/android/upid_span_view.sql",
    ],
    outs = [
        "src/trace_processor/metrics/sql_metrics.h",
    ],
    cmd = "$(location :gen_merged_sql_metrics) --cpp_out=$@ $(SRCS)",
    tools = [
        ":gen_merged_sql_metrics",
    ],
)

###################### Internal Protos ######################

proto_library(
    name = "common_proto",
    srcs = [
        "perfetto/common/android_log_constants.proto",
        "perfetto/common/commit_data_request.proto",
        "perfetto/common/data_source_descriptor.proto",
        "perfetto/common/descriptor.proto",
        "perfetto/common/gpu_counter_descriptor.proto",
        "perfetto/common/observable_events.proto",
        "perfetto/common/sys_stats_counters.proto",
        "perfetto/common/trace_stats.proto",
        "perfetto/common/tracing_service_state.proto",
    ],
)

cc_proto_library(
    name = "common_cc_proto",
    deps = [":common_proto"],
)

cc_protozero_library(
    name = "common_zero_proto",
    copts = _COPTS,
    deps = [":common_proto"],
)

proto_library(
    name = "config_proto",
    srcs = [
        "perfetto/config/android/android_log_config.proto",
        "perfetto/config/android/packages_list_config.proto",
        "perfetto/config/chrome/chrome_config.proto",
        "perfetto/config/data_source_config.proto",
        "perfetto/config/ftrace/ftrace_config.proto",
        "perfetto/config/gpu/gpu_counter_config.proto",
        "perfetto/config/inode_file/inode_file_config.proto",
        "perfetto/config/power/android_power_config.proto",
        "perfetto/config/process_stats/process_stats_config.proto",
        "perfetto/config/profiling/heapprofd_config.proto",
        "perfetto/config/sys_stats/sys_stats_config.proto",
        "perfetto/config/test_config.proto",
        "perfetto/config/trace_config.proto",
    ],
    deps = [
        ":common_proto",
    ],
)

cc_proto_library(
    name = "config_cc_proto",
    deps = [":config_proto"],
)

cc_protozero_library(
    name = "config_zero_proto",
    copts = _COPTS,
    deps = [":config_proto"],
)

proto_library(
    name = "ipc_proto",
    srcs = [
        "perfetto/ipc/consumer_port.proto",
        "perfetto/ipc/producer_port.proto",
    ],
    deps = [
        ":common_proto",
        ":config_proto",
    ],
)

cc_proto_library(
    name = "ipc_cc_proto",
    deps = [":ipc_proto"],
)

cc_ipc_library(
    name = "ipc_cc_ipc",
    cdeps = [
        ":ipc_cc_proto",
        ":base_ipc",
    ],
    copts = _COPTS,
    deps = [":ipc_proto"],
)

proto_library(
    name = "metrics_proto",
    srcs = [
        "perfetto/metrics/android/batt_metric.proto",
        "perfetto/metrics/android/cpu_metric.proto",
        "perfetto/metrics/android/heap_profile_callsite_stats.proto",
        "perfetto/metrics/android/ion_metric.proto",
        "perfetto/metrics/android/lmk_metric.proto",
        "perfetto/metrics/android/mem_metric.proto",
        "perfetto/metrics/android/mem_unagg_metric.proto",
        "perfetto/metrics/android/package_list.proto",
        "perfetto/metrics/android/powrails_metric.proto",
        "perfetto/metrics/android/process_growth.proto",
        "perfetto/metrics/android/startup_metric.proto",
        "perfetto/metrics/metrics.proto",
    ],
)

cc_protozero_library(
    name = "metrics_zero_proto",
    copts = _COPTS,
    deps = [":metrics_proto"],
)

proto_library(
    name = "trace_processor_proto",
    srcs = [
        "perfetto/trace_processor/metrics_impl.proto",
    ],
)

cc_protozero_library(
    name = "trace_processor_zero_proto",
    copts = _COPTS,
    deps = [":trace_processor_proto"],
)

proto_library(
    name = "trace_proto",
    srcs = [
        "perfetto/trace/android/android_log.proto",
        "perfetto/trace/android/graphics_frame_event.proto",
        "perfetto/trace/android/packages_list.proto",
        "perfetto/trace/chrome/chrome_benchmark_metadata.proto",
        "perfetto/trace/chrome/chrome_metadata.proto",
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
        "perfetto/trace/ftrace/gpu.proto",
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
        "perfetto/trace/ftrace/systrace.proto",
        "perfetto/trace/ftrace/task.proto",
        "perfetto/trace/ftrace/test_bundle_wrapper.proto",
        "perfetto/trace/ftrace/vmscan.proto",
        "perfetto/trace/ftrace/workqueue.proto",
        "perfetto/trace/gpu/gpu_counter_event.proto",
        "perfetto/trace/gpu/gpu_render_stage_event.proto",
        "perfetto/trace/interned_data/interned_data.proto",
        "perfetto/trace/perfetto/perfetto_metatrace.proto",
        "perfetto/trace/power/battery_counters.proto",
        "perfetto/trace/power/power_rails.proto",
        "perfetto/trace/profiling/heap_graph.proto",
        "perfetto/trace/profiling/profile_common.proto",
        "perfetto/trace/profiling/profile_packet.proto",
        "perfetto/trace/ps/process_stats.proto",
        "perfetto/trace/ps/process_tree.proto",
        "perfetto/trace/sys_stats/sys_stats.proto",
        "perfetto/trace/system_info.proto",
        "perfetto/trace/test_event.proto",
        "perfetto/trace/trace.proto",
        "perfetto/trace/trace_packet.proto",
        "perfetto/trace/trace_packet_defaults.proto",
        "perfetto/trace/track_event/debug_annotation.proto",
        "perfetto/trace/track_event/log_message.proto",
        "perfetto/trace/track_event/process_descriptor.proto",
        "perfetto/trace/track_event/source_location.proto",
        "perfetto/trace/track_event/task_execution.proto",
        "perfetto/trace/track_event/thread_descriptor.proto",
        "perfetto/trace/track_event/track_event.proto",
        "perfetto/trace/trigger.proto",
        "perfetto/trace/trusted_packet.proto",
    ],
    deps = [
        ":common_proto",
        ":config_proto",
    ],
)

cc_proto_library(
    name = "trace_cc_proto",
    deps = [":trace_proto"],
)

cc_protozero_library(
    name = "trace_zero_proto",
    copts = _COPTS,
    deps = [":trace_proto"],
)

proto_library(
    name = "wire_protocol_proto",
    srcs = [
        "src/ipc/wire_protocol.proto",
    ],
)

cc_proto_library(
    name = "wire_protocol_cc_proto",
    deps = [":wire_protocol_proto"],
)

proto_library(
    name = "perfetto_cmd_proto",
    srcs = [
        "src/perfetto_cmd/perfetto_cmd_state.proto",
    ],
)

cc_proto_library(
    name = "perfetto_cmd_cc_proto",
    deps = [":perfetto_cmd_proto"],
)

###################### Binaries ######################

cc_stripped_binary(
    name = "perfetto_cmd",
    srcs = [
        "src/perfetto_cmd/config.cc",
        "src/perfetto_cmd/main.cc",
        "src/perfetto_cmd/packet_writer.cc",
        "src/perfetto_cmd/pbtxt_to_pb.cc",
        "src/perfetto_cmd/perfetto_cmd.cc",
        "src/perfetto_cmd/rate_limiter.cc",
        "src/perfetto_cmd/trigger_perfetto.cc",
        "src/perfetto_cmd/trigger_producer.cc",
    ] + glob([
        "src/perfetto_cmd/**/*.h",
        "src/android_internal/*.h",
    ]) + select({
        "@gapid//tools/build:linux": [],
        "@gapid//tools/build:windows": [],
        "@gapid//tools/build:darwin": [],
        # Android
        "//conditions:default": [
            "src/perfetto_cmd/perfetto_cmd_android.cc",
        ],
    }),
    copts = _COPTS,
    visibility = ["//visibility:public"],
    deps = [
        ":base",
        ":config_cc_proto",
        ":perfetto_cmd_cc_proto",
        ":protozero",
        ":trace_cc_proto",
        ":tracing_ipc",
        "@com_google_protobuf//:protobuf",
        "@net_zlib//:z",
    ],
)

cc_stripped_binary(
    name = "traced",
    srcs = [
        "src/traced/service/builtin_producer.cc",
        "src/traced/service/main.cc",
        "src/traced/service/service.cc",
    ] + glob([
        "src/traced/service/**/*.h",
    ]),
    copts = _COPTS_BASE,
    linkopts = select({
        "@gapid//tools/build:linux": ["-lrt"],
        "@gapid//tools/build:darwin": [],
        "@gapid//tools/build:windows": [],
    }),  # keep
    visibility = ["//visibility:public"],
    deps = [
        ":base",
        ":perfetto_cmd_cc_proto",
        ":protozero",
        ":trace_cc_proto",
        ":tracing_ipc",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_stripped_binary(
    name = "traced_probes",
    srcs = [
        "src/traced/probes/android_log/android_log_data_source.cc",
        "src/traced/probes/main.cc",
        "src/traced/probes/probes_data_source.cc",
        "src/traced/probes/probes_producer.cc",
        "src/traced/probes/probes.cc",
        "src/traced/probes/ftrace/event_info.cc",
        "src/traced/probes/ftrace/atrace_wrapper.cc",
        "src/traced/probes/ftrace/atrace_hal_wrapper.cc",
        "src/traced/probes/ftrace/ftrace_metadata.cc",
        "src/traced/probes/ftrace/ftrace_config_muxer.cc",
        "src/traced/probes/ftrace/format_parser.cc",
        "src/traced/probes/ftrace/ftrace_stats.cc",
        "src/traced/probes/ftrace/ftrace_config.cc",
        "src/traced/probes/ftrace/proto_translation_table.cc",
        "src/traced/probes/ftrace/ftrace_data_source.cc",
        "src/traced/probes/ftrace/ftrace_config_utils.cc",
        "src/traced/probes/ftrace/ftrace_controller.cc",
        "src/traced/probes/ftrace/ftrace_procfs.cc",
        "src/traced/probes/ftrace/cpu_reader.cc",
        "src/traced/probes/ftrace/event_info_constants.cc",
        "src/traced/probes/ftrace/cpu_stats_parser.cc",
        "src/traced/probes/filesystem/inode_file_data_source.cc",
        "src/traced/probes/filesystem/fs_mount.cc",
        "src/traced/probes/filesystem/lru_inode_cache.cc",
        "src/traced/probes/filesystem/range_tree.cc",
        "src/traced/probes/filesystem/prefix_finder.cc",
        "src/traced/probes/filesystem/file_scanner.cc",
        "src/traced/probes/metatrace/metatrace_data_source.cc",
        "src/traced/probes/packages_list/packages_list_data_source.cc",
        "src/traced/probes/power/android_power_data_source.cc",
        "src/traced/probes/ps/process_stats_data_source.cc",
        "src/traced/probes/sys_stats/sys_stats_data_source.cc",
    ] + glob([
        "src/android_internal/*.h",
        "src/traced/probes/**/*.h",
    ]) + select({
        "@gapid//tools/build:linux": [],
        "@gapid//tools/build:windows": [],
        "@gapid//tools/build:darwin": [],
        # Android
        "//conditions:default": [
            "src/android_internal/atrace_hal.cc",
            "src/android_internal/dropbox_service.cc",
            "src/android_internal/power_stats_hal.cc",
            "src/android_internal/health_hal.cc",
        ],
    }),
    copts = _COPTS_BASE,
    linkopts = select({
        "@gapid//tools/build:linux": [
            "-ldl",
            "-lrt",
        ],
        "@gapid//tools/build:darwin": [],
        "@gapid//tools/build:windows": [],
    }),  # keep
    visibility = ["//visibility:public"],
    deps = [
        ":base",
        ":common_cc_proto",
        ":common_zero_proto",
        ":config_zero_proto",
        ":perfetto_cmd_cc_proto",
        ":protozero",
        ":trace_zero_proto",
        ":tracing_ipc",
        "@com_google_protobuf//:protobuf",
    ],
)

###################### Plugins & Generators ######################

py_binary(
    name = "gen_merged_sql_metrics",
    srcs = ["tools/gen_merged_sql_metrics.py"],
)

cc_binary(
    name = "protozero_plugin",
    srcs = [
        "src/protozero/protoc_plugin/protozero_plugin.cc",
    ],
    copts = _COPTS,
    deps = [
        "@com_google_protobuf//:protoc_lib",
    ],
)

cc_binary(
    name = "ipc_plugin",
    srcs = [
        "src/ipc/protoc_plugin/ipc_plugin.cc",
    ],
    copts = _COPTS,
    deps = [
        "@com_google_protobuf//:protoc_lib",
    ],
)
