load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "api.go",
        "expression.go",
        "function.go",
        "identifier.go",
        "operator.go",
        "parser.go",
        "statement.go",
        "type.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/parse:go_default_library",
        "//gapil/ast:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["parser_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/text/parse:go_default_library",
        "//core/text/parse/test:go_default_library",
    ],
)
