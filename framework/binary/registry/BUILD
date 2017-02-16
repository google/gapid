load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "registry.go",
        "schema.go",
    ],
    visibility = ["//visibility:public"],
    deps = ["//framework/binary:go_default_library"],
)
