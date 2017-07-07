load("//tools/build:rules.bzl", "apic", "cc_copts", "cc_defines")


apic(
    name = "core",
    api = "//gapis/gfxapi/core:api",
    templates = [
        "//gapis/gfxapi/templates:api_imports.h",
        "//gapis/gfxapi/templates:api_spy.h",
        "//gapis/gfxapi/templates:api_spy.cpp",
        "//gapis/gfxapi/templates:api_types.h",
    ]
)

apic(
    name = "gles",
    api = "//gapis/gfxapi/gles:api",
    templates = [
        "//gapis/gfxapi/templates:api_imports.h",
        "//gapis/gfxapi/templates:api_spy.h",
        "//gapis/gfxapi/templates:api_spy.cpp",
        "//gapis/gfxapi/templates:api_types.h",
        "//gapis/gfxapi/gles/templates:api_exports.cpp",
        "//gapis/gfxapi/gles/templates:api_imports.cpp",
    ]
)

apic(
    name = "vulkan",
    api = "//gapis/gfxapi/vulkan:api",
    templates = [
        "//gapis/gfxapi/templates:api_imports.h",
        "//gapis/gfxapi/templates:api_spy.h",
        "//gapis/gfxapi/templates:api_spy.cpp",
        "//gapis/gfxapi/templates:api_types.h",
        "//gapis/gfxapi/vulkan/templates:api_exports.cpp",
        "//gapis/gfxapi/vulkan/templates:api_imports.cpp",
        "//gapis/gfxapi/vulkan/templates:vk_spy_helpers.cpp",
    ]
)

cc_library(
    name = "gapii",
    srcs = [
        ":core",
        ":gles",
        ":vulkan",
    ] + glob([
        "*.cpp",
    ], exclude = [
        "*_test.cpp",
    ]) + select({
        "//tools/build:linux": glob(["linux/*.cpp"]),
        "//tools/build:darwin": ["//gapii/cc/osx:generated"],
        "//tools/build:windows": ["//gapii/cc/windows:srcs"],
        "//conditions:default": glob(["android/*.cpp"]),
    }),
    hdrs = glob([
        "*.h",
        "*.inl",
    ]),
    defines = cc_defines(),
    deps = [
        "//tools/build/codergen:cc_coders",
        "//core/cc",
        "//core/memory_tracker/cc",
    ],
    copts = cc_copts(),
    linkopts = select({
        "//tools/build:linux": ["-ldl", "-lGL", "-lX11"],
        "//tools/build:darwin": ["-framework Cocoa", "-framework OpenGL"],
        "//tools/build:windows": [],
        "//conditions:default": ["-lEGL", "-llog", "-lm"],
    }),
    visibility = ["//visibility:public"],
)
