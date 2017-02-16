load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")
load("//tools/build:rules.bzl", "go_proto_sources", "java_proto_library")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
    deps = [
        "//core/stream:pb",
    ],
)

go_proto_sources(
    name = "pb_go",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

java_proto_library(
    name = "pb_java",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "astc.go",
        "atc.go",
        "convert.go",
        "doc.go",
        "etc1.go",
        "etc2.go",
        "format.go",
        "id.go",
        "image.go",
        "png.go",
        "resizer.go",
        "rgba_f32.go",
        "s3.go",
        "s3_dxt1_rgb.go",
        "s3_dxt1_rgba.go",
        "s3_dxt3_rgba.go",
        "s3_dxt5_rgba.go",
        "thumbnailer.go",
        "uncompressed.go",
        ":pb_go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/data/protoutil:go_default_library",
        "//core/math/sint:go_default_library",
        "//core/math/u64:go_default_library",
        "//core/os/device:go_default_library",
        "//core/stream:go_default_library",
        "//core/stream/fmts:go_default_library",
        "//framework/binary:go_default_library",
        "//gapis/database:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "image_test.go",
        "rgba_f32_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/data/endian:go_default_library",
        "//core/math/f32:go_default_library",
        "//core/os/device:go_default_library",
        "//gapis/database:go_default_library",
    ],
)
