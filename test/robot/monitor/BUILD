load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "build.go",
        "doc.go",
        "generation.go",
        "job.go",
        "monitor.go",
        "replay.go",
        "report.go",
        "subject.go",
        "trace.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/search:go_default_library",
        "//core/data/stash:go_default_library",
        "//core/os/android/apk:go_default_library",
        "//test/robot/build:go_default_library",
        "//test/robot/job:go_default_library",
        "//test/robot/job/worker:go_default_library",
        "//test/robot/master:go_default_library",
        "//test/robot/replay:go_default_library",
        "//test/robot/report:go_default_library",
        "//test/robot/subject:go_default_library",
        "//test/robot/trace:go_default_library",
    ],
)
