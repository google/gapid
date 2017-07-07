load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "go_proto_sources", "java_proto_library")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
    deps = [
        "//:ptypes",
        "//gapis/service/pod:pb",
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
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "pb.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/log:go_default_library",
        "//gapis/service/pod:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@com_github_golang_protobuf//ptypes/timestamp:go_default_library",
    ],
)
