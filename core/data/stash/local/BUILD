load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "entity.go",
        "file.go",
        "memory.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/search:go_default_library",
        "//core/data/search/eval:go_default_library",
        "//core/data/stash:go_default_library",
        "//core/event:go_default_library",
        "//core/log:go_default_library",
        "//core/os/file:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@com_github_golang_protobuf//ptypes:go_default_library",
    ],
)
