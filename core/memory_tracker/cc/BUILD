load("//tools/build:rules.bzl", "cc_copts", "cc_defines")

cc_library(
    name = "cc",
    srcs = glob([
        "*.cpp",
    ], exclude = [
        "*_test.cpp",
    ]),
    hdrs = glob([
        "*.h",
    ]),
    defines = cc_defines(),
    copts = cc_copts(),
    visibility = ["//visibility:public"],
    deps = [
        "//core/cc",
    ]
)
