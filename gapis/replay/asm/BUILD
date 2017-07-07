load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "instructions.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/pod:go_default_library",
        "//gapis/replay/opcode:go_default_library",
        "//gapis/replay/protocol:go_default_library",
        "//gapis/replay/value:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["instructions_test.go"],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//core/data/endian:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//gapis/replay/opcode:go_default_library",
        "//gapis/replay/protocol:go_default_library",
        "//gapis/replay/value:go_default_library",
    ],
)
