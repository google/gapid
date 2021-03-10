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

load("@gapid//tools/build/rules:common.bzl", "copy_exec")
load("@bazel_tools//tools/cpp:toolchain_utils.bzl", "find_cpp_toolchain")

_ANDROID_COPTS = [
    "-fdata-sections",
    "-ffunction-sections",
    "-fvisibility-inlines-hidden",
    "-DANDROID",
    "-DTARGET_OS_ANDROID",
]

# This should probably all be done by fixing the toolchains...
def cc_copts():
    return ["-Werror"] + select({
        "@gapid//tools/build:linux": ["-DTARGET_OS_LINUX"],
        "@gapid//tools/build:darwin": ["-DTARGET_OS_OSX"],
        "@gapid//tools/build:windows": ["-DTARGET_OS_WINDOWS"],
        "@gapid//tools/build:android-armeabi-v7a": _ANDROID_COPTS,
        "@gapid//tools/build:android-arm64-v8a": _ANDROID_COPTS,
        "@gapid//tools/build:android-x86": _ANDROID_COPTS,
    })

# Strip rule implementation, which invokes the cc_toolchain.strip_executable
# to strip debugging information from binaries.
def _strip_impl(ctx):
    if len(ctx.files.srcs) != 1:
        fail("strip rule with multiple inputs")

    src = ctx.files.srcs[0]
    extension = src.extension
    if not extension == "":
        extension = "." + extension
    if ctx.label.name.endswith(extension):
        extension = ""
    out = ctx.actions.declare_file(ctx.label.name + extension)

    flags = []
    cc_toolchain = find_cpp_toolchain(ctx)
    if cc_toolchain.cpu == "k8" or cc_toolchain.cpu == "x64_windows":
        flags = ["--strip-unneeded", "-p"]
    elif cc_toolchain.cpu == "darwin_x86_64":
        flags = ["-x"]
    elif cc_toolchain.cpu == "darwin":
        fail("Please install Xcode and setup the path using xcode-select. You need Xcode, the CLI tools are not enough.")
    else:
        fail("Unhandled CPU type in strip rule: " + cc_toolchain.cpu)

    ctx.actions.run(
        executable = cc_toolchain.strip_executable,
        arguments = flags + ["-o", out.path, src.path],
        inputs = [src],
        outputs = [out],
    )
    return struct(
        files = depset([out]),
        runfiles = ctx.runfiles(collect_data = True),
        executable = out,
    )

# Strip rule to strip debugging information from binaries. Has a single
# "src" attribute, which should point to the binary to be stripped.
strip = rule(
    _strip_impl,
    attrs = {
        # Needs to be a list and called srcs, so collect_data above will work.
        # If more than one label is provided, strip will fail.
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
            allow_empty = False,
        ),
        "_cc_toolchain": attr.label(
            default = Label("@bazel_tools//tools/cpp:current_cc_toolchain"),
        ),
    },
    executable = True,
)

# Symbol rule implementation, which invokes the _dump_syms binary to generate
# a symbol dump file that can be uploaded to the crash server to symbolize
# stack traces of uploaded crash dumps.
def _symbols_impl(ctx):
    out = ctx.actions.declare_file(ctx.label.name)
    bin = ctx.file.src
    cc_toolchain = find_cpp_toolchain(ctx)
    if cc_toolchain.cpu.startswith("darwin"):
        dsym = ctx.actions.declare_directory(bin.basename + ".dSYM")
        ctx.actions.run_shell(
            command = "dsymutil  -o {} {}".format(dsym.path, bin.path),
            inputs = [bin],
            outputs = [dsym],
            use_default_shell_env = True,
        )
        bin = dsym
    ctx.actions.run_shell(
        command = "{} {} > {}".format(ctx.executable._dump_syms.path, bin.path, out.path),
        tools = [ctx.executable._dump_syms, bin],
        outputs = [out],
        use_default_shell_env = True,
    )
    return struct(
        files = depset([out]),
    )

# Symbol rule to dump the symbol information of a binary to be uploaded to the
# crash server. Has a single "src" attribute, which should point to the
# (unstripped) binary whose symbol information should be extracted. Generates
# the symbol data file that can be uploaded to the crash server.
symbols = rule(
    _symbols_impl,
    attrs = {
        "src": attr.label(
            allow_single_file = True,
        ),
        "_dump_syms": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("@breakpad//:dump_syms"),
        ),
        "_cc_toolchain": attr.label(
            default = Label("@bazel_tools//tools/cpp:current_cc_toolchain"),
        ),
    },
)

# Macro to replace cc_binary rules. Creates the following targets:
#  <name>_unstripped[.<extension>] - The cc_binary linked with debug _symbols
#  <name>.sym - The symbol dump file that can be uploaded to the crash server
#  <name> - The stripped cc_binary
def cc_stripped_binary(name, **kwargs):
    visibility = kwargs.pop("visibility")

    parts = name.rpartition(".")
    unstripped = name + "_unstripped" if parts[1] == "" else parts[0] + "_unstripped." + parts[2]
    stripped = name + "_stripped" if parts[1] == "" else parts[0] + "_stripped." + parts[2]

    native.cc_binary(
        name = unstripped,
        **kwargs
    )

    symbols(
        name = name + ".sym",
        src = unstripped,
        visibility = ["//visibility:public"],
    )

    strip(
        name = stripped,
        srcs = [unstripped],
    )

    copy_exec(
        name = name,
        srcs = select({
            "@gapid//tools/build:debug": [unstripped],
            "//conditions:default": [stripped],
        }),
        visibility = visibility,
    )
