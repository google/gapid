def _codergen_impl(ctx):
    intermediates = []
    outpath = ctx.outputs.signature.dirname
    for f in ctx.files.deps:
        output = ctx.new_file("src/github.com/google/gapid/" + f.short_path)
        intermediates.append(output)
        ctx.action(
            inputs = [f],
            outputs = [output],
            command = ["cp", f.path, output.path],
        )
    base = outpath+"/src/github.com/google/gapid"
    ctx.action(
        inputs=ctx.files.deps + intermediates,
        outputs=[ctx.outputs.signature] + ctx.outputs.generates,
        arguments=[
            "--gopath", outpath,
            "--base", base,
            "--signatures", ctx.outputs.signature.path,
            "-go",
            "-cpp", base + "/core/cc/coder",
            "-java", base + "/gapic/src",
            "./...",
        ],
        progress_message="Running codergen",
        executable=ctx.executable._codergen,
    )

_codergen = rule(
    attrs = {
        "deps": attr.label_list(allow_files = True),
        "signature": attr.output(),
        "generates": attr.output_list(),
        "_codergen": attr.label(
            cfg = "host",
            executable = True,
            allow_files = True,
            default = Label("//cmd/codergen:codergen"),
        ),
    },
    output_to_genfiles = True,
    implementation = _codergen_impl,
)

def codergen(name, reads, generates, signature):
    paths = []
    outputs = []
    for gen in generates:
        pkg = gen.rsplit("/", 1)[0]
        if "/cc/" not in pkg and "gapic/src" not in pkg:
            paths += [pkg]
        fulloutput = "src/github.com/google/gapid/{0}".format(gen)
        outputs += [fulloutput]
        native.filegroup(
            name = gen,
            srcs = [fulloutput],
            visibility = ["//visibility:public"],
        )
    _codergen(
        name=name,
        deps=["//%s:codergen_inputs"%p for p in paths+reads],
        signature=signature,
        generates = outputs,
    )
