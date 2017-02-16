load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "pointer_resolver.go",
        "value.go",
        "values.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/math/interval:go_default_library",
        "//gapis/replay/protocol:go_default_library",
    ],
)
