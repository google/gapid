def _stringgen_impl(ctx):
    go = ctx.new_file(ctx.label.name+".go")
    api = ctx.new_file(ctx.label.name+".api")
    table = ctx.new_file("en-us.stb")
    ctx.actions.run(
        inputs = [ctx.file.input],
        outputs = [go, api, table],
        progress_message = "Stringgen %s" % ctx.file.input.short_path,
        executable = ctx.executable._stringgen,
        arguments = [
            "--def-go", go.path,
            "--def-api", api.path,
            "--pkg", table.dirname,
            ctx.file.input.path
        ],
        use_default_shell_env = True,
    )
    return [DefaultInfo(files=depset([go, api, table]))]

"""Builds a stringgen source converter rule"""
stringgen = rule(
    _stringgen_impl,
    attrs = {
        "input": attr.label(
            single_file = True,
            allow_files = True,
            mandatory = True,
        ),
        "_stringgen": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/stringgen:stringgen"),
        ),
    },
)
