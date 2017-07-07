load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["executor.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/replay/builder:go_default_library",
        "//gapis/replay/protocol:go_default_library",
    ],
)
