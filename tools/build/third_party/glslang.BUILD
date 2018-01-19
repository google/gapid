load("@//tools/build:rules.bzl", "cc_copts")

cc_library(
    name = "glslang",
    srcs = glob([
        "glslang/GenericCodeGen/*.h",
        "glslang/GenericCodeGen/*.cpp",
        "glslang/Include/*.h",
        "glslang/MachineIndependent/**/*.h",
        "glslang/MachineIndependent/**/*.cpp",
        "glslang/OSDependent/*.h",
        "glslang/OSDependent/*.cpp",
        "glslang/Public/*.h",
        "glslang/Public/*.cpp",
        "OGLCompilersDLL/*.h",
        "OGLCompilersDLL/*.cpp",
        "SPIRV/*.h",
        "SPIRV/*.cpp",
        "SPIRV/*.hpp",
    ]) + select({
        "@//tools/build:windows": glob(["glslang/OSDependent/Windows/*.cpp"]),
        "//conditions:default": glob(["glslang/OSDependent/Unix/*.cpp"]),
    }),
    hdrs = glob([
        "glslang/Include/*.h",
        "glslang/Public/*.h",
        "glslang/MachineIndependent/*.h",
        "SPIRV/*.h",
    ]),
    copts = cc_copts(),
    include_prefix = "third_party/glslang",
    visibility = ["//visibility:public"],
)
