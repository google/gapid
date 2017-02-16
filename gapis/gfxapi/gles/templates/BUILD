load("//tools/build:rules.bzl", "api_template")

package(default_visibility = ["//visibility:public"])

api_template(
    name = "api_exports.cpp",
    template = "api_exports.cpp.tmpl",
    includes = ["//gapis/gfxapi/templates:templates"],
    outputs = ["{api}_exports.cpp"], 
)

api_template(
    name = "api_imports.cpp",
    template = "api_imports.cpp.tmpl",
    includes = ["//gapis/gfxapi/templates:templates"],
    outputs = ["{api}_imports.cpp"],
)
