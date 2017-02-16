load("//tools/build:rules.bzl", "copy_tree")

# This is an ugly ugly hack
# The bazel resource rules expect a specific directory structure, so we copy_resources
# into that struture first
copy_tree(
    name = "copy_resources",
    srcs = glob(["**/*"]),
    strip = PACKAGE_NAME + "/",
    to = "java",
)

java_library(
    name="res",
    resources=[":copy_resources"],
    visibility = ["//visibility:public"],
)

