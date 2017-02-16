load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "bitstream.go",
        "box.go",
        "control.go",
        "decoder.go",
        "doc.go",
        "encoder.go",
        "entity.go",
        "object.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/pod:go_default_library",
        "//core/math/u32:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["bitstream_test.go"],
    library = ":go_default_library",
    deps = ["//core/assert:go_default_library"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "object_example_test.go",
        "roundtrip_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/data/id:go_default_library",
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//framework/binary/test:go_default_library",
        "//framework/binary/vle:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
)
