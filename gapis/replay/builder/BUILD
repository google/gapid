load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "allocator.go",
        "builder.go",
        "constant_encoder.go",
        "function_info.go",
        "mapped_memory_range.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/math/interval:go_default_library",
        "//core/os/device:go_default_library",
        "//gapis/config:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/replay/asm:go_default_library",
        "//gapis/replay/protocol:go_default_library",
        "//gapis/replay/value:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = [
        "builder_test.go",
        "constant_encoder_test.go",
    ],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/replay/asm:go_default_library",
        "//gapis/replay/protocol:go_default_library",
        "//gapis/replay/value:go_default_library",
    ],
)
