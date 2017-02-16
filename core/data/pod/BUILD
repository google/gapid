load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "pod.go",
        "reader.go",
        "writer.go",
    ],
    visibility = ["//visibility:public"],
    deps = ["//core/math/f16:go_default_library"],
)
