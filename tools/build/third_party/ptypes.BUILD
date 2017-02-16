load("@//tools/build:rules.bzl", "copy_tree")

copy_tree(
    name = "well_known_protos",
    srcs = [ "@com_google_protobuf//:well_known_protos" ],
    strip = "external/com_google_protobuf/src/",
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

proto_library(
    name = "ptypes",
    srcs = [":well_known_protos"],
    tags = ["manual"],
    visibility = ["//visibility:public"],
)

