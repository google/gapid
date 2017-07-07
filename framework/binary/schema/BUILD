load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "any.go",
        "array.go",
        "class.go",
        "coder.go",
        "constant.go",
        "interface.go",
        "map.go",
        "message.go",
        "object.go",
        "pointer.go",
        "primitive.go",
        "schema.go",
        "struct.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["schema_test.go"],
    deps = [
        ":go_default_library",
        "//framework/binary:go_default_library",
    ],
)
