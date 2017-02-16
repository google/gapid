cc_binary(
    name = "protoc-gen-java",
    srcs = [
        "compiler/src/java_plugin/cpp/java_generator.h",
        "compiler/src/java_plugin/cpp/java_generator.cpp",
        "compiler/src/java_plugin/cpp/java_plugin.cpp",
    ],
    deps = [
        "@com_google_protobuf//:protoc_lib",
        "@com_google_protobuf//:protobuf",
    ],
    visibility = ["//visibility:public"],
)
