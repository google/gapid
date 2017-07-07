load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "go_proto_sources")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/search:pb",
        "//core/os/device:pb",
    ],
)

go_proto_sources(
    name = "pb_go",
    grpc = True,
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "device.go",
        "doc.go",
        "job.go",
        "local.go",
        "manager.go",
        "remote.go",
        "server.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/record:go_default_library",
        "//core/data/search:go_default_library",
        "//core/data/search/eval:go_default_library",
        "//core/event:go_default_library",
        "//core/net/grpcutil:go_default_library",
        "//core/os/device:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_x_net//context:go_default_library",
    ],
)
