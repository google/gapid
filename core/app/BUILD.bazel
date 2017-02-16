load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "atexit.go",
        "doc.go",
        "flags.go",
        "help.go",
        "log.go",
        "profile.go",
        "run.go",
        "verbs.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app/flags:go_default_library",
        "//core/event/task:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/os/file:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ],
)
