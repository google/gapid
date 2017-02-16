load("//tools/build:rules.bzl", "apic", "mm_library", "cc_copts", "cc_defines")

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
        "//gapis/gfxapi/templates:specific_gfx_api.cpp",
        "//gapis/gfxapi/templates:specific_gfx_api.h",
    ]
)

apic(
    name = "vulkan",
    api = "//gapis/gfxapi/vulkan:api",
    templates = [
        "//gapis/gfxapi/templates:specific_gfx_api.cpp",
        "//gapis/gfxapi/templates:specific_gfx_api.h",
        "//gapis/gfxapi/templates:vulkan_gfx_api_extras",
    ]
)

filegroup(
    name = "headers",
    srcs = glob([
        "*.h",
        "*.inl",
    ]),
    visibility = ["//visibility:public"],
)

mm_library(
    name = "darwin_renderer",
    srcs = glob(["osx/*.mm"]),
    defines = cc_defines(),
    hdrs = [
        "//core/cc:headers",
        ":headers",
        ":vulkan",
        ":gles",
    ],
    deps = [],
)

cc_library(
    name = "gapir",
    srcs = [
        ":gles",
        ":vulkan",
    ] + glob([
        "*.cpp"
    ], exclude = [
        "*_test.cpp"
    ]) + select({
        "//tools/build:linux": glob(["linux/*.cpp"]),
        "//tools/build:darwin": glob(["osx/*.cpp"]),
        "//tools/build:windows": glob(["windows/*.cpp"]),
        "//conditions:default": glob(["android/*.cpp"]),
    }),
    defines = cc_defines(),
    hdrs = [":headers"],
    deps = [
        "//core/cc",
        "//core/vulkan/vk_virtual_swapchain/cc",
    ] + select({
        "//tools/build:darwin": [":darwin_renderer"],
        "//conditions:default": [],
    }),
    copts = cc_copts(),
    linkopts = select({
        "//tools/build:linux": ["-ldl", "-lGL", "-lX11"],
        "//tools/build:darwin": ["-framework Cocoa", "-framework OpenGL"],
        "//tools/build:windows": [],
        "//conditions:default": [],
    }),
    visibility = ["//visibility:public"],
)
