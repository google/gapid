load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "choices.go",
        "doc.go",
        "flags.go",
        "strings.go",
    ],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["choices_test.go"],
    deps = [
        ":go_default_library",
        "//core/app:go_default_library",
        "//core/assert:go_default_library",
    ],
)
