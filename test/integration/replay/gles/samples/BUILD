load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "builder.go",
        "clear_backbuffer.go",
        "draw_textured_square.go",
        "samples.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//gapis/atom:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/gfxapi/gles:go_default_library",
        "//gapis/memory:go_default_library",
    ],
)
