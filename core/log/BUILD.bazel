load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "broadcast.go",
        "channel.go",
        "clock.go",
        "err.go",
        "filter.go",
        "handler.go",
        "log.go",
        "message.go",
        "process.go",
        "severity.go",
        "stacktracer.go",
        "style.go",
        "styles.go",
        "tag.go",
        "testing.go",
        "trace.go",
        "values.go",
        "writer.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app/flags:go_default_library",
        "//core/context/keys:go_default_library",
        "//core/text:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "broadcast_test.go",
        "channel_test.go",
        "log_test.go",
        "styles_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
    ],
)
