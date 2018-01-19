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
