load("//tools/build:rules.bzl", "extract", "filehash")

def _strip_impl(ctx):
    outs = depset()
    if ctx.fragments.cpp.cpu == ctx.attr.abi:
        out = ctx.new_file("lib/{}/{}".format(ctx.attr.abi, ctx.file.lib.basename))
        ctx.actions.run(
            executable = ctx.fragments.cpp.strip_executable,
            arguments = ["-s", "-o", out.path, ctx.file.lib.path],
            inputs = [ctx.file.lib] + ctx.files._ndk,
            outputs = [out],
        )
        outs += [out]
    return struct(
        files = outs
    )

_strip = rule(
    implementation = _strip_impl,
    attrs = {
        "lib": attr.label(
            single_file = True,
            allow_files = True,
            mandatory = True,
        ),
        "abi": attr.string(),
        "_ndk": attr.label(
            default = "@androidndk//:files",
        )
    },
    fragments = ["cpp"]
)

def gapid_apk(name = "", abi = "", pkg = "", libs = {}):
    natives = []
    fatapks = []
    for lib in libs:
        libname = name + "_" + lib
        fatapk = "{}:{}.apk".format(libs[lib], lib)
        natives += [":" + libname]
        fatapks += [fatapk]
        extract(
            name = libname + "_unstripped",
            zip = fatapk,
            entries = ["lib/{}/lib{}.so".format(abi, lib)],
            dir = "unstripped",
        )
        _strip(
            name = libname,
            lib = ":" + libname + "_unstripped",
            abi = abi,
        )
    filehash(
        name = name+"_manifest",
        template = "AndroidManifest.xml.in",
        out = name + "/" + "AndroidManifest.xml",
        replace = "£{srchash}",
        srcs = fatapks + ["//gapidapk/android/app/src/main:gapid"],
        visibility = ["//visibility:public"],
    )
    native.cc_library(
        name = name+"_native",
        linkstatic = 1,
        srcs = select({
            "//tools/build:android-" + name: natives,
            "//conditions:default" : [],
        })
    )
    native.android_binary(
        name = name,
        manifest_values = {
            "name": pkg,
        },
        custom_package = "com.google.android.gapid",
        manifest = ":"+name+"_manifest",
        manifest_merger = "android",
        deps = [
            "//gapidapk/android/app/src/main:gapid",
            ":" + name + "_native",
        ],
        visibility = ["//visibility:public"],
    )
