load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "vle.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/pod:go_default_library",
        "//core/math/f16:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["vle_test.go"],
    deps = [
        ":go_default_library",
        "//core/data/pod:go_default_library",
        "//core/log:go_default_library",
        "//framework/binary/test:go_default_library",
    ],
)
