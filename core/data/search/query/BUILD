load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "builder.go",
        "doc.go",
        "expression.go",
        "replace.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/search:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
