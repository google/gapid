java_binary(
    name="gapic",
    main_class="com.google.gapid.Main",
    visibility = ["//visibility:public"],
    runtime_deps=[
        "//gapic/src/main",
        "//gapic/res:res",
    ],
    data = [
        "//:install-runtime",
        "//:install-gapit",
    ],
    jvm_flags = select({
        "//tools/build:darwin": ["-XstartOnFirstThread"],
        "//conditions:default": [],
    }),
    args = [
        "--gapid", ".",
    ]
)
