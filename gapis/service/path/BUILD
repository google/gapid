load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "go_proto_sources", "java_proto_library")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
    deps = [
        "//core/image:pb",
        "//gapis/service/pod:pb",
        "//gapis/vertex:pb",
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
        "//gapis/service/pod:pb_java",
        "//gapis/vertex:pb_java",
    ],
    grpc = True,
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "id.go",
        "linker.go",
        "path.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/data/protoutil:go_default_library",
        "//core/image:go_default_library",
        "//gapis/service/pod:go_default_library",
        "//gapis/vertex:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
