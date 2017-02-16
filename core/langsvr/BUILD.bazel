load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "body.go",
        "codelens.go",
        "command.go",
        "completion.go",
        "diagnositcs.go",
        "doc.go",
        "document.go",
        "formatting_options.go",
        "highlight.go",
        "langsvr.go",
        "position.go",
        "signature.go",
        "source.go",
        "symbol.go",
        "textedit.go",
        "uri.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/event/task:go_default_library",
        "//core/langsvr/protocol:go_default_library",
        "//core/log:go_default_library",
        "//core/math/sint:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = ["body_test.go"],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
    ],
)
