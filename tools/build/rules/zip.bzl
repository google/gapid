def _extract_impl(ctx):
    outs = depset()
    base = ""
    if ctx.attr.dir:
        base = ctx.attr.dir + "/"
    for entry in ctx.attr.entries:
        out = ctx.new_file(ctx.bin_dir, base + entry)
        to =  out.path[:-len(entry)]
        outs += [out]
        ctx.action(
            inputs=[ctx.file.zip],
            outputs=[out],
            command=["unzip", "-d", to, ctx.file.zip.path, entry],
        )
    return struct(
        files = outs
    )

extract = rule(
    attrs = {
        "zip": attr.label(
            single_file = True,
            allow_files = True,
            mandatory = True,
        ),
        "entries": attr.string_list(mandatory=True),
        "dir": attr.string(),
    },
    implementation = _extract_impl,
)
