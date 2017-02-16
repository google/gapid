load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "client.go",
        "doc.go",
        "pipe.go",
        "server.go",
        "stream.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/event:go_default_library",
        "//core/event/task:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["pipe_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/event/task:go_default_library",
        "//core/log:go_default_library",
    ],
)
