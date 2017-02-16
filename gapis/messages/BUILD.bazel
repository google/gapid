load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "stringgen", "api_library")

stringgen(
    name = "messages",
    input = "en-us.stb.md",
    visibility = ["//visibility:public"],
)

api_library(
    name = "api",
    api = "messages.api",
    apiname = "messages",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        ":messages",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = ["//gapis/stringtable:go_default_library"],
)
