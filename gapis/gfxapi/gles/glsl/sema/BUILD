load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["sema.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//gapis/gfxapi/gles/glsl/ast:go_default_library",
        "//gapis/gfxapi/gles/glsl/parser:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["sema_test.go"],
    library = ":go_default_library",
    deps = [
        "//gapis/gfxapi/gles/glsl/ast:go_default_library",
        "//gapis/gfxapi/gles/glsl/evaluator:go_default_library",
        "//gapis/gfxapi/gles/glsl/parser:go_default_library",
    ],
)
