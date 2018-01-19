load("//:version.bzl", "version_define_copts")

_ANDROID_COPTS = [
    "-fdata-sections",
    "-ffunction-sections",
    "-fvisibility-inlines-hidden",
    "-DANDROID",
    "-DTARGET_OS_ANDROID",
]

# This should probably all be done by fixing the toolchains...
def cc_copts():
    return version_define_copts() + select({
        "@//tools/build:linux": ["-DTARGET_OS_LINUX"],
        "@//tools/build:darwin": ["-DTARGET_OS_OSX"],
        "@//tools/build:windows": ["-DTARGET_OS_WINDOWS"],
        "@//tools/build:android-armeabi-v7a": _ANDROID_COPTS,
        "@//tools/build:android-arm64-v8a": _ANDROID_COPTS,
        "@//tools/build:android-x86": _ANDROID_COPTS,
    })
