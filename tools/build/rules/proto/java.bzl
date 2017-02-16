load("//tools/build/rules:common.bzl", "copy")

def _java_proto_grpc_rule_impl(ctx):
    sources = []
    transitive_sources = depset()
    for dep in ctx.attr.deps:
        sources += dep.proto.direct_sources
        transitive_sources += dep.proto.transitive_sources
    all_sources = transitive_sources.to_list()
    ctx.action(
        executable=ctx.executable._protoc,
        mnemonic="protoc",
        progress_message="Generating " + ctx.outputs.jar.path,
        inputs= [ctx.executable._protoc, ctx.executable._protoc_gen_java] + all_sources,
        outputs=[ctx.outputs.jar],
        arguments= [
            "--plugin=protoc-gen-grpc-java=" + ctx.executable._protoc_gen_java.path,
            "--java_out="+ctx.outputs.jar.path,
            "--grpc-java_out="+ctx.outputs.jar.path,
        ] + 
            ["-I" + s.short_path + "=" + s.path for s in all_sources] +
            #["--proto_path="+path for path in paths] +
            [s.path for s in sources],
    )

_java_proto_grpc_rule = rule(
    attrs = {
        "deps": attr.label_list(
            allow_files = False,
            providers = [
                "proto",
            ],
        ),
        "jar": attr.output(),
        "_protoc": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("@com_google_protobuf//:protoc"),
        ),
        "_protoc_gen_java": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("@com_github_grpc_java//:protoc-gen-java"),
        ),
    },
    output_to_genfiles = True,
    implementation = _java_proto_grpc_rule_impl,
)

def java_proto_library(name, grpc=False, deps=[], extra_deps=[], **kwargs):
    if not grpc:
        # Use the built in rules, they do exactly what we need
        native.java_proto_library(name=name, deps=deps+extra_deps, **kwargs)
        return
    # Hoop jumping time, native rules don't do grpc
    protocjar = name+".pbsrc.jar"
    srcjar = name+".srcjar"
    # First we make a jar, because that's what protoc can do
    _java_proto_grpc_rule(
        name = name + "_protoc",
        deps = deps,
        jar = protocjar, 
    )
    # Now we rename the jar to srcjar, because that's what bazel needs
    copy(
        name = name+"_copy",
        src = protocjar,
        dst = srcjar,
    )
    # And then we make a compiled jar from that jar
    native.java_library(
        name=name, 
        srcs=[srcjar], 
        deps=[
            "//gapic/third_party:grpc",
            "//gapic/third_party:guava",
            "@com_google_protobuf//:protobuf_java",
        ] + extra_deps,
        **kwargs
    )