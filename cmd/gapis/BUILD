load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
        "version.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//core/app/auth:go_default_library",
        "//core/event/task:go_default_library",
        "//core/log:go_default_library",
        "//core/os/android/adb:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/device/bind:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/replay:go_default_library",
        "//gapis/server:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/stringtable:go_default_library",
    ],
)

go_binary(
    name = "gapis",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
