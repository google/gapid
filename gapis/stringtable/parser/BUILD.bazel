load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["parser.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//gapis/stringtable:go_default_library",
        "//gapis/stringtable/minidown:go_default_library",
        "//gapis/stringtable/minidown/node:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["parser_test.go"],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//gapis/stringtable:go_default_library",
    ],
)
