load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")
load("//tools/build:rules.bzl", "go_proto_sources")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
    deps = [
        "//gapis/gfxapi:pb",
        "//gapis/service:pb",
        "//gapis/service/path:pb",
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
        "as.go",
        "contexts.go",
        "doc.go",
        "follow.go",
        "framebuffer_attachment.go",
        "framebuffer_attachment_data.go",
        "framebuffer_changes.go",
        "get.go",
        "hierarchies.go",
        "index_limits.go",
        "memory.go",
        "mesh.go",
        "report.go",
        "resolve.go",
        "resource_data.go",
        "resource_meta.go",
        "resources.go",
        "set.go",
        "state.go",
        "thumbnail.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/fault:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/math/interval:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/device/bind:go_default_library",
        "//framework/binary:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/messages:go_default_library",
        "//gapis/replay:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/service/path:go_default_library",
        "//gapis/stringtable:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = [
        "get_set_test.go",
        "requests_test.go",
        "//tools/build/codergen:gapis/resolve/resolve_binary_test.go",  # keep
    ],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/any:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/messages:go_default_library",
        "//gapis/replay/builder:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/service/path:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
)
