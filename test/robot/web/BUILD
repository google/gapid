load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "embed")

embed(
    name = "embed",
    srcs = glob([
        "www/*.html",
        "www/*.css",
        "www/*.js",
        "www/*.png",
        "template/*.tmpl",
    ]),
)

go_library(
    name = "go_default_library",
    srcs = [
        "action.go",
        "build.go",
        "doc.go",
        "entity.go",
        "job.go",
        "master.go",
        "static.go",
        "status.go",
        "subject.go",
        "web.go",
        ":embed",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/search/query:go_default_library",
        "//core/data/stash:go_default_library",
        "//core/log:go_default_library",
        "//core/os/file:go_default_library",
        "//test/robot/build:go_default_library",
        "//test/robot/job:go_default_library",
        "//test/robot/master:go_default_library",
        "//test/robot/monitor:go_default_library",
        "//test/robot/replay:go_default_library",
        "//test/robot/report:go_default_library",
        "//test/robot/subject:go_default_library",
        "//test/robot/trace:go_default_library",
    ],
)
