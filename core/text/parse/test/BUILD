load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "ast.go",
        "cst.go",
        "verify.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//core/text/parse:go_default_library",
    ],
)
