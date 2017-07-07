load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["parse.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/parse:go_default_library",
        "//gapis/stringtable/minidown/node:go_default_library",
        "//gapis/stringtable/minidown/scanner:go_default_library",
        "//gapis/stringtable/minidown/token:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["parse_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//gapis/stringtable/minidown/node:go_default_library",
    ],
)
