load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
        "summarise.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//core/data/compare:go_default_library",
        "//core/log:go_default_library",
        "//core/os/file:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/atom_pb:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/gfxapi/all:go_default_library",
    ],
)

go_binary(
    name = "gfxtrace",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
