load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "go_proto_sources")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
    deps = ["//core/os/device:pb"],
)

go_proto_sources(
    name = "pb_go",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "analysis.go",
        "apk.go",
        "debugifier.go",
        "doc.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/os/android/binaryxml:go_default_library",
        "//core/os/android/manifest:go_default_library",
        "//core/os/device:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
