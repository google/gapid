load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")
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
        "api.go",
        "context.go",
        "doc.go",
        "mesh.go",
        "resource.go",
        "snippet.go",
        "state.go",
        "texture.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/math/f32:go_default_library",
        "//core/os/device:go_default_library",
        "//core/stream:go_default_library",
        "//core/stream/fmts:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/vle:go_default_library",
        "//gapil/snippets:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/replay/value:go_default_library",
        "//gapis/service/path:go_default_library",
        "//gapis/service/pod:go_default_library",
        "//gapis/stringtable:go_default_library",
        "//gapis/vertex:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["texture_test.go"],
    deps = [
        ":go_default_library",
        "//core/image:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
)
