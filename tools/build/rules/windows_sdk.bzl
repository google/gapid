_WINDOWS_SDK = "WINDOWS_SDK"
_WINDOWS_KIT = "WINDOWS_KIT"

def _get_environ_path(ctx, name, default=None):
    value = None
    mode = "guessed"
    if name in ctx.os.environ:
        value = ctx.os.environ[name]
        mode = "specified"
    else:
        value = default
    path = ctx.path(value)
    if not path.exists:
        fail(" {} {} does not exist = {}".format(mode, name, value))
    return path

def _windows_sdk_impl(ctx):
    win_includes = _get_environ_path(ctx, _WINDOWS_SDK, "C:/tools/msys64/usr/include/w32api")
    ctx.symlink(win_includes, "sdk/include")
    kit = _get_environ_path(ctx, _WINDOWS_KIT, "C:/Program Files (x86)/Windows Kits/8.1")
    ctx.symlink(kit.get_child("Include"), "kit/include")
    ctx.file("BUILD", content="""
cc_library(
    name = "includes",
    hdrs = glob([
        "sdk/include/**/*.h",
        "kit/include/**/*.h",
    ]),
    includes = [
        "sdk/include",
        "kit/include",
    ],
    defines = [
        "WIN32_LEAN_AND_MEAN",
        "_M_AMD64",
        "__USE_W32_SOCKETS",
    ],
    visibility = ["//visibility:public"],
)
    """)

windows_sdk = repository_rule(
    implementation=_windows_sdk_impl,
    local=True,
    # TODO: uncomment the environ line once bazel 0.4.5 is standardised
    #environ=[_WINDOWS_SDK, _WINDOWS_KIT],
)
