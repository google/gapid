load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "generator.go",
        "introspect.go",
        "rewrite.go",
        "wrap.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/os/file:go_default_library",
    ],
)
