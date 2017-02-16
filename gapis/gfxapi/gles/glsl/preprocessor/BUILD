load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "constants.go",
        "lexer.go",
        "macroExpander.go",
        "preprocessor.go",
        "preprocessorImpl.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/parse:go_default_library",
        "//gapis/gfxapi/gles/glsl/ast:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["preprocessor_test.go"],
    library = ":go_default_library",
    deps = ["//gapis/gfxapi/gles/glsl/ast:go_default_library"],
)
