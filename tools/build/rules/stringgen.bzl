def _stringgen_impl(ctx):
    ctx.action(
        inputs=[ctx.file.input],
        outputs=[ctx.outputs.go, ctx.outputs.api, ctx.outputs.table],
        progress_message="Stringgen %s" % ctx.file.input.short_path,
        executable=ctx.executable._stringgen,
        arguments=[
            "--def-go", ctx.outputs.go.path,
            "--def-api", ctx.outputs.api.path,
            "--pkg", ctx.outputs.table.dirname,
            ctx.file.input.path
        ],
    )

"""Builds a stringgen source converter rule"""
stringgen = rule(
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
    output_to_genfiles = True,
    outputs = {
        "go": "%{name}.go",
        "api": "%{name}.api",
        "table": "en-us.stb",
    },
    implementation = _stringgen_impl,
)
