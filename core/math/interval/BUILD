load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "algorithm.go",
        "doc.go",
        "interval_binary_locked.go",
        "list.go",
        "u64.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["u64_test.go"],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
    ],
)
