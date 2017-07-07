load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "ast.go",
        "errors.go",
        "operators.go",
        "types.go",
        "value.go",
        "visitor.go",
    ],
    visibility = ["//visibility:public"],
    deps = ["//core/text/parse:go_default_library"],
)
