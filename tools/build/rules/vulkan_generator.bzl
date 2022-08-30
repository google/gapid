def _basic_vulkan_generator_impl(ctx):
    outs = [
        ctx.actions.declare_file("".join([ctx.attr.target, ".h"])),
        ctx.actions.declare_file("".join([ctx.attr.target, ".cc"])),
        ctx.actions.declare_file("".join([ctx.attr.target, "_tests.cc"])),
    ]

    ctx.actions.run(
        inputs = [ctx.file._xml],
        outputs = outs,
        arguments = [
            ctx.file._xml.path,
            ctx.attr.target,
            outs[0].dirname,
        ],
        mnemonic = ("".join(["BasicVulkanGenerator", ctx.attr.target])).replace("_", ""),
        executable = ctx.executable._generator,
        use_default_shell_env = True,
    )

    return [
        DefaultInfo(files = depset(outs)),
    ]

basic_vulkan_generator = rule(
    _basic_vulkan_generator_impl,
    attrs = {
        "_generator": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//vulkan_generator:main"),
        ),
        "_xml": attr.label(
            cfg = "host",
            allow_single_file = True,
            default = "@vulkan-headers//:vk.xml",
        ),
        "target": attr.string(
            mandatory = True,
        ),
    },
)