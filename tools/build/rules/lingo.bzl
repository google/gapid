def _lingo_impl(ctx):
    outs = [ctx.new_file(src.basename[:-6]+".go") for src in ctx.files.srcs]
    ctx.action(
        inputs=ctx.files.srcs,
        outputs=outs,
        arguments=["-base", outs[0].dirname] + [f.path for f in ctx.files.srcs],
        progress_message="Lingo",
        executable=ctx.executable._lingo,
    )

"""Builds a lingo source converter rule"""

lingo = rule(
    _lingo_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "_lingo": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/lingo:lingo"),
        ),
    },
)
