load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "embed")

embed(
    name = "embed",
    srcs = glob(["*.tmpl"]),
)

go_library(
    name = "go_default_library",
    srcs = [
        "functions.go",
        "section.go",
        "template.go",
        ":embed",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/text/reflow:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//tools/codergen/generate:go_default_library",
    ],
)
