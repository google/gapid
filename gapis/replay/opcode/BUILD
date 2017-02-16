load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "disassemble.go",
        "doc.go",
        "opcodes.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/os/device:go_default_library",
        "//gapis/replay/protocol:go_default_library",
    ],
)
