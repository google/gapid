load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "format.go",
        "indenter.go",
        "ws_trimmer.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/parse:go_default_library",
        "//gapil/ast:go_default_library",
        "//gapil/parser:go_default_library",
    ],
)
