load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "api.go",
        "format.go",
        "functions.go",
        "globals.go",
        "list.go",
        "literals.go",
        "names.go",
        "strings.go",
        "template.go",
        "types.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/os/shell:go_default_library",
        "//core/text/copyright:go_default_library",
        "//core/text/reflow:go_default_library",
        "//gapil/resolver:go_default_library",
        "//gapil/semantic:go_default_library",
        "@org_golang_x_tools//imports:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = [
        "literals_test.go",
        "strings_examples_test.go",
        "strings_test.go",
    ],
    library = ":go_default_library",
)
