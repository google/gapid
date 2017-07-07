load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "replay.go",
        "report.go",
        "scheduler.go",
        "trace.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/log:go_default_library",
        "//test/robot/build:go_default_library",
        "//test/robot/job:go_default_library",
        "//test/robot/monitor:go_default_library",
        "//test/robot/replay:go_default_library",
        "//test/robot/report:go_default_library",
        "//test/robot/trace:go_default_library",
    ],
)
