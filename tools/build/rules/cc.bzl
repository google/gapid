# This should probably all be done by fixing the toolchains...
def cc_copts():
    return select({
        "@//tools/build:linux": [],
        "@//tools/build:darwin": [],
        "@//tools/build:windows": [],
        "@//tools/build:android-armeabi": [],
        "@//tools/build:android-x86": [],
        "@//tools/build:android-aarch64": [],
    })

def cc_defines():
    return select({
        "@//tools/build:linux": ["TARGET_OS_LINUX"],
        "@//tools/build:darwin": ["TARGET_OS_OSX"],
        "@//tools/build:windows": ["TARGET_OS_WINDOWS"],
        "@//tools/build:android-armeabi": ["TARGET_OS_ANDROID"],
        "@//tools/build:android-x86": ["TARGET_OS_ANDROID"],
        "@//tools/build:android-aarch64": ["TARGET_OS_ANDROID"],
    }) + [
        "GAPID_VERSION_AND_BUILD=\\\"dunno\\\"", #TODO
    ]