load(":rules.bzl", "gapid_apk")

_NATIVE_LIBRARIES = {
    "deviceinfo": "//core/os/device/deviceinfo/apk",
    "VkLayer_VirtualSwapchain": "//core/vulkan/vk_virtual_swapchain/apk",
    "gapii": "//gapii/apk",
    "interceptor": "//gapii/apk",
    "VKLayerGraphicsSpy": "//gapii/apk",
    "gapir": "//gapir/apk",
}

gapid_apk(
    name = "armeabi", 
    abi = "armeabi-v7a",
    libs = _NATIVE_LIBRARIES,
)

gapid_apk(
    name = "x86", 
    abi = "x86",
    libs = _NATIVE_LIBRARIES,
)

filegroup(
    name = "gapid",
    srcs = select({
        "//tools/build:config-armeabi": [":armeabi"],
        "//tools/build:config-x86": [":x86"],
        "//tools/build:config-aarch64": [":aarch64"],
    }),
)
