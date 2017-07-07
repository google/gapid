load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "lingo")

lingo(
    name = "lingo",
    srcs = glob(["*.lingo"]),
)

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        ":lingo",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/search/query:go_default_library",
        "//core/log:go_default_library",
        "//core/text/lingo:go_default_library",
    ],
)
