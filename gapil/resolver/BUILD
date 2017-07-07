load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "api.go",
        "docs.go",
        "expression.go",
        "extract_calls.go",
        "flow.go",
        "function.go",
        "generic_subroutine.go",
        "inference.go",
        "internal.go",
        "operator.go",
        "remove_dead_code.go",
        "resolve.go",
        "resolver.go",
        "rules.go",
        "statement.go",
        "type.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/parse:go_default_library",
        "//gapil/ast:go_default_library",
        "//gapil/parser:go_default_library",
        "//gapil/semantic:go_default_library",
        "//gapil/semantic/printer:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["resolve_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//core/text/parse:go_default_library",
        "//gapil/ast:go_default_library",
        "//gapil/parser:go_default_library",
        "//gapil/semantic:go_default_library",
        "//gapil/semantic/printer:go_default_library",
    ],
)
