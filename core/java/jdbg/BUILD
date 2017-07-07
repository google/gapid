load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "assignable.go",
        "errors.go",
        "failure.go",
        "jdbg.go",
        "marshal.go",
        "method_signature.go",
        "methods.go",
        "resolve_overload.go",
        "signature.go",
        "type.go",
        "value.go",
    ],
    visibility = ["//visibility:public"],
    deps = ["//core/java/jdwp:go_default_library"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["jdbg_test.go"],
    tags = ["integration"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/java/jdwp:go_default_library",
        "//core/java/jdwp/test:go_default_library",
        "//core/log:go_default_library",
    ],
)
