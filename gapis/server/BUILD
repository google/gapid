load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "grpc.go",
        "server.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app/auth:go_default_library",
        "//core/app/benchmark:go_default_library",
        "//core/context/keys:go_default_library",
        "//core/event/task:go_default_library",
        "//core/log:go_default_library",
        "//core/log/log_pb:go_default_library",
        "//core/net/grpcutil:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/device/bind:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/gfxapi/all:go_default_library",
        "//gapis/replay:go_default_library",
        "//gapis/resolve:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/service/path:go_default_library",
        "//gapis/stringtable:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_x_net//context:go_default_library",
    ],
)
