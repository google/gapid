
def _generate_impl(ctx):
    ctx.file_action(
        output = ctx.outputs.output,
        content = ctx.attr.content,
    )

generate = rule(
    attrs = {
        "output" : attr.output(mandatory=True),
        "content" : attr.string(mandatory=True),
    },
    implementation = _generate_impl,
)

def _copy_impl(ctx):
    ctx.action(
        inputs=ctx.files.src,
        outputs=[ctx.outputs.dst],
        command=["cp", ctx.file.src.path, ctx.outputs.dst.path],
    )

copy = rule(
    attrs = {
        "src": attr.label(
            single_file = True,
            allow_files = True,
            mandatory = True,
        ),
        "dst": attr.output(),
    },
    implementation = _copy_impl,
    executable = False,
)

def _copy_to_impl(ctx):
    filtered = []
    if not ctx.attr.matching:
        filtered = ctx.files.srcs
    else:
        for src in ctx.files.srcs:
            if src.basename in ctx.attr.matching:
                filtered += [src]
    outs = depset()
    for src in filtered:
        dst = ctx.new_file(ctx.bin_dir, ctx.attr.to + "/" + src.basename)
        outs += [dst]
        ctx.action(
            inputs=[src],
            outputs=[dst],
            command=["cp", src.path, dst.path],
        )
    return struct(
        files= outs,
    )

copy_to = rule(
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
        "matching": attr.string_list(),
        "to": attr.string(
            mandatory=True,
        ),
    },
    implementation = _copy_to_impl,
)


def _copy_tree_impl(ctx):
    outs = depset()
    for src in ctx.files.srcs:
        path = src.path
        if path.startswith(ctx.attr.strip):
            path = path[len(ctx.attr.strip):]
        if ctx.attr.to:
            path = ctx.attr.to + "/" + path
        dst = ctx.new_file(ctx.bin_dir, path)
        outs += [dst]
        ctx.action(
            inputs=[src],
            outputs=[dst],
            command=["cp", src.path, dst.path],
        )
    return struct(
        files= outs,
    )

copy_tree = rule(
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
        "strip": attr.string(),
        "to": attr.string(),
    },
    implementation = _copy_tree_impl,
)

def copy_platform_binaries(name, src, **kwargs):
    copy_to(
        name = "linux_"+name,
        srcs = [src],
        to = "linux/x86_64",
        tags = ["manual"],
    )
    copy_to(
        name = "osx_"+name,
        srcs = [src],
        to = "osx/x86_64",
        tags = ["manual"],
    )
    copy_to(
        name = "windows_"+name,
        srcs = [src],
        to = "windows/x86_64",
        tags = ["manual"],
    )
    native.filegroup(
        name = name,
        srcs = select({
            "//tools/build:linux": [":linux_"+name],
            "//tools/build:darwin": [":osx_"+name],
            "//tools/build:windows": [":windows_"+name],
        }),
        **kwargs
    )

