exports_files([
    "source.properties",
])

config_setting(
    name = "linux",
    values = {
        "cpu": "k8",
    },
    visibility = ["//visibility:public"],
)

config_setting(
    name = "darwin",
    values = {
        "cpu": "darwin",
    },
    visibility = ["//visibility:public"],
)

config_setting(
    name = "windows",
    values = {
        "cpu": "x64_windows",
    },
    visibility = ["//visibility:public"],
)

config_setting(
    name = "config-armeabi",
    values = {
        "fat_apk_cpu": "armeabi-v7a",
    },
    visibility = ["//visibility:public"],
)

config_setting(
    name = "config-x86",
    values = {
        "fat_apk_cpu": "x86",
    },
    visibility = ["//visibility:public"],
)

config_setting(
    name = "config-aarch64",
    values = {
        "fat_apk_cpu": "aarch64",
    },
    visibility = ["//visibility:public"],
)

config_setting(
    name = "android-armeabi",
    values = {
        "cpu": "armeabi-v7a",
    },
    visibility = ["//visibility:public"],
)

config_setting(
    name = "android-x86",
    values = {
        "cpu": "x86",
    },
    visibility = ["//visibility:public"],
)

config_setting(
    name = "android-aarch64",
    values = {
        "cpu": "aarch64",
    },
    visibility = ["//visibility:public"],
)

alias(
    name="java_toolchain",
    actual="//tools/build/rules/proto:java_toolchain",
    visibility = ["//visibility:public"],
)