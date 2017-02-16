load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "changelist.go",
        "constants.go",
        "doc.go",
        "fetch.go",
        "get.go",
        "git.go",
        "log.go",
        "patch.go",
        "rebase.go",
        "reset_to_head.go",
        "sha.go",
        "status.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/os/shell:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["log_test.go"],
    library = ":go_default_library",
    deps = ["//core/assert:go_default_library"],
)
