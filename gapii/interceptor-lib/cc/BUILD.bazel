load("//tools/build:rules.bzl", "cc_copts", "cc_defines")

cc_library(
    name = "interceptor",
    tags = ["manual"],
    srcs = glob([
        "lib/*.cc",
        "lib/*.h",
    ]) + select({
        "//tools/build:android-armeabi": glob([
            "lib/ARM/*.cc",
            "lib/ARM/*.h",
        ]),
        "//tools/build:android-x86": glob([
            "lib/X86/*.cc",
            "lib/X86/*.h",
        ]),
        "//tools/build:android-aarch64": glob([
            "lib/AArch64/*.cc",
            "lib/AArch64/*.h",
        ]),
    }),
    includes = ["lib"],
    hdrs = glob([
        "include/*.h",
        "include/*.hpp",
    ]),
    strip_include_prefix = "include",
    defines = cc_defines(),
    copts = cc_copts(),
    linkopts = select({
        "//tools/build:linux": [],
        "//tools/build:darwin": [],
        "//tools/build:windows": [],
        "//conditions:default": ["-lm", "-llog", "-lz"],
    }),
    deps = [
        "@llvm//:AArch64CodeGen",
        "@llvm//:AArch64Disassembler",
        "@llvm//:ARMCodeGen",
        "@llvm//:ARMDisassembler",
        "@llvm//:MipsCodeGen",
        "@llvm//:MipsDisassembler",
        "@llvm//:X86CodeGen",
        "@llvm//:X86Disassembler",
    ],
    visibility = ["//visibility:public"],
)
