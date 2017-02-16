load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
        "search.go",
        "server.go",
        "upload.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//core/app:go_default_library",
        "//core/data/search/script:go_default_library",
        "//core/data/stash:go_default_library",
        "//core/data/stash/grpc:go_default_library",
        "//core/data/stash/local:go_default_library",
        "//core/log:go_default_library",
        "//core/net/grpcutil:go_default_library",
        "//core/os/file:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
    ],
)

go_binary(
    name = "stash",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
