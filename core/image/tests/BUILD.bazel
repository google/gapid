load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["image_test.go"],
    data = glob([
        "*.png",
        "*.bin",
    ]),
    deps = ["//core/image:go_default_library"],
)
