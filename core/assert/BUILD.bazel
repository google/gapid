load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "assertion.go",
        "deprecated.go",
        "enum.go",
        "error.go",
        "float.go",
        "integer.go",
        "manager.go",
        "slice.go",
        "string.go",
        "time.go",
        "value.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/compare:go_default_library",
        "//core/log:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "enum_test.go",
        "error_test.go",
        "float_test.go",
        "integer_test.go",
        "manager_test.go",
        "slice_test.go",
        "string_test.go",
        "time_test.go",
        "value_test.go",
    ],
    deps = [":go_default_library"],
)
