load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["fuzz_test.go"],
    deps = [
        "//core/log:go_default_library",
        "//gapil:go_default_library",
        "//gapil/resolver:go_default_library",
    ],
)
