java_import(
    name = "guava",
    jars = [
        "guava-20.0.jar",
    ],
    visibility = ["//visibility:public"],
)

java_import(
    name = "jface",
    jars = [
        "jface/org.eclipse.core.commands_3.8.0.v20160316-1921.jar",
        "jface/org.eclipse.core.runtime_3.12.0.v20160606-1342.jar",
        "jface/org.eclipse.equinox.common_3.8.0.v20160509-1230.jar",
        "jface/org.eclipse.jface_3.12.1.v20160923-1528.jar",
        "jface/org.eclipse.jface.databinding_1.8.1.v20161026-1531.jar",
        "jface/org.eclipse.jface.text_3.11.2.v20161113-1700.jar",
        "jface/org.eclipse.osgi_3.11.2.v20161107-1947.jar",
        "jface/org.eclipse.text_3.6.0.v20160503-1849.jar",
    ],
    visibility = ["//visibility:public"],
)

java_import(
    name = "protobuf",
    jars = [
        "protobuf-java-3.0.0-beta-4.jar",
    ],
    visibility = ["//visibility:public"],
)

java_import(
    name = "grpc",
    jars = [
        "grpc/grpc-context-1.0.1.jar",
        "grpc/grpc-core-1.0.1.jar",
        "grpc/grpc-okhttp-1.0.1.jar",
        "grpc/grpc-protobuf-1.0.1.jar",
        "grpc/grpc-protobuf-lite-1.0.1.jar",
        "grpc/grpc-stub-1.0.1.jar",
        "grpc/okhttp-2.5.0.jar",
        "grpc/okio-1.6.0.jar",
    ],
    visibility = ["//visibility:public"],
)

java_import(
    name = "swt",
    jars = select({
        "//tools/build:linux": [
            "platform/linux/swt.jar",
        ],
        "//tools/build:darwin": [
            "platform/osx/swt.jar",
        ],
        "//tools/build:windows": [
            "platform/windows/swt.jar",
        ],
    }),
    visibility = ["//visibility:public"],
)

java_import(
    name = "lwjgl",
    jars = [
        "lwjgl/lwjgl.jar",
        "lwjgl/lwjgl-opengl.jar",
    ] + select({
        "//tools/build:linux": [
            "platform/linux/lwjgl-natives.jar",
            "platform/linux/lwjgl-opengl-natives.jar",
        ],
        "//tools/build:darwin": [
            "platform/osx/lwjgl-natives.jar",
            "platform/osx/lwjgl-opengl-natives.jar",
        ],
        "//tools/build:windows": [
            "platform/windows/lwjgl-natives.jar",
            "platform/windows/lwjgl-opengl-natives.jar",
        ],
    }),
    visibility = ["//visibility:public"],
)
