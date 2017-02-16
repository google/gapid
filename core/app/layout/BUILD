load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "layout.go",
        "pkgdata.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/file:go_default_library",
    ],
)
