load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["doc.go"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["postback_test.go"],
    library = ":go_default_library",
    tags = [
        "fail",
        "integration",
        "manual",
    ],
    deps = [
        "//core/app:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/event/task:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device/bind:go_default_library",
        "//framework/binary/any:go_default_library",
        "//gapir/client:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/replay/builder:go_default_library",
        "//gapis/replay/executor:go_default_library",
        "//gapis/replay/value:go_default_library",
    ],
)
