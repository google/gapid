load("@//tools/build:rules.bzl", "copy")

copy(
    name = "config",
    src = "@//tools/build/third_party:cityhash-config.h",
    dst = "config.h",
)

cc_library(
    name = "cityhash",
    srcs = [
        "src/city.cc",
        ":config",
    ],
    hdrs = [
        "src/city.h",
    ],
    copts = [
        "-I$(BINDIR)/external/cityhash",
    ],
    strip_include_prefix = "src/",
    visibility = ["//visibility:public"],
)