load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "baton.go",
        "context.go",
        "doc.go",
        "event.go",
        "executor.go",
        "factory.go",
        "handle.go",
        "runner.go",
        "signal.go",
        "task.go",
    ],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "baton_test.go",
        "context_test.go",
        "event_test.go",
        "executor_test.go",
        "factory_test.go",
        "runner_test.go",
        "signal_test.go",
        "task_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
    ],
)
