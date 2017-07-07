load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "client.go",
        "doc.go",
        "host_log_parser.go",
        "session.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app:go_default_library",
        "//core/app/auth:go_default_library",
        "//core/data/endian:go_default_library",
        "//core/event/task:go_default_library",
        "//core/log:go_default_library",
        "//core/os/android/adb:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/device/bind:go_default_library",
        "//core/os/file:go_default_library",
        "//core/os/process:go_default_library",
        "//core/os/shell:go_default_library",
        "//core/text:go_default_library",
        "//gapidapk:go_default_library",
        "//gapis/replay/protocol:go_default_library",
    ],
)
