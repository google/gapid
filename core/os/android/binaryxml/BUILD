load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "debuggable.go",
        "decode.go",
        "doc.go",
        "string_pool.go",
        "value.go",
        "xml_attribute.go",
        "xml_cdata.go",
        "xml_context.go",
        "xml_end_element.go",
        "xml_end_namespace.go",
        "xml_resource_map.go",
        "xml_start_element.go",
        "xml_start_namespace.go",
        "xml_tree.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = [
        "debuggable_test.go",
        "decode_test.go",
    ],
    data = glob(["testdata/*"]),
    library = ":go_default_library",
    deps = ["//core/assert:go_default_library"],
)
