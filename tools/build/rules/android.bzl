load("//tools/build/rules:common.bzl", "copy")

def android_native(name, deps=[], **kwargs):
    copied = name+"fake-src"
    copy(
        name=copied,
        src="//tools/build/rules:Ignore.java",
        dst="Ignore{}.java".format(name),
        visibility = ["//visibility:private"],
        tags = ["manual"],
    )
    native.android_binary(
        name = name,
        deps = deps,
        manifest = "//tools/build/rules:AndroidManifest.xml",
        custom_package = "com.google.android.gapid.ignore",
        srcs = [":"+copied],
        tags = ["manual"],
        **kwargs
    )

def _android_ndk_file_impl(ctx):
    outs = depset()

    for f in ctx.attr.files:
        out = ctx.new_file(f.rpartition("/")[2])
        ctx.actions.run_shell(
            command = "cp {}/{} {}".format(
                ctx.files._ndk[0].path,
                f,
                out.path
            ),
            inputs = ctx.files._ndk,
            outputs = [out]
        )
        outs += [out]
    return struct(
        files = outs
    )

android_ndk_file = rule(
    implementation = _android_ndk_file_impl,
    attrs = {
        "files": attr.string_list(
            mandatory = True,
        ),
        "_ndk": attr.label(
            default = "@androidndk//:files",
        )
    },
)

def android_native_app_glue(name, **kwargs):
    android_ndk_file(
        name = "android_native_app_glue.c",
        files = ["sources/android/native_app_glue/android_native_app_glue.c"],
    )
    android_ndk_file(
        name = "android_native_app_glue.h",
        files = ["sources/android/native_app_glue/android_native_app_glue.h"],
    )
    native.cc_library(
        name = name,
        srcs = ["android_native_app_glue.c"],
        hdrs = ["android_native_app_glue.h"],
        strip_include_prefix = "", # force the virtual include link creation so the header is found
        **kwargs
    )
