def _lingo_impl(ctx):
    ctx.action(
        inputs=ctx.files.srcs,
        outputs=ctx.outputs.outs,
        arguments=["-base", ctx.outputs.outs[0].dirname] + [f.path for f in ctx.files.srcs],
        progress_message="Lingo",
        executable=ctx.executable._lingo)

"""Builds a lingo source converter rule"""

_lingo = rule(
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "outs": attr.output_list(),
        "_lingo": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/lingo:lingo"),
        ),
    },
    output_to_genfiles = True,
    implementation = _lingo_impl,
)

def lingo(srcs, name="lingo"):
    _lingo(
        name=name,
        srcs=srcs,
        outs=[src[:-6]+".go" for src in srcs],
        )
