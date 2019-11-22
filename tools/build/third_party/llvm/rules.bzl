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

load("@gapid//tools/build:rules.bzl", "cc_copts")

# llvm_cc_copts returns cc_copts() but with LLVM warnings muted.
def llvm_cc_copts():
    return cc_copts() + [
            # LLVM is full of warnings that are hard to separate into
            # individual compiler exclusions. Go nuclear.
            "-w",
            # LLVM is used to build parts of GAPID. We're not so interested in
            # debugging LLVM itself, but we would like the parts of the build
            # using LLVM to be as fast as possible, so always build LLVM
            # optimized.
            "-g0",
            "-O2",
            "-DNDEBUG",
        ]

def llvm_sources(name, exclude=[]):
    return native.glob([
            name+"/*.c",
            name+"/*.cpp",
            name+"/*.h",
            name+"/*.inc",
        ], exclude=exclude) + select({
            "@gapid//tools/build:windows": native.glob([
                name+"/Windows/*.h",
                name+"/Windows/*.inc",
            ], exclude=exclude),
            "//conditions:default": native.glob([
                name+"/Unix/*.h",
                name+"/Unix/*.inc",
            ], exclude=exclude),
        })

def llvmLibrary(name, path="", deps=[], excludes={}, extras={}):
    exclude = []
    if name in extras:
        deps += extras[name]
    if name in excludes:
        exclude += excludes[name]
    native.cc_library(
        name = name,
        srcs = llvm_sources(path, exclude=exclude),
        hdrs = native.glob([path + "/*.def"]),
        deps = deps,
        copts = llvm_cc_copts() + select({
            "@gapid//tools/build:linux": [],
            "@gapid//tools/build:darwin": [],
            "@gapid//tools/build:windows": ["-D__STDC_FORMAT_MACROS"],
            # Android
            "//conditions:default": [
                "-fno-rtti",
                "-fno-exceptions",
            ]
        }),
    )

def _tablegen_impl(ctx):
    include = ctx.files.deps[0].path.split("/include/")[0]+"/include"
    args = ctx.attr.flags + [
        "-I", include,
        "-I", ctx.file.table.dirname,
        "-o", ctx.outputs.generate.path,
        ctx.file.table.path,
    ]
    ctx.actions.run(
        inputs = [ctx.file.table] + ctx.files.deps,
        outputs = [ctx.outputs.generate],
        arguments = args,
        progress_message = "Table gen into %s" % ctx.outputs.generate,
        executable = ctx.executable._tblgen,
        use_default_shell_env = True,
    )

llvm_tablegen = rule(
    _tablegen_impl,
    attrs = {
        "table": attr.label(
            allow_single_file = True,
        ),
        "deps": attr.label_list(allow_files = True),
        "generate": attr.output(),
        "flags": attr.string_list(),
        "_tblgen": attr.label(
            cfg = "host",
            executable = True,
            allow_files = True,
            default = Label("@llvm//:llvm-tblgen"),
        ),
    },
)

def tablegen(name = "", table="", rules=[], deps=[], **kwargs):
    targets = []
    for entry in rules:
        generate = entry[0]
        flags = entry[1:]
        gen = name + "-" + generate.rsplit("/", 1)[-1]
        targets += [":"+gen]
        llvm_tablegen(
            name=gen,
            table=table,
            deps=deps,
            generate=generate,
            flags=flags,
        )
    native.cc_library(
        name = name,
        hdrs=targets,
        linkstatic=1,
        **kwargs
    )
