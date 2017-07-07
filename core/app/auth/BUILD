load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "auth.go",
        "doc.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/os/device:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_google_grpc//metadata:go_default_library",
        "@org_golang_x_net//context:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["auth_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
    ],
)
