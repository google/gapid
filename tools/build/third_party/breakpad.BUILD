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
    deps = [
        "@lss",
    ],
    strip_include_prefix = "src/",
    visibility = ["//visibility:public"],
)
