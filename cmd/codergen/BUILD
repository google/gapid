load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//core/log:go_default_library",
        "//core/text/copyright:go_default_library",
        "//tools/codergen/generate:go_default_library",
        "//tools/codergen/scan:go_default_library",
        "//tools/codergen/template:go_default_library",
    ],
)

go_binary(
    name = "codergen",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
