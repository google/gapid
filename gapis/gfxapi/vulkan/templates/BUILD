load("//tools/build:rules.bzl", "api_template")

package(default_visibility = ["//visibility:public"])

api_template(
    name = "api_exports.cpp",
    template = "api_exports.cpp.tmpl",
    includes = ["//gapis/gfxapi/templates:templates"],
    outputs = [
        "{api}_exports.h",
        "{api}_exports.cpp",
        "{api}_VkCommandBuffer_exports.cpp",
        "{api}_VkDevice_exports.cpp",
        "{api}_VkInstance_exports.cpp",
        "{api}_VkPhysicalDevice_exports.cpp",
        "{api}_VkQueue_exports.cpp",
    ],
)

api_template(
    name = "api_imports.cpp",
    template = "api_imports.cpp.tmpl",
    includes = ["//gapis/gfxapi/templates:templates"],
    outputs = ["{api}_imports.cpp"],
)

api_template(
    name = "vk_spy_helpers.cpp",
    template = "vk_spy_helpers.cpp.tmpl",
    includes = [
        "vulkan_common.tmpl",
        "//gapis/gfxapi/templates:templates",
    ],
    outputs = ["{api}_spy_helpers.cpp"],
)
