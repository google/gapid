load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "format.go",
        "main.go",
        "template.go",
        "validate.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//core/app/flags:go_default_library",
        "//core/log:go_default_library",
        "//core/os/file:go_default_library",
        "//core/text/parse:go_default_library",
        "//gapil:go_default_library",
        "//gapil/format:go_default_library",
        "//gapil/parser:go_default_library",
        "//gapil/template:go_default_library",
        "//gapil/validate:go_default_library",
    ],
)

go_binary(
    name = "apic",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
