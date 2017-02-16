load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "delegate.go",
        "doc.go",
        "echo.go",
        "match.go",
        "response.go",
        "sequence.go",
        "stub.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/event/task:go_default_library",
        "//core/os/shell:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["stub_example_test.go"],
    deps = [
        ":go_default_library",
        "//core/event/task:go_default_library",
        "//core/log:go_default_library",
        "//core/os/shell:go_default_library",
    ],
)
