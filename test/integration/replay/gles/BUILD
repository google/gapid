load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")
load("//tools/build:rules.bzl", "embed")

embed(
    name = "embed",
    srcs = glob([
        "reference/*.png",
    ]),
)

go_library(
    name = "go_default_library",
    srcs = [
        "direct_call.go",
        "doc.go",
        "image.go",
        ":embed",  # keep
        "//tools/build/codergen:test/integration/replay/gles/gles_binary.go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//core/stream:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/replay/builder:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = [
        "direct_call_test.go",
        "gles_test.go",
    ],
    library = ":go_default_library",
    tags = [
        "fail",
        "integration",
        "manual",
    ],
    deps = [
        "//core/app:go_default_library",
        "//core/assert:go_default_library",
        "//core/data/id:go_default_library",
        "//core/event/task:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/device/bind:go_default_library",
        "//framework/binary/any:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/gfxapi/core:go_default_library",
        "//gapis/gfxapi/gles:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/replay:go_default_library",
        "//gapis/resolve:go_default_library",
        "//gapis/service/path:go_default_library",
        "//test/integration/replay/gles/samples:go_default_library",
    ],
)
