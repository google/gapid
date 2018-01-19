cc_library(
    name = "spirv-include",
    hdrs = glob([
        "include/**/*.h",
        "include/**/*.hpp",
    ]),
    strip_include_prefix = "include/",
    visibility = ["//visibility:private"],
)

cc_library(
    name = "spirv-source",
    hdrs = glob([
        "source/**/*.h",
        "source/**/*.inc",
    ]),
    strip_include_prefix = "source/",
    visibility = ["//visibility:private"],
    deps = [
        "@//tools/build/third_party:spirv-tools-generated",
        "@spirv-headers//:spirv-headers",
    ],
)

cc_library(
    name = "spirv-tools",
    srcs = glob([
        "*.h",
        "*.cpp",
        "*.hpp",
        "source/**/*.h",
        "source/**/*.cpp",
        "source/**/*.hpp",
    ]),
    hdrs = glob([
        "include/spirv-tools/*",
        "source/**/*.h",
    ]),
    include_prefix = "third_party/SPIRV-Tools/",
    visibility = ["//visibility:public"],
    deps = [
        ":spirv-include",
        ":spirv-source",
        "@spirv-headers//:spirv-headers",
    ],
)
