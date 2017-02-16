load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "go_proto_sources", "java_proto_library")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
    deps = [
        "//core/image:pb",
        "//core/log/log_pb:pb",
        "//core/os/device:pb",
        "//gapis/gfxapi:pb",
        "//gapis/service/path:pb",
        "//gapis/service/pod:pb",
        "//gapis/stringtable:pb",
    ],
)

go_proto_sources(
    name = "pb_go",
    grpc = True,
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

java_proto_library(
    name = "pb_java",
    extra_deps = [
        "//core/image:pb_java",
        "//core/log/log_pb:pb_java",
        "//core/os/device:pb_java",
        "//gapis/gfxapi:pb_java",
        "//gapis/service/path:pb_java",
        "//gapis/service/pod:pb_java",
        "//gapis/stringtable:pb_java",
    ],
    grpc = True,
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "errors.go",
        "object.go",
        "report.go",
        "service.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/data/protoutil:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/log/log_pb:go_default_library",
        "//core/os/device:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/any:go_default_library",
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//framework/binary/vle:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/service/path:go_default_library",
        "//gapis/service/pod:go_default_library",
        "//gapis/stringtable:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_x_net//context:go_default_library",
    ],
)
