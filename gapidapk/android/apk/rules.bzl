# Copyright (C) 2018 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("//tools/build:rules.bzl", "extract", "filehash")

def _strip_impl(ctx):
    outs = depset()
    if ctx.fragments.cpp.cpu == ctx.attr.abi:
        out = ctx.new_file("lib/{}/{}".format(ctx.attr.abi, ctx.file.lib.basename))
        ctx.actions.run(
            executable = ctx.fragments.cpp.strip_executable,
            arguments = ["--strip-unneeded", "-o", out.path, ctx.file.lib.path],
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
        replace = "{srchash}",
        srcs = fatapks + [
            "AndroidManifest.xml.in",
            "//gapidapk/android/app/src/main:source",
        ],
        visibility = ["//visibility:public"],
    )
    native.filegroup(
        name = name + "_libs",
        srcs = select({
            "//tools/build:debug": [l + "_unstripped" for l in natives],
            "//conditions:default" : natives,
        })
    )
    native.cc_library(
        name = name + "_native",
        linkstatic = 1,
        srcs = select({
            "//tools/build:android-" + name: [name + "_libs"],
            "//conditions:default": [],
        })
    )
    native.android_binary(
        name = name,
        manifest_values = {
            "name": pkg,
        },
        custom_package = "com.google.android.gapid",
        manifest = ":" + name + "_manifest",
        deps = [
            "//gapidapk/android/app/src/main:gapid",
            ":" + name + "_native",
        ],
        visibility = ["//visibility:public"],
    )
