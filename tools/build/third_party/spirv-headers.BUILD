cc_library(
    name = "spirv-internal",
    hdrs = [
        "include/spirv/1.2/spirv.hpp",
        "include/spirv/1.2/spirv.h",
    ],
    strip_include_prefix = "include/",
    visibility = ["//visibility:private"],
)

cc_library(
    name = "spirv-headers",
    hdrs = [
        "include/spirv/1.2/spirv.hpp",
    ],
    deps = [
        ":spirv-internal",
    ],
    include_prefix = "third_party/SPIRV-Headers/",
    visibility = ["//visibility:public"],
)