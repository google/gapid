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

# This is needed due to https://github.com/bazelbuild/bazel/issues/914

load("//tools/build/rules:cc.bzl", "cc_stripped_binary")

def _symbol_exports(name, exports):
    # Creates an assembly file that references all the exported symbols, so the linker won't trim them.
    native.genrule(
        name = name + "_syms",
        srcs = [exports],
        outs = [name + "_syms.S"],
        cmd = "cat $< | awk '{print \".global\",$$0}' > $@",
    )

    # Creates a linker script to export the public symbols/hide the rest.
    native.genrule(
        name = name + "_ldscript",
        srcs = [exports],
        outs = [name + ".ldscript"],
        cmd = "(" + ";".join([
            "echo '{ global:'",
            "cat $< | awk '{print $$0\";\"}'",
            "echo 'local: *;};'",
            "echo",
        ]) + ") > $@",
    )

    # Creates a OSX linker script to export the public symbols/hide the rest.
    native.genrule(
        name = name + "_osx_ldscript",
        srcs = [exports],
        outs = [name + "_osx.ldscript"],
        cmd = "cat $< | awk '{print \"_\"$$0}' > $@",
    )

def cc_dynamic_library(name, exports = "", visibility = ["//visibility:private"], deps = [], linkopts = [], **kwargs):
    _symbol_exports(name, exports)

    # All but one of these will fail, but the select in the filegroup
    # will pick up the correct one.
    cc_stripped_binary(
        name = name + ".so",
        srcs = [":" + name + "_syms"],
        deps = deps + [name + ".ldscript"],
        linkopts = linkopts + [
            "-Wl,--version-script", "$(location " + name + ".ldscript)",
            "-Wl,-z,defs",
        ],
        linkshared = 1,
        visibility = ["//visibility:private"],
        **kwargs
    )
    cc_stripped_binary(
        name = name + ".dylib",
        deps = deps + [name + "_osx.ldscript"],
        linkopts = linkopts + [
            "-Wl,-exported_symbols_list", "$(location " + name + "_osx.ldscript)",
            "-Wl,-dead_strip",
        ],
        linkshared = 1,
        visibility = ["//visibility:private"],
        **kwargs
    )
    cc_stripped_binary(
        name = name + ".dll",
        srcs = [":" + name + "_syms"],
        deps = deps + [name + ".ldscript"],
        linkopts = linkopts + ["-Wl,--version-script", "$(location " + name + ".ldscript)"],
        linkshared = 1,
        visibility = ["//visibility:private"],
        **kwargs
    )

    native.filegroup(
        name = name,
        visibility = visibility,
        srcs = select({
            "//tools/build:linux": [":" + name + ".so"],
            "//tools/build:fuchsia-arm64": [":" + name + ".so"],
            "//tools/build:darwin": [":" + name + ".dylib"],
            "//tools/build:windows": [":" + name + ".dll"],
            # Android
            "//conditions:default": [":" + name + ".so"],
        })
    )

def android_dynamic_library(name, exports = "", deps = [], linkopts = [], **kwargs):
    _symbol_exports(name, exports)

    # This doesn't actually create a dynamic library, but sets up the linking
    # correctly for when the android_binary actually creates the .so.
    native.cc_library(
        name = name,
        srcs = [":" + name + "_syms"],
        deps = deps + [name + ".ldscript"],
        linkopts = linkopts + [
            "-Wl,--version-script", "$(location " + name + ".ldscript)",
            "-Wl,--unresolved-symbols=report-all",
            "-Wl,--gc-sections",
            "-Wl,--exclude-libs,libgcc.a",
            "-Wl,-z,noexecstack,-z,relro,-z,now,-z,nocopyreloc",
        ],
        alwayslink = 1,
        **kwargs
    )
