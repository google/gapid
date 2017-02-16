load("//tools/build:rules.bzl", "cc_copts", "cc_defines")

filegroup(
    name = "headers",
    srcs = glob([
        "*.h",
        "gl/*.h",
    ]),
    visibility = ["//visibility:public"],
)

cc_library(
    name = "cc",
    srcs = glob([
        "*.cpp",
        "gl/*.cpp",
    ], exclude = [
        "*_test.cpp",
    ]) + select({
        "//tools/build:linux": glob(["linux/*.cpp"]),
        "//tools/build:darwin": glob(["osx/*.cpp"]),
        "//tools/build:windows": glob(["windows/*.cpp"]),
        "//conditions:default": glob(["android/*.cpp"]),
    }) + select({
        "//tools/build:windows": [],
        "//conditions:default": glob(["posix/*.cpp"]),
    }),
    defines = cc_defines(),
    copts = cc_copts(),
    visibility = ["//visibility:public"],
    hdrs = [":headers"],
    deps = [
        "@cityhash//:cityhash",
    ] + select({
        "//tools/build:windows": ["@windows_sdk//:includes"],
        "//conditions:default": [],
    }),
    linkopts = select({
        "//tools/build:linux": [],
        "//tools/build:darwin": [],
        "//tools/build:windows": ["-lws2_32"],
        "//conditions:default": [],
    }),
)
