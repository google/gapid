load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "collect.go",
        "doc.go",
        "types.go",
    ],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["fault_test.go"],
    deps = [":go_default_library"],
)
