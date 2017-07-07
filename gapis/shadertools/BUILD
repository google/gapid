load("@io_bazel_rules_go//go:def.bzl", "cgo_library", "go_library")

cgo_library(
    name = "cgo_default_library",
    srcs = ["shadertools.go"],
    cdeps = [
        "//gapis/shadertools/cc:cc",
    ],
    clinkopts = [],
    visibility = ["//visibility:private"],
)

go_library(
    name = "go_default_library",
    library = ":cgo_default_library",
    visibility = ["//visibility:public"],
)
