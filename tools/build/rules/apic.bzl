load("@io_bazel_rules_go//go:def.bzl", "GoLibrary")
load("@io_bazel_rules_go//go/private:providers.bzl", "CgoLibrary")

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
        srcs = depset([f for f in generated if f.basename.endswith(".go")])
    return [
        GoLibrary(#TODO: this is too complicated, needs cleaning up
            label = ctx.label,
            srcs = srcs,
            transformed = srcs,
            direct=(),
            cgo_deps=(),
            gc_goopts=(),
            cover_vars=(),
        ),
        CgoLibrary(#TODO: remove this when we switch from the library attribute
            object=None,
        ),
        DefaultInfo(
            files = generated,
        ),
    ]

"""Adds an API compiler rule"""
apic = rule(
    _apic_impl,
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
)
