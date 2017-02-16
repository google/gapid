load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "adb.go",
        "capture.go",
        "doc.go",
        "header.go",
        "jdwp_loader.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/event/task:go_default_library",
        "//core/java/jdbg:go_default_library",
        "//core/java/jdwp:go_default_library",
        "//core/log:go_default_library",
        "//core/os/android:go_default_library",
        "//core/os/android/adb:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/file:go_default_library",
        "//gapidapk:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ],
)
