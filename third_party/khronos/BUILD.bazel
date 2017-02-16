licenses(["notice"])

cc_library(
    name = "khronos",
    hdrs = glob([
        "**/*.h",
        "**/*.hpp",
        "**/*.inc",
    ]),
    srcs = glob([
        "SPIRV-Cross/*.cpp",
        "SPIRV-Tools/source/*.cpp",
        "SPIRV-Tools/source/opt/*.cpp",
        "glslang/OGLCompilersDLL/*.cpp",
        "glslang/SPIRV/*.cpp",
        "glslang/glslang/GenericCodeGen/*.cpp",
        "glslang/glslang/MachineIndependent/*.cpp",
        "glslang/glslang/MachineIndependent/preprocessor/*.cpp",
        "glslang/hlsl/*.cpp",
    ], exclude = [
        "SPIRV-Tools/source/enum_set.cpp",
    ]) + select({
        "//tools/build:linux": glob(["glslang/glslang/OSDependent/Unix/*.cpp"]),
        "//tools/build:darwin": glob(["glslang/glslang/OSDependent/Unix/*.cpp"]),
        "//tools/build:windows": glob(["glslang/glslang/OSDependent/Windows/*.cpp"]),
    }),
    includes = [
        "SPIRV-Tools/external/spirv-headers/include",
        "SPIRV-Tools/include",
        "SPIRV-Tools/source",
        "glslang/OGLCompilersDLL",
    ],
    deps = select({
        "//tools/build:windows": ["@windows_sdk//:includes"],
        "//conditions:default": [],
    }),
    linkopts = select({
        "//tools/build:linux": ["-lpthread"],
        "//tools/build:darwin": [],
        "//tools/build:windows": [],
    }),
    visibility = ["//visibility:public"],
)
