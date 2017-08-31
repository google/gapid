def _embed_impl(ctx):
    out = ctx.new_file(ctx.label.name + ".go")
    args = ["--out", out.path]
    if ctx.attr.package:
        args += ["--package", ctx.attr.package]
    if ctx.attr.root:
        args += ["--root", ctx.attr.root]
    args += [f.path for f in ctx.files.srcs]
    ctx.action(
        inputs=ctx.files.srcs,
        outputs=[out],
        arguments=args,
        progress_message="Embedding into %s" % out.short_path,
        executable=ctx.executable._embedder,
    )
    return [DefaultInfo(files = depset([out]))]

"""Builds a go embedding file from a collection of data files"""

embed = rule(
    _embed_impl,
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
)
