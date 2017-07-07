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
        "doc.go",
        "opcode.go",
        "payload.go",
        "type.go",
        ":pb_go",  # keep
        "//tools/build/codergen:gapis/replay/protocol/protocol_binary.go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
)
