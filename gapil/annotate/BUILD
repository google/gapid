load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "annotate.go",
        "container.go",
        "doc.go",
        "entity.go",
        "location.go",
        "nested.go",
        "path.go",
        "pointer.go",
        "snippets.go",
        "symbol.go",
        "verb.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/vle:go_default_library",
        "//gapil/semantic:go_default_library",
        "//gapil/semantic/printer:go_default_library",
        "//gapil/snippets:go_default_library",
    ],
)
