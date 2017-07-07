load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "abort.go",
        "atom.go",
        "cast.go",
        "command_counter.go",
        "convert.go",
        "data.go",
        "doc.go",
        "extras.go",
        "field_alignments.go",
        "flags.go",
        "framebuffer_observation.go",
        "group.go",
        "group_list.go",
        "id_set.go",
        "labeled.go",
        "list.go",
        "observations.go",
        "range.go",
        "range_list.go",
        "resource.go",
        "schema.go",
        "snippet.go",
        "writer.go",
        "//tools/build/codergen:gapis/atom/atom_binary.go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/math/interval:go_default_library",
        "//core/os/device:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//framework/binary/vle:go_default_library",
        "//gapil/snippets:go_default_library",
        "//gapis/atom/atom_pb:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/memory/memory_pb:go_default_library",
        "//gapis/replay/builder:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["group_test.go"],
    library = ":go_default_library",
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["list_test.go"],
    deps = [
        ":go_default_library",
        "//core/log:go_default_library",
        "//gapis/atom/test:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
)
