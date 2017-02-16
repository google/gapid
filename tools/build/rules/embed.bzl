def _embed_impl(ctx):
    args = ["--out", ctx.outputs.out.path]
    if ctx.attr.package:
        args += ["--package", ctx.attr.package]
    if ctx.attr.root:
        args += ["--root", ctx.attr.root]
    args += [f.path for f in ctx.files.srcs]
    ctx.action(
        inputs=ctx.files.srcs,
        outputs=[ctx.outputs.out],
        arguments=args,
        progress_message="Embedding into %s" % ctx.outputs.out.short_path,
        executable=ctx.executable._embedder)

"""Builds a go embedding file from a collection of data files"""

embed = rule(
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "package": attr.string(),
        "root": attr.string(),
        "_embedder": attr.label(
            cfg = "host",
            executable = True,
            allow_files = True,
            default = Label("//cmd/embed:embed"),
        ),
    },
    output_to_genfiles = True,
    outputs = {"out": "%{name}.go"},
    implementation = _embed_impl,
)
