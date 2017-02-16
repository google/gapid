load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "chain.go",
        "doc.go",
        "keys.go",
    ],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["keys_test.go"],
    deps = [":go_default_library"],
)
