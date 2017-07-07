load(":embed.bzl", "embed")
load(":apic.bzl", "api_search_path")

def _annotate_impl(ctx):
    inputs = ctx.attr.api.includes.to_list()
    ctx.action(
        inputs=inputs,
        outputs=[
            ctx.outputs.base64, 
            ctx.outputs.globals_base64,
            ctx.outputs.base64_text, 
            ctx.outputs.globals_base64_text,
        ],
        arguments=[
            "--base64", ctx.outputs.base64.path,
            "--text", ctx.outputs.base64_text.path,
            "--globals_base64", ctx.outputs.globals_base64.path,
            "--globals_text", ctx.outputs.globals_base64_text.path,
            "--search", api_search_path(inputs),
            ctx.attr.api.main.path,
        ],
        progress_message="Annotating %s" % ctx.outputs.base64.short_path,
        executable=ctx.executable._annotator)

annotate = rule(
    attrs = {
        "api": attr.label(
            allow_files = False,
            mandatory = True,
            providers = [
                "main",
                "includes",
            ],
        ),
        "_annotator": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/annotate:annotate"),
        ),
    },
    output_to_genfiles = True,
    outputs = {
        "base64": "snippets.base64",
        "base64_text": "snippets.text",
        "globals_base64": "globals_snippets.base64",
        "globals_base64_text": "globals_snippets.text",
    },
    implementation = _annotate_impl,
)

def snippets(name, api, visibility=[]):
    generate = name+"-generate"
    annotate(
        name = generate,
        api = api,
        visibility = ["//visibility:private"],
    )
    embed(
        name = name,
        srcs = [generate],
        visibility = visibility,
    )
