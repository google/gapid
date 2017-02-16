load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//gapis/shadertools:go_default_library",
    ],
)

go_binary(
    name = "shadertool",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
