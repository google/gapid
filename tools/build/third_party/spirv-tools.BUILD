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
    deps = [
        "@spirv-headers//:spirv-headers",
        "@//tools/build/third_party:spirv-tools-generated",
    ],
    strip_include_prefix = "source/",
    visibility = ["//visibility:private"],
)

cc_library(
    name = "spirv-tools",
    deps = [
        ":spirv-source",
        ":spirv-include",
    ],
    hdrs = glob([
        "include/spirv-tools/*",
        "source/**/*.h",
    ]),
    include_prefix = "third_party/SPIRV-Tools/",
    visibility = ["//visibility:public"],
)