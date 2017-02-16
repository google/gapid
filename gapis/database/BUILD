load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "database.go",
        "hash.go",
        "memory.go",
        "resolvable.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/context/keys:go_default_library",
        "//core/data/id:go_default_library",
        "//core/event/task:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/vle:go_default_library",
        "//gapis/config:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
