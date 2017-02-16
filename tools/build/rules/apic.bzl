def api_search_path(inputs):
    roots = {}
    for dep in inputs:
        if dep.root.path:
            roots[dep.root.path] = True
    return ",".join(["."] + roots.keys())

def _apic_impl(ctx):
    api = ctx.attr.api
    apiname = api.apiname
    apilist = api.includes.to_list()
    dir = ctx.genfiles_dir
    generated = depset()
    for template in ctx.attr.templates:
        templatelist = template.uses.to_list()
        outputs = [ctx.new_file(dir, out.format(api=apiname)) for out in template.outputs]
        generated += outputs
        ctx.action(
            inputs = apilist + templatelist,
            outputs = outputs,
            arguments = [
                "template",
                "--dir", outputs[0].dirname,
                "--search", api_search_path(apilist),
                api.main.path,
                template.main.path, 
            ],
            mnemonic = "apic",
            progress_message = "apic " + api.main.short_path + " with " + template.main.short_path,
            executable = ctx.executable._apic
        )
    return struct(
        files = generated,
    )

"""Adds an API compiler rule"""
apic = rule(
    attrs = {
        "api": attr.label(
            allow_files = False,
            mandatory = True,
            providers = [
                "apiname",
                "main",
                "includes",
            ],
        ),
        "templates": attr.label_list(
            allow_files = False,
            mandatory = True,
            providers = [
                "main",
                "uses",
                "outputs",
            ],
        ),
        "_apic": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/apic:apic"),
        ),
    },
    output_to_genfiles = True,
    implementation = _apic_impl,
)
