load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["parse.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/parse:go_default_library",
        "//gapis/stringtable/minidown/node:go_default_library",
        "//gapis/stringtable/minidown/parser:go_default_library",
    ],
)
