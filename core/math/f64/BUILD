load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "f64.go",
        "vec3.go",
        "vec4.go",
    ],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "f64_test.go",
        "vec3_test.go",
        "vec4_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
    ],
)
