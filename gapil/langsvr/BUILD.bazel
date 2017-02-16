load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "analyze.go",
        "debug_logger.go",
        "main.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//core/event/task:go_default_library",
        "//core/langsvr:go_default_library",
        "//core/log:go_default_library",
        "//core/os/file:go_default_library",
        "//core/text/parse:go_default_library",
        "//gapil:go_default_library",
        "//gapil/analysis:go_default_library",
        "//gapil/ast:go_default_library",
        "//gapil/format:go_default_library",
        "//gapil/parser:go_default_library",
        "//gapil/resolver:go_default_library",
        "//gapil/semantic:go_default_library",
        "//gapil/semantic/printer:go_default_library",
        "//gapil/validate:go_default_library",
    ],
)

go_binary(
    name = "langsvr",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
