load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["printer.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/reflow:go_default_library",
        "//gapil/semantic:go_default_library",
    ],
)
