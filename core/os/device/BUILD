load("@io_bazel_rules_go//go:def.bzl", "cgo_library", "go_library", "go_test")
load("//tools/build:rules.bzl", "go_proto_sources", "java_proto_library")

proto_library(
    name = "pb",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
)

go_proto_sources(
    name = "pb_go",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

cc_proto_library(
    name = "pb_cc",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

java_proto_library(
    name = "pb_java",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

cgo_library(
    name = "cgo_default_library",
    srcs = [
        "host_c.go",
    ],
    cdeps = [
        "//core/os/device/deviceinfo/cc:cc",
    ],
    clinkopts = [],
    visibility = ["//visibility:private"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/fault:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_library(
    name = "go_default_library",
    srcs = [
        "abi.go",
        "android.go",
        "architecture.go",
        "cpu.go",
        "device.go",
        "doc.go",
        "gpu.go",
        "hardware.go",
        "host.go",
        "id.go",
        "instance.go",
        "linux.go",
        "osx.go",
        ":pb_go",  # keep
    ],
    library = ":cgo_default_library",
    visibility = ["//visibility:public"],
    deps = [
        "//core/data/id:go_default_library",
        "//core/fault:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "abi_test.go",
        "android_test.go",
        "architecture_test.go",
        "cpu_test.go",
        "gpu_test.go",
        "host_test.go",
        "instance_test.go",
        "linux_test.go",
        "memory_layout_test.go",
        "osx_test.go",
    ],
    tags = ["manual"],  # TODO: This is only while it requires the x display
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/data/id:go_default_library",
        "//core/log:go_default_library",
    ],
)
