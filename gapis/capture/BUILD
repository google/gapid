load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "go_proto_sources")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
    deps = [
        "//core/os/device:pb",
    ],
)

go_proto_sources(
    name = "pb_go",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "capture.go",
        "context.go",
        "doc.go",
        "id.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/context/keys:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pack:go_default_library",
        "//core/log:go_default_library",
        "//core/math/interval:go_default_library",
        "//core/os/device:go_default_library",
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//framework/binary/vle:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/atom_pb:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/replay/value:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/service/path:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ],
)
