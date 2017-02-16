exports_files(glob([
    "*.BUILD",
    "*.h",
    "llvm/rules.bzl",
]))

cc_library(
    name = "llvm-platform-config",
    hdrs = select({
        "@//tools/build:linux": glob(["llvm/include-linux/**/*"]),
        "@//tools/build:darwin": glob(["llvm/include-darwin/**/*"]),
        "@//tools/build:windows": glob(["llvm/include-windows/**/*"]),
        "@//tools/build:android-armeabi": glob(["llvm/include-armeabi/**/*"]),
        "@//tools/build:android-x86": glob(["llvm/include-x86/**/*"]),
        "@//tools/build:android-aarch64": glob(["llvm/include-aarch64/**/*"]),
    }),
    strip_include_prefix = select({
        "//tools/build:linux": "llvm/include-linux",
        "//tools/build:darwin": "llvm/include-darwin",
        "//tools/build:windows": "llvm/include-windows",
        "//tools/build:android-armeabi": "llvm/include-armeabi",
        "//tools/build:android-x86": "llvm/include-x86",
        "//tools/build:android-aarch64": "llvm/include-aarch64",
    }),
    visibility = ["//visibility:public"],
)

cc_library(
    name = "llvm-config",
    hdrs = glob(["llvm/include/**/*"]),
    deps = [":llvm-platform-config"],
    strip_include_prefix = "llvm/include",
    visibility = ["//visibility:public"],
)
