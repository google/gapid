def _go_proto_sources_impl(ctx):
    sources = []
    transitive_sources = depset()
    for dep in ctx.attr.deps:
        sources += dep.proto.direct_sources
        transitive_sources += dep.proto.transitive_sources
    all_sources = transitive_sources.to_list()
    options = []
    if ctx.attr.grpc:
        options += ["plugins=grpc"]
    for src in all_sources:
        proto = src.short_path
        if "google/protobuf/" in proto:
            ptype = src.basename[:-len(".proto")]
            options += ["M{0}=github.com/golang/protobuf/ptypes/{1}".format(proto, ptype)]
        else:
            options += ["M{0}={1}/{2}".format(src.short_path, ctx.attr._go_prefix.go_prefix, src.dirname)]
    go_files = []
    for src in sources:
        prefix = ctx.label.package + "/"
        if not src.short_path.startswith(prefix):
            fail("Source {0} not in path {1}", src.short_path, prefix)
        relative_path = src.short_path[len(prefix):]
        go_files += [ctx.new_file(relative_path[:-len(".proto")] + ".pb.go")]
    ctx.action(
        executable=ctx.executable._protoc,
        mnemonic="protoc",
        progress_message="Generating .pb.go for " + ctx.label.name,
        inputs= [ctx.executable._protoc, ctx.executable._protoc_gen_go] + all_sources,
        outputs=go_files,
        arguments= [
            "--go_out=" + ",".join(options) + ":" + ctx.var["GENDIR"],
            "--plugin=protoc-gen-go=" + ctx.executable._protoc_gen_go.path,
        ] + 
            ["-I" + s.short_path + "=" + s.path for s in all_sources] +
            #["--proto_path="+path for path in paths] +
            [s.path for s in sources],
    )
    return struct(
        files=depset(go_files),
    )

go_proto_sources = rule(
    attrs = {
        "deps": attr.label_list(
            allow_files = False,
            providers = [
                "proto",
            ],
        ),
        "grpc": attr.bool(),
        "_protoc": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("@com_google_protobuf//:protoc"),
        ),
        "_protoc_gen_go": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("@com_github_golang_protobuf//protoc-gen-go:protoc-gen-go"),
        ),
        "_go_prefix": attr.label(default = Label("//:go_prefix")),
    },
    output_to_genfiles = True,
    implementation = _go_proto_sources_impl,
)

