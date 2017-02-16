load("@//tools/build:rules.bzl", "copy_to", "cc_copts")

def llvm_sources(name, exclude=[]):
    return native.glob([
            name+"/*.c",
            name+"/*.cpp",
            name+"/*.h",
            name+"/*.inc",
        ], exclude=exclude) + select({
            "@//tools/build:windows": native.glob([
                name+"/Windows/*.h",
                name+"/Windows/*.inc",
            ], exclude=exclude),
            "//conditions:default": native.glob([
                name+"/Unix/*.h",
                name+"/Unix/*.inc",
            ], exclude=exclude),
        })

def llvmLibrary(name, path="", deps=[], excludes={}, extras={}):
    exclude = []
    if name in extras:
        deps += extras[name]
    if name in excludes:
        exclude += excludes[name]
    native.cc_library(
        name = name,
        srcs =  llvm_sources(path, exclude=exclude),
        deps = deps,
        copts = cc_copts(),
    )

def _tablegen_impl(ctx):
    include = ctx.files.deps[0].path.split("/include/")[0]+"/include"
    args = ctx.attr.flags + [
        "-I", include,
        "-I", ctx.file.table.dirname,
        "-o", ctx.outputs.generate.path, 
        ctx.file.table.path,
    ]
    ctx.action(
        inputs=[ctx.file.table] + ctx.files.deps,
        outputs=[ctx.outputs.generate],
        arguments=args,
        progress_message="Table gen into %s" % ctx.outputs.generate,
        executable=ctx.executable._tblgen)

llvm_tablegen = rule(
    attrs = {
        "table": attr.label(
            single_file = True,
            allow_files = True,
        ),
        "deps": attr.label_list(allow_files = True),
        "generate": attr.output(),
        "flags": attr.string_list(),
        "_tblgen": attr.label(
            cfg = "host",
            executable = True,
            allow_files = True,
            default = Label("@llvm//:llvm-tblgen"),
        ),
    },
    output_to_genfiles = True,
    implementation = _tablegen_impl,
)

def tablegen(name = "", table="", rules=[], deps=[], **kwargs):
    targets = []
    for entry in rules:
        generate = entry[0]
        flags = entry[1:]
        gen = name + "-" + generate.rsplit("/", 1)[-1]
        targets += [":"+gen]
        llvm_tablegen(
            name=gen,
            table=table,
            deps=deps,
            generate=generate,
            flags=flags,
        )
    native.cc_library(
        name = name,
        hdrs=targets,
        linkstatic=1,
        **kwargs
    )
