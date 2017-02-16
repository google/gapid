load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")
load("//tools/build:rules.bzl", "go_proto_sources", "java_proto_library")

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

java_proto_library(
    name = "pb_java",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "channel.go",
        "component.go",
        "convert.go",
        "datatype.go",
        "doc.go",
        "format.go",
        "sampling.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/protoutil:go_default_library",
        "//core/math/f16:go_default_library",
        "//core/math/f32:go_default_library",
        "//core/math/f64:go_default_library",
        "//framework/binary:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["convert_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/stream/fmts:go_default_library",
    ],
)
