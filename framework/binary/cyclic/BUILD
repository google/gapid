load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "control.go",
        "cyclic.go",
        "doc.go",
        "substack.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/pod:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["cyclic_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/test:go_default_library",
        "//framework/binary/vle:go_default_library",
    ],
)
