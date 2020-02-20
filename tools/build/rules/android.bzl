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

load("//tools/build/rules:common.bzl", "copy")

def android_native(name, deps=[], **kwargs):
    copied = name+"fake-src"
    copy(
        name=copied,
        src="//tools/build/rules:Ignore.java",
        dst="Ignore{}.java".format(name),
        visibility = ["//visibility:private"],
    )
    native.android_binary(
        name = name,
        deps = deps,
        manifest = "//tools/build/rules:AndroidManifest.xml",
        custom_package = "com.google.android.gapid.ignore",
        srcs = [":"+copied],
        **kwargs
    )

def _android_cc_binary_impl(ctx):
  outs = []
  groups = {}
  base = ctx.attr.out
  if base == "":
      base = ctx.label.name
  for cpu, binary in ctx.split_attr.dep.items():
      src = binary.files.to_list()[0]
      out = ctx.actions.declare_file(cpu + "/" + base)
      ctx.actions.run_shell(
         command = "cp \"" + src.path + "\" \"" + out.path + "\"",
          inputs = [src],
          outputs = [out]
      )
      outs += [out]
      groups[cpu] = [out]

  return [
      DefaultInfo(files = depset(outs)),
      OutputGroupInfo(**groups),
  ]

_android_cc_binary = rule(
    implementation = _android_cc_binary_impl,
    attrs = {
        "out": attr.string(),
        "dep": attr.label(
            cfg = android_common.multi_cpu_configuration,
            allow_files = True,
        ),
    },
)

def android_native_binary(name, out = "", **kwargs):
    visibility = kwargs.pop("visibility", default = ["//visibility:public"])
    native.cc_binary(
        name = name + "-bin",
        visibility = ["//visibility:private"],
        **kwargs
    )
    _android_cc_binary(
        name = name,
        out = out,
        dep = ":" + name + "-bin",
        visibility = visibility,
    )


def _android_native_app_glue_impl(ctx):
    ctx.symlink(
        ctx.path(ctx.os.environ["ANDROID_NDK_HOME"] +
            "/sources/android/native_app_glue/android_native_app_glue.c"),
        "android_native_app_glue.c")
    ctx.symlink(
        ctx.path(ctx.os.environ["ANDROID_NDK_HOME"] +
            "/sources/android/native_app_glue/android_native_app_glue.h"),
        "android_native_app_glue.h")

    ctx.file("BUILD", "\n".join([
        "cc_library(",
        "    name = \"native_app_glue\",",
        "    srcs = [\"android_native_app_glue.c\", \"android_native_app_glue.h\"],",
        "    hdrs = [\"android_native_app_glue.h\"],",
        "    visibility = [\"//visibility:public\"],",
        ")"
    ]))

android_native_app_glue = repository_rule(
    implementation = _android_native_app_glue_impl,
    local = True,
    environ = [
        "ANDROID_NDK_HOME",
    ]
)

# Retrieve Vulkan validation layers from the Android NDK
# Since NDK r21, use the single VK_LAYER_KHRONOS_validation
def _ndk_vk_validation_layer(ctx):
    build = ""
    for abi in ["armeabi-v7a", "arm64-v8a", "x86", "x86_64"]:
        layerpath = abi + "/libVkLayer_khronos_validation.so"
        ctx.symlink(
            ctx.path(ctx.os.environ["ANDROID_NDK_HOME"] +
                        "/sources/third_party/vulkan/src/build-android/jniLibs/" + layerpath),
            ctx.path(layerpath),
        )

        build += "\n".join([
            "cc_library(",
            "    name = \"" + abi + "\",",
            "    srcs = glob([\"" + abi + "/libVkLayer*.so\"]),",
            "    visibility = [\"//visibility:public\"],",
            ")",
        ]) + "\n"

    ctx.file("BUILD", build)

ndk_vk_validation_layer = repository_rule(
    implementation = _ndk_vk_validation_layer,
    local = True,
    environ = [
        "ANDROID_NDK_HOME",
    ],
)
