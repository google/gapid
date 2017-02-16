load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "inspect.go",
        "issues.go",
        "no_unused.go",
        "validate.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/parse:go_default_library",
        "//gapil/analysis:go_default_library",
        "//gapil/resolver:go_default_library",
        "//gapil/semantic:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["inspect_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//gapil:go_default_library",
        "//gapil/ast:go_default_library",
        "//gapil/parser:go_default_library",
        "//gapil/resolver:go_default_library",
        "//gapil/semantic:go_default_library",
    ],
)
