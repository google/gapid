load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")
load("//tools/build:rules.bzl", "embed")

embed(
    name = "embed",
    srcs = glob(["*.tmpl"]),
    package = "main",
)

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
        "template.go",
        ":embed",  # keep
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/text/copyright:go_default_library",
        "//core/text/reflow:go_default_library",
        "//gapis/stringtable:go_default_library",
        "//gapis/stringtable/parser:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_binary(
    name = "stringgen",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
