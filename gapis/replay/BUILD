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
        "batcher.go",
        "context.go",
        "custom.go",
        "doc.go",
        "events.go",
        "interfaces.go",
        "manager.go",
        "replay.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app/benchmark:go_default_library",
        "//core/context/keys:go_default_library",
        "//core/data/id:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/device/bind:go_default_library",
        "//framework/binary:go_default_library",
        "//gapir/client:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/transform:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/config:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/replay/builder:go_default_library",
        "//gapis/replay/executor:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/service/path:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
