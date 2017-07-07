def _filehash_impl(ctx):
    args = [
        "-in", ctx.file.template.path,
        "-replace", ctx.attr.replace,
        "-out", ctx.outputs.out.path,
    ]
    args += [f.path for f in ctx.files.srcs]
    ctx.action(
        inputs=ctx.files.srcs + [ctx.file.template],
        outputs=[ctx.outputs.out],
        arguments=args,
        progress_message="Hashing files into %s" % ctx.outputs.out.short_path,
        executable=ctx.executable._filehash)

filehash = rule(
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
        "template": attr.label(
            mandatory = True,
            single_file = True,
            allow_files = True,
        ),
        "replace": attr.string(mandatory = True),
        "out": attr.output(mandatory = True),
        "_filehash": attr.label(
            cfg = "host",
            executable = True,
            allow_files = True,
            default = Label("//cmd/filehash:filehash"),
        ),
    },
    output_to_genfiles = True,
    implementation = _filehash_impl,
)
