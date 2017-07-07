java_library(
    name="service",
    srcs = glob([
        "com/google/gapid/service/*.java",
        "com/google/gapid/service/atom/*.java",
        "com/google/gapid/service/memory/*.java",
        "com/google/gapid/service/snippets/*.java",
    ]),
    visibility = ["//visibility:public"],
    deps=[
        "//gapic/src/rpclib",
        "//gapil/snippets:pb_java",
        "//gapic/third_party:guava",
    ],
)