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
        "allocator.go",
        "blob.go",
        "doc.go",
        "id.go",
        "pointer.go",
        "pool.go",
        "pool_write.go",
        "range.go",
        "range_list.go",
        "resource.go",
        "slice.go",
        "slice_info.go",
        "subslice.go",
        "write.go",
        "writer.go",
        ":pb_go",  # keep
        "//tools/build/codergen:gapis/memory/memory_binary.go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/fault:go_default_library",
        "//core/math/interval:go_default_library",
        "//core/math/u64:go_default_library",
        "//core/os/device:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//gapis/atom/atom_pb:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/memory/memory_pb:go_default_library",
        "//gapis/service/path:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = [
        "allocator_test.go",
        "pool_test.go",
        "write_test.go",
    ],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/log:go_default_library",
        "//core/math/interval:go_default_library",
        "//core/os/device:go_default_library",
        "//framework/binary/any:go_default_library",
        "//gapis/database:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
)
