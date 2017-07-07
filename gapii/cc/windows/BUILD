load("//tools/build:rules.bzl", "apic")

apic(
    name = "generated",
    api = "//gapis/gfxapi/gles:api",
    templates = [
        "//gapis/gfxapi/templates:opengl32_exports.def",
        "//gapis/gfxapi/templates:opengl32_resolve.cpp",
        "//gapis/gfxapi/templates:opengl32_x64.asm",
    ]
)

filegroup(
    name = "srcs",
    srcs = [
        "wgl.h",
        "wgl.cpp",
        ":generated",
    ],
    visibility = ["//visibility:public"]
)
