load("//:version.bzl", "version_defines")

_ANDROID_COPTS = [
    "-fdata-sections",
    "-ffunction-sections",
    "-fvisibility-inlines-hidden",
    "-DANDROID",
]

# This should probably all be done by fixing the toolchains...
def cc_copts():
    return select({
        "@//tools/build:linux": [],
        "@//tools/build:darwin": [],
        "@//tools/build:windows": [],
        "@//tools/build:android-armeabi-v7a": _ANDROID_COPTS,
        "@//tools/build:android-arm64-v8a": _ANDROID_COPTS,
        "@//tools/build:android-x86": _ANDROID_COPTS,
    })

def cc_defines():
    return select({
        "@//tools/build:linux": ["TARGET_OS_LINUX"],
        "@//tools/build:darwin": ["TARGET_OS_OSX"],
        "@//tools/build:windows": ["TARGET_OS_WINDOWS"],
        "@//tools/build:android-armeabi-v7a": ["TARGET_OS_ANDROID"],
        "@//tools/build:android-arm64-v8a": ["TARGET_OS_ANDROID"],
        "@//tools/build:android-x86": ["TARGET_OS_ANDROID"],
    }) + version_defines()
