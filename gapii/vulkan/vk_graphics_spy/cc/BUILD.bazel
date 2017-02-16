load("//tools/build:rules.bzl", "cc_copts", "cc_defines")

cc_library(
    name="cc",
    srcs=glob([
        "*.cpp",
    ]),
    hdrs=glob([
        "*.h",
    ]),
    deps=[
        "//core/vulkan/cc/include/vulkan",
    ],
    defines = cc_defines(),
    copts = cc_copts(),
    visibility = ["//visibility:public"],
)