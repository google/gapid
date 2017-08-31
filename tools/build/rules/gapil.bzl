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
    _api_library_impl,
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
    _api_template_impl,
    attrs = {
        "template": attr.label(
            single_file = True,
            allow_files = True,
        ),
        "includes": attr.label_list(allow_files = True),
        "outputs": attr.string_list(),
    },
)
