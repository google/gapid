load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "a.go",
        "bgr.go",
        "bgra.go",
        "d.go",
        "doc.go",
        "ds.go",
        "gray.go",
        "l.go",
        "la.go",
        "r.go",
        "rg.go",
        "rgb.go",
        "rgba.go",
        "sd.go",
        "xy.go",
        "xyz.go",
        "xyzw.go",
    ],
    visibility = ["//visibility:public"],
    deps = ["//core/stream:go_default_library"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["fmts_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/stream:go_default_library",
    ],
)
