load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "early_terminator.go",
        "file_log.go",
        "injector.go",
        "trace_log.go",
        "transformer.go",
        "transforms.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/log:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/config:go_default_library",
        "//gapis/gfxapi:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = [
        "early_terminator_test.go",
        "injector_test.go",
    ],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/test:go_default_library",
    ],
)
