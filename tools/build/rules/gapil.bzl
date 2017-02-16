def _api_library_impl(ctx):
    includes = depset()
    includes += [ctx.file.api]
    includes += ctx.files.includes
    for dep in ctx.attr.deps:
        includes += dep.includes
    return struct(
        apiname = ctx.attr.apiname,
        main = ctx.file.api,
        includes = includes,
    )

"""Adds an API library rule"""
api_library = rule(
    attrs = {
        "apiname": attr.string(mandatory=True),
        "api": attr.label(
            single_file = True,
            allow_files = True,
        ),
        "includes": attr.label_list(allow_files = True),
        "deps": attr.label_list(
            allow_files = False,
            providers = [
                "apiname",
                "main",
                "includes",
            ],
        ),
    },
    output_to_genfiles = True,
    implementation = _api_library_impl,
)

def _api_template_impl(ctx):
    uses = depset()
    uses += [ctx.file.template]
    uses += ctx.files.includes
    return struct(
        main = ctx.file.template,
        uses = uses,
        outputs = ctx.attr.outputs,
    )

"""Adds an API template library rule"""
api_template = rule(
    attrs = {
        "template": attr.label(
            single_file = True,
            allow_files = True,
        ),
        "includes": attr.label_list(allow_files = True),
        "outputs": attr.string_list(),
    },
    output_to_genfiles = True,
    implementation = _api_template_impl,
)
