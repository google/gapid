load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "client.go",
        "doc.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app/auth:go_default_library",
        "//core/os/shell:go_default_library",
    ],
)
