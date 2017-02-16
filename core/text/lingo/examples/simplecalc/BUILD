load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")
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
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//core/log:go_default_library",
        "//core/text/lingo:go_default_library",
    ],
)

go_binary(
    name = "simplecalc",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
