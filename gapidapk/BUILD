load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "deviceinfo.go",
        "doc.go",
        "gapidapk.go",
        "packageinfo.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app/layout:go_default_library",
        "//core/log:go_default_library",
        "//core/os/android:go_default_library",
        "//core/os/android/adb:go_default_library",
        "//core/os/android/apk:go_default_library",
        "//core/os/device:go_default_library",
        "//gapidapk/pkginfo:go_default_library",
        "@com_github_golang_protobuf//jsonpb:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
