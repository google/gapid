load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "api_library", "snippets", "apic")

api_library(
    name = "api",
    api = "vulkan.api",
    apiname = "vulkan",
    includes = native.glob([
        "*.api",
        "linux/*.api",
        "windows/*.api",
        "android/*.api",
    ]),
    visibility = ["//visibility:public"],
    deps = ["//gapis/messages:api"],
)

snippets(
    name = "snippets",
    api = ":api",
    visibility = ["//visibility:public"],
)

apic(
    name = "generated",
    api = ":api",
    templates = [
        "//gapis/gfxapi/templates:api",
        "//gapis/gfxapi/templates:mutate",
        "//gapis/gfxapi/templates:convert",
        "//gapis/gfxapi/templates:proto",
    ],
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "buffer_command.go",
        "custom_replay.go",
        "doc.go",
        "externs.go",
        "find_issues.go",
        "read_framebuffer.go",
        "replay.go",
        "resources.go",
        "state.go",
        "vulkan.go",
        "vulkan_binary_metatadata.go",
        ":generated",  # keep
        ":snippets",  # keep
        "//tools/build/codergen:gapis/gfxapi/vulkan/vulkan_binary.go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/event/task:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/math/u64:go_default_library",
        "//core/os/device:go_default_library",
        "//core/stream/fmts:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/atom_pb:go_default_library",
        "//gapis/atom/transform:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/config:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/gfxapi/vulkan/vulkan_pb:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/memory/memory_pb:go_default_library",
        "//gapis/messages:go_default_library",
        "//gapis/replay:go_default_library",
        "//gapis/replay/builder:go_default_library",
        "//gapis/replay/protocol:go_default_library",
        "//gapis/replay/value:go_default_library",
        "//gapis/resolve:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/service/path:go_default_library",
        "//gapis/shadertools:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]) + [":generated"],
    visibility = ["//visibility:public"],
)
