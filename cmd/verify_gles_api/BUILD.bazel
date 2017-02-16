load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
        "registry.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//core/text/parse:go_default_library",
        "//gapil:go_default_library",
        "//gapil/resolver:go_default_library",
        "//gapil/semantic:go_default_library",
    ],
)

go_binary(
    name = "verify_gles_api",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
