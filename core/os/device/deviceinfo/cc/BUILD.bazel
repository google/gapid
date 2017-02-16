load("//tools/build:rules.bzl", "mm_library", "cc_copts", "cc_defines")

mm_library(
    name = "darwin_query",
    srcs = glob(["osx/*.mm"]),
    copy_hdrs = [
        "query.h",
    ],
    deps = [
        "//core/os/device:pb_cc",
    ]
)

cc_library(
    name = "cc",
    srcs = glob(["*.cpp"]) + select({
        "//tools/build:linux": glob(["linux/query.cpp"]),
        "//tools/build:windows": glob(["windows/query.cpp"]),
        "//tools/build:android-armeabi": glob([
            "android/*.cpp",
            "android/*.h",
        ]),
        "//tools/build:android-x86": glob([
            "android/*.cpp",
            "android/*.h",
        ]),
        "//tools/build:android-aarch64": glob([
            "android/*.cpp",
            "android/*.h",
        ]),
        "//conditions:default": [],
    }),
    deps = [
        "//core/cc",
        "@com_google_protobuf//:protobuf",
        "//core/os/device:pb_cc",
    ] + select({
        "//tools/build:darwin": [":darwin_query"],
        "//conditions:default": [],
    }),
    hdrs = glob(["*.h"]),
    defines = cc_defines(),
    copts = cc_copts(),
    linkopts = select({
        "//tools/build:linux": ["-lGL", "-lX11"],
        "//tools/build:darwin": ["-framework Cocoa", "-framework OpenGL"],
        "//tools/build:windows": [],
        "//conditions:default": ["-lEGL", "-lGLESv2", "-llog", "-landroid", "-lm"],
    }),
    visibility = ["//visibility:public"],
)