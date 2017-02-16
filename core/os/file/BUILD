load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "file.go",
        "list.go",
        "path.go",
        "rooted.go",
        "set.go",
        "unix.go",
    ],
    visibility = ["//visibility:public"],
    deps = ["//core/log:go_default_library"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["path_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
    ],
)
