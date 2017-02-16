load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "branch.go",
        "doc.go",
        "error.go",
        "fragment.go",
        "leaf.go",
        "node.go",
        "parser.go",
        "reader.go",
        "skip.go",
        "source.go",
        "token.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/compare:go_default_library",
        "//core/fault:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["reader_test.go"],
    library = ":go_default_library",
    deps = ["//core/assert:go_default_library"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["parser_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/text/parse/test:go_default_library",
    ],
)
