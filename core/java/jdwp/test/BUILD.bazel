load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["test.go"],
    tags = ["integration"],
    visibility = ["//visibility:public"],
    deps = [
        "//core/event/task:go_default_library",
        "//core/java/jdwp:go_default_library",
        "//core/log:go_default_library",
        "//core/os/shell:go_default_library",
    ],
)
