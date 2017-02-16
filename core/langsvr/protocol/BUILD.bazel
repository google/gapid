load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "connection.go",
        "doc.go",
        "message.go",
        "notifications.go",
        "requests.go",
        "types.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/event/task:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
    ],
)
