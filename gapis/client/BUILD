load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "client.go",
        "doc.go",
        "process.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app/auth:go_default_library",
        "//core/event:go_default_library",
        "//core/log:go_default_library",
        "//core/log/log_pb:go_default_library",
        "//core/net/grpcutil:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/file:go_default_library",
        "//core/os/process:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/service/path:go_default_library",
        "//gapis/stringtable:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
    ],
)
