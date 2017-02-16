load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["api.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//core/os/file:go_default_library",
        "//core/text/parse:go_default_library",
        "//gapil/ast:go_default_library",
        "//gapil/parser:go_default_library",
        "//gapil/resolver:go_default_library",
        "//gapil/semantic:go_default_library",
    ],
)
