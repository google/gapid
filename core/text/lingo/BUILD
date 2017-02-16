load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "error.go",
        "node.go",
        "scanner.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
    ],
)
