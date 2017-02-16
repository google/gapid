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
        "file.go",
        "json.go",
        "ledger.go",
        "library.go",
        "null.go",
        "pb.go",
        "pbtxt.go",
        "record.go",
        "shelf.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/event:go_default_library",
        "//core/log:go_default_library",
        "//core/os/file:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
