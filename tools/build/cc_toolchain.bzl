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

# Automatically configures the CC toolchain. This is adapted from
# Bazel's cc_configure, but will use mingw on Windows.

load("@bazel_tools//tools/cpp:lib_cc_configure.bzl", "get_cpu_value", "resolve_labels")
load("@bazel_tools//tools/cpp:osx_cc_configure.bzl", "configure_osx_toolchain")
load("@bazel_tools//tools/cpp:unix_cc_configure.bzl", "configure_unix_toolchain")
load("@bazel_tools//tools/osx:xcode_configure.bzl", "run_xcode_locator")

def _find_cc(repository_ctx):
    cc = repository_ctx.os.environ.get("CC")
    if cc == None:
        cc = repository_ctx.which("gcc")
    if cc == None:
        cc = "c:/tools/msys64/mingw64/bin/gcc.exe"
    cc = repository_ctx.path(cc)
    if not cc.exists:
        fail(("Couldn't find gcc at %s") % cc)
    return cc

def _get_inc_directories(repository_ctx, cc):
    cmd = [cc, "-E", "-xc++", "-Wp,-v", str(repository_ctx.path("empty.cpp"))]
    r = repository_ctx.execute(cmd, environment = { "PATH": str(cc.dirname) })
    if r.return_code != 0:
        fail(("Failed to execute '%s' to get include dirs: %s\n%s") % (cmd, r.return_code, r.stderr))
    s = r.stderr.find("#include <...>")
    e = r.stderr.find("End of search list", s)
    if s == -1 or e == -1:
        return []
    return [repository_ctx.path(l.strip()) for l in r.stderr[s:e - 1].splitlines()[1:]]

def _compile_wrapper(repository_ctx, cc, name):
    cmd = [
        cc,
        str(repository_ctx.path(name + ".cpp")),
        str(repository_ctx.path("file_collector.cpp")),
        "-o", str(repository_ctx.path(name + ".exe")),
        "-lstdc++", "-lshlwapi"
    ]
    r = repository_ctx.execute(cmd, environment = { "PATH": str(cc.dirname) })
    if r.return_code != 0:
        fail(("Failed to build %s: %s\n%s") % (name, r.return_code, r.stderr))

def _configure_windows_toolchain(repository_ctx):
    repository_ctx.file("empty.cpp", executable = False)

    cc = _find_cc(repository_ctx)
    inc = _get_inc_directories(repository_ctx, cc)

    repository_ctx.symlink(Label("@gapid//tools/build/mingw_toolchain:file_collector.cpp"), "file_collector.cpp")
    repository_ctx.symlink(Label("@gapid//tools/build/mingw_toolchain:file_collector.h"), "file_collector.h")

    repository_ctx.template("gcc_wrapper.cpp", Label("@gapid//tools/build/mingw_toolchain:gcc_wrapper.cpp.tpl"), {
        "%{CC}": str(cc),
    })
    _compile_wrapper(repository_ctx, cc, "gcc_wrapper")

    repository_ctx.template("ar_wrapper.cpp", Label("@gapid//tools/build/mingw_toolchain:ar_wrapper.cpp.tpl"), {
        "%{AR}": str(cc.dirname) + "/ar.exe",
    })
    _compile_wrapper(repository_ctx, cc, "ar_wrapper")

    repository_ctx.symlink(Label("@gapid//tools/build/mingw_toolchain:mingw.BUILD"), "BUILD.bazel")
    repository_ctx.template("toolchain.bzl", Label("@gapid//tools/build/mingw_toolchain:toolchain.bzl.in"), {
        "%{BINDIR}": str(cc.dirname),
        "%{GCC_WRAPPER}": str(repository_ctx.path("gcc_wrapper.exe")),
        "%{AR_WRAPPER}": str(repository_ctx.path("ar_wrapper.exe")),
        "%{CXX_BUILTIN_INCLUDE_DIRECTORIES}": ",\n".join([
            ("\"%s\"" % p) for p in inc
        ]),
    }, executable = False)

def _generate_cpp_only_build_file(repository_ctx, cpu_value, paths):
    repository_ctx.template(
        "BUILD",
        paths["@bazel_tools//tools/cpp:BUILD.toolchains.tpl"],
        {"%{name}": cpu_value},
    )

def _cc_autoconf_toolchains_impl(repository_ctx):
    paths = resolve_labels(repository_ctx, [
        "@bazel_tools//tools/cpp:BUILD.toolchains.tpl",
        "@bazel_tools//tools/osx/crosstool:BUILD.toolchains",
        "@bazel_tools//tools/osx/crosstool:osx_archs.bzl",
        "@bazel_tools//tools/osx:xcode_locator.m",
    ])
    cpu_value = get_cpu_value(repository_ctx)

    should_detect_cpp_toolchain = True
    should_use_cpp_only_toolchain = False
    should_use_xcode = False

    if cpu_value.startswith("darwin"):
        (xcode_toolchains, _xcodeloc_err) = run_xcode_locator(
            repository_ctx,
            paths["@bazel_tools//tools/osx:xcode_locator.m"],
        )
        if xcode_toolchains:
            repository_ctx.symlink(paths["@bazel_tools//tools/osx/crosstool:BUILD.toolchains"], "BUILD")
            repository_ctx.symlink(paths["@bazel_tools//tools/osx/crosstool:osx_archs.bzl"], "osx_archs.bzl")
        else:
            _generate_cpp_only_build_file(repository_ctx, cpu_value, paths)
    else:
        _generate_cpp_only_build_file(repository_ctx, cpu_value, paths)

_cc_autoconf_toolchains = repository_rule(
    implementation = _cc_autoconf_toolchains_impl,
    configure = True,
)

def _cc_autoconf_impl(repository_ctx):
  cpu_value = get_cpu_value(repository_ctx)
  if cpu_value == "x64_windows":
    _configure_windows_toolchain(repository_ctx)
  elif cpu_value == "darwin":
    configure_osx_toolchain(repository_ctx, cpu_value, {})
  else:
    configure_unix_toolchain(repository_ctx, cpu_value, {})

MSVC_ENVVARS = [
    "BAZEL_VC",
    "BAZEL_VC_FULL_VERSION",
    "BAZEL_VS",
    "BAZEL_WINSDK_FULL_VERSION",
    "VS90COMNTOOLS",
    "VS100COMNTOOLS",
    "VS110COMNTOOLS",
    "VS120COMNTOOLS",
    "VS140COMNTOOLS",
    "VS150COMNTOOLS",
    "VS160COMNTOOLS",
    "TMP",
    "TEMP",
]

_cc_autoconf = repository_rule(
    environ = [
        "ABI_LIBC_VERSION",
        "ABI_VERSION",
        "BAZEL_COMPILER",
        "BAZEL_HOST_SYSTEM",
        "BAZEL_CXXOPTS",
        "BAZEL_LINKOPTS",
        "BAZEL_LINKLIBS",
        "BAZEL_LLVM_COV",
        "BAZEL_PYTHON",
        "BAZEL_SH",
        "BAZEL_TARGET_CPU",
        "BAZEL_TARGET_LIBC",
        "BAZEL_TARGET_SYSTEM",
        "BAZEL_USE_LLVM_NATIVE_COVERAGE",
        "BAZEL_LLVM",
        "BAZEL_IGNORE_SYSTEM_HEADERS_VERSIONS",
        "USE_CLANG_CL",
        "CC",
        "CC_CONFIGURE_DEBUG",
        "CC_TOOLCHAIN_NAME",
        "CPLUS_INCLUDE_PATH",
        "DEVELOPER_DIR",
        "GCOV",
        "HOMEBREW_RUBY_PATH",
        "SYSTEMROOT",
        "USER",
    ] + MSVC_ENVVARS,
    implementation = _cc_autoconf_impl,
    configure = True,
)

def cc_configure():
    _cc_autoconf_toolchains(name = "local_config_cc_toolchains")
    _cc_autoconf(name = "local_config_cc")
    native.bind(name = "cc_toolchain", actual = "@local_config_cc//:toolchain")
    native.register_toolchains(
        "@local_config_cc_toolchains//:all",
    )
