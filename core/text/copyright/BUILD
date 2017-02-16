load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/build:rules.bzl", "embed")

embed(
    name = "embed",
    srcs = glob(
        ["*"],
        exclude = [
            "*.go",
            "BUILD",
            "CMake*",
        ],
    ),
)

go_library(
    name = "go_default_library",
    srcs = [
        "copyright.go",
        ":embed",  # keep
    ],
    visibility = ["//visibility:public"],
)
