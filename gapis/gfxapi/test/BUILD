load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")
load("//tools/build:rules.bzl", "api_library", "snippets", "apic")

api_library(
    name = "api",
    api = "gfxapi_test.api",
    apiname = "test",
    includes = glob(["*.api"]),
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
        "doc.go",
        "test.go",
        ":generated",  # keep
        ":snippets",  # keep
        "//tools/build/codergen:gapis/gfxapi/test/test_binary.go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/math/u64:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/atom_pb:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/gfxapi/test/gfxapi_test_import:go_default_library",
        "//gapis/gfxapi/test/test_pb:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/memory/memory_pb:go_default_library",
        "//gapis/replay/builder:go_default_library",
        "//gapis/replay/protocol:go_default_library",
        "//gapis/replay/value:go_default_library",
        "//gapis/service/path:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = [
        "intrinsics_test.go",
        "mutate_test.go",
        "subroutines_test.go",
    ],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//core/data/id:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//framework/binary/any:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/replay/builder:go_default_library",
        "//gapis/replay/opcode:go_default_library",
        "//gapis/replay/protocol:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]) + [":generated"],
    visibility = ["//visibility:public"],
)
