java_library(
    name="platform",
    srcs = select({
        "//tools/build:linux": [
            "linux/com/google/gapid/glcanvas/GlCanvas.java",
        ],
        "//tools/build:darwin": [
            "osx/com/google/gapid/glcanvas/GlCanvas.java",
        ],
        "//tools/build:windows": [
            "windows/com/google/gapid/glcanvas/GlCanvas.java",
        ],
    }),
    visibility = ["//visibility:public"],
    deps=[
        "//gapic/third_party:swt",
        "//gapic/third_party:lwjgl",
    ],
)
