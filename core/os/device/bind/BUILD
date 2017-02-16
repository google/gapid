load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "go_proto_sources")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
)

go_proto_sources(
    name = "pb_go",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "bind.go",
        "device.go",
        "doc.go",
        "registry.go",
        "simple.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/context/keys:go_default_library",
        "//core/data/id:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/shell:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
