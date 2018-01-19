cc_library(
    name = "spirv-cross",
    srcs = glob([
        "*.h",
        "*.cpp",
        "*.hpp",
    ]),
    hdrs = [
        "spirv_glsl.hpp",
    ],
    include_prefix = "third_party/SPIRV-Cross",
    visibility = ["//visibility:public"],
    deps = [
        "@spirv-headers//:spirv-headers",
        "@spirv-tools//:spirv-tools",
    ],
)
