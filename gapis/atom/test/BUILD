load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "atoms.go",
        "common.go",
        "doc.go",
        "//tools/build/codergen:gapis/atom/test/test_binary.go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
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
