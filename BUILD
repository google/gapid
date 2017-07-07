load("@io_bazel_rules_go//go:def.bzl", "go_prefix")
load("//tools/build:rules.bzl", "copy", "copy_tree", "copy_platform_binaries")

go_prefix("github.com/google/gapid")

# Aliases for easy access
alias(
    name = "gapic",
    actual = "//gapic:gapic",
)

alias(
    name = "gapit",
    actual = "//cmd/gapit",
)

alias(
    name = "gazelle",
    actual = "@io_bazel_rules_go_repository_tools//:bin/gazelle",
)

# Rules to build the expected installed structure for running
filegroup(
    name = "install",
    srcs = [
        ":install-gapic",
        ":install-gapit",
        ":install-runtime",
    ],
)

filegroup(
    name = "install-runtime",
    srcs = [
        "//:install-gapir",
        "//:install-gapis",
        "//:install-properties",
        "//:install-strings",
    ] + select({
        # TODO: temporary until android rules work on windows
        "//tools/build:windows": [],
        "//conditions:default": ["//:install-gapid.apk"],
    }),
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

filegroup(
    name = "install-gapid.apk",
    srcs = select({
        "//tools/build:config-armeabi": [":install-armeabi_gapid.apk"],
        "//tools/build:config-x86": [":install-x86_gapid.apk"],
        "//tools/build:config-aarch64": [":install-aarch64_gapid.apk"],
    }),
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

filegroup(
    name = "install-strings",
    srcs = [
        ":install-strings-en-us",
    ],
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

copy_platform_binaries(
    name = "install-gapic",
    src = "//gapic",
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

copy_platform_binaries(
    name = "install-gapir",
    src = "//cmd/gapir/cc:gapir",
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

copy_platform_binaries(
    name = "install-gapis",
    src = "//cmd/gapis",
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

copy_platform_binaries(
    name = "install-gapit",
    src = "//cmd/gapit",
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

copy(
    name = "install-armeabi_gapid.apk",
    src = "//gapidapk/android/apk:armeabi.apk",
    dst = "android/armeabi-v7a/gapid.apk",
    tags = ["manual"],
    visibility = ["//visibility:private"],
)

copy(
    name = "install-x86_gapid.apk",
    src = "//gapidapk/android/apk:x86.apk",
    dst = "android/x86/gapid.apk",
    tags = ["manual"],
    visibility = ["//visibility:private"],
)

copy(
    name = "install-strings-en-us",
    src = "//gapis/messages:en-us.stb",
    dst = "strings/en-us.stb",
    tags = ["manual"],
    visibility = ["//visibility:private"],
)

copy(
    name = "install-properties",
    src = "//tools/build:source.properties",
    dst = "source.properties",
    tags = ["manual"],
    visibility = ["//visibility:private"],
)

proto_library(
    name = "ptypes",
    srcs = glob(["google/protobuf/*.proto"]),
    visibility = ["//visibility:public"],
)

# Really hacky rule to generate java code in place for now...
genrule(
    name = "java_coders",
    srcs = ["//tools/build/codergen:java_coders"],
    outs = ["java_coders_stamp_file"],
    cmd = """
    BASE=$$(dirname $$(readlink WORKSPACE))
    for f in $(SRCS);do
        cp $$f $$BASE/gapic/src/$${f##*/gapic/src/}
    done
    touch $@
    """,
    local = True,
    tools = ["//cmd/codergen"],
)
