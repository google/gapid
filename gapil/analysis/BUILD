load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "analysis.go",
        "analyze.go",
        "bool_value.go",
        "class_value.go",
        "enum_value.go",
        "expressions.go",
        "map_value.go",
        "possibility.go",
        "reference_value.go",
        "results.go",
        "scope.go",
        "uint_value.go",
        "unreachables.go",
        "untracked_value.go",
        "value.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/math/interval:go_default_library",
        "//core/math/u64:go_default_library",
        "//core/text/parse:go_default_library",
        "//gapil/ast:go_default_library",
        "//gapil/resolver:go_default_library",
        "//gapil/semantic:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "analyze_test.go",
        "class_value_test.go",
        "map_value_test.go",
        "possibility_test.go",
        "reference_value_test.go",
        "uint_value_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//core/math/interval:go_default_library",
        "//gapil:go_default_library",
        "//gapil/ast:go_default_library",
        "//gapil/parser:go_default_library",
        "//gapil/resolver:go_default_library",
        "//gapil/semantic:go_default_library",
    ],
)
