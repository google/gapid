load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "apic", "go_proto_sources")

apic(
    name = "api_proto",
    api = "//gapis/gfxapi/core:api",
    templates = ["//gapis/gfxapi/templates:proto"],
    visibility = ["//visibility:public"],
)

proto_library(
    name = "pb",
    srcs = [":api_proto"],
    tags = ["manual"],  # proto descriptor generation fails on generated files...
    visibility = ["//visibility:public"],
    deps = ["//gapis/memory/memory_pb:pb"],
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
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//gapis/memory/memory_pb:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
