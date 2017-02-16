load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "command.go",
        "doc.go",
        "env.go",
        "local.go",
        "process.go",
        "target.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/event/task:go_default_library",
        "//core/log:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "command_example_test.go",
        "command_test.go",
        "env_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/event/task:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
    ],
)
