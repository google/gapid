load("@//tools/build:rules.bzl", "copy")
cc_library(
    name = "breakpad",
    srcs = [
    ],
    hdrs = glob([
        "src/**/*.h",
    ]),
    copts = [
        "-I$(BINDIR)/external/brekpad",
    ],
    strip_include_prefix = "src/",
    visibility = ["//visibility:public"],
)