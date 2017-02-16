load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "encoder.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/log:go_default_library",
        "//core/os/shell:go_default_library",
    ],
)
