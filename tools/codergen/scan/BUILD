load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "load.go",
        "process.go",
        "scanner.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/text/copyright:go_default_library",
        "@org_golang_x_tools//go/types:go_default_library",
    ],
)
