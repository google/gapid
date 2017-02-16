load("//tools/build:rules.bzl", "apic")

apic(
    name = "generated",
    api = "//gapis/gfxapi/gles:api",
    templates = [
        "//gapis/gfxapi/templates:opengl_framework_exports.cpp",
    ],
    visibility = ["//visibility:public"]
)
