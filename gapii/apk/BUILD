load("//tools/build:rules.bzl", "android_native")

android_native(
    name = "gapii",
    deps = ["//gapii/cc:gapii"],
    visibility = ["//visibility:public"],
)

android_native(
    name = "interceptor",
    deps = ["//gapii/interceptor-lib/cc:interceptor"],
    visibility = ["//visibility:public"],
)

android_native(
    name = "VKLayerGraphicsSpy",
    deps = ["//gapii/vulkan/vk_graphics_spy/cc:cc"],
    visibility = ["//visibility:public"],
)
