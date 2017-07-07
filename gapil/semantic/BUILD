load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "api.go",
        "expression.go",
        "function.go",
        "internal.go",
        "invalid.go",
        "logical_order.go",
        "node.go",
        "statement.go",
        "symbols.go",
        "type.go",
        "visit.go",
    ],
    visibility = ["//visibility:public"],
    deps = ["//gapil/ast:go_default_library"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "interface_test.go",
        "semantic_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
    ],
)
