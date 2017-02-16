load("//tools/build:rules.bzl", "api_template")

package(default_visibility = ["//visibility:public"])

filegroup(
    name = "templates",
    srcs = glob(["**/*.tmpl"]),
)

api_template(
    name = "api",
    template = "api.go.tmpl",
    includes = [":templates"],
    outputs = ["api.go", "enum.go"], 
)

api_template(
    name = "mutate",
    template = "mutate.go.tmpl",
    includes = [":templates"],
    outputs = ["mutate.go"],
)

api_template(
    name = "convert",
    template = "convert.go.tmpl",
    includes = [":templates"],
    outputs = ["convert.go"],
)

api_template(
    name = "proto",
    template = "api.proto.tmpl",
    includes = [":templates"],
    outputs = ["api.proto"],
)

api_template(
    name = "specific_gfx_api.cpp",
    template = "specific_gfx_api.cpp.tmpl",
    includes = [":templates"],
    outputs = ["{api}_gfx_api.cpp"],
)

api_template(
    name = "specific_gfx_api.h",
    template = "specific_gfx_api.h.tmpl",
    includes = [":templates"],
    outputs = ["{api}_gfx_api.h"],
)

api_template(
    name = "vulkan_gfx_api_extras",
    template = "vulkan_gfx_api_extras.tmpl",
    includes = [":templates"],
    outputs = ["vulkan_gfx_api_extras.cpp"],
)

api_template(
    name = "api_imports.h",
    template = "api_imports.h.tmpl",
    includes = [":templates"],
    outputs = ["{api}_imports.h"],
)

api_template(
    name = "api_spy.h",
    template = "api_spy.h.tmpl",
    includes = [":templates"],
    outputs = ["{api}_spy.h"],
)

api_template(
    name = "api_spy.cpp",
    template = "api_spy.cpp.tmpl",
    includes = [":templates"],
    outputs = [
        "{api}_spy_0.cpp",
        "{api}_spy_1.cpp",
        "{api}_spy_2.cpp",
        "{api}_spy_3.cpp",
        "{api}_spy_4.cpp",
        "{api}_spy_5.cpp",
        "{api}_spy_6.cpp",
        "{api}_spy_7.cpp",
        "{api}_spy_subroutines_0.cpp",
        "{api}_spy_subroutines_1.cpp",
    ],
)

api_template(
    name = "api_types.h",
    template = "api_types.h.tmpl",
    includes = [":templates"],
    outputs = ["{api}_types.h"],
)

api_template(
    name = "opengl32_exports.def",
    template = "opengl32_exports.def.tmpl",
    includes = [":templates"],
    outputs = ["opengl32_exports.def"],
)

api_template(
    name = "opengl32_resolve.cpp",
    template = "opengl32_resolve.cpp.tmpl",
    includes = [":templates"],
    outputs = ["opengl32_resolve.cpp"],
)

api_template(
    name = "opengl32_x64.asm",
    template = "opengl32_x64.asm.tmpl",
    includes = [":templates"],
    outputs = ["opengl32_x64.asm"],
)

api_template(
    name = "opengl_framework_exports.cpp",
    template = "opengl_framework_exports.cpp.tmpl",
    includes = [":templates"],
    outputs = ["opengl_framework_exports.cpp"],
)
