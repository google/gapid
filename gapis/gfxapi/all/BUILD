load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["all.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/gfxapi/core:go_default_library",
        "//gapis/gfxapi/gles:go_default_library",
        "//gapis/gfxapi/vulkan:go_default_library",
    ],
)
