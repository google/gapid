load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "cpp.go",
        "generate.go",
        "go.go",
        "java.go",
        "signatures.go",
        "struct.go",
        "tags.go",
        "types.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/copyright:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//framework/binary/vle:go_default_library",
        "//tools/codergen/scan:go_default_library",
        "@org_golang_x_tools//go/types:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["generate_test.go"],
    library = ":go_default_library",
    tags = ["manual"],
    deps = [
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//tools/codergen/scan:go_default_library",
    ],
)
