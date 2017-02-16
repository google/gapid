load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "builder.go",
        "doc.go",
        "expression_builder.go",
        "format.go",
        "shader_symbols.go",
        "verbatim.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/text/parse:go_default_library",
        "//gapis/gfxapi/gles/glsl/ast:go_default_library",
        "//gapis/gfxapi/gles/glsl/builtins:go_default_library",
        "//gapis/gfxapi/gles/glsl/preprocessor:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = [
        "builder_test.go",
        "formatter_test.go",
    ],
    library = ":go_default_library",
    deps = [
        "//gapis/gfxapi/gles/glsl/ast:go_default_library",
        "//gapis/gfxapi/gles/glsl/preprocessor:go_default_library",
    ],
)
