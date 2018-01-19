load(":common.bzl", "copy")

def _filter_headers_impl(ctx):
    outs = depset()
    for src in ctx.files.srcs:
        path = src.short_path
        if path.endswith(".h") or path.endswith(".inc"):
            outs += [src]
    return struct(
        files = outs,
    )

filter_headers = rule(
    _filter_headers_impl,
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
    },
)

def mm_library(name, srcs=[], hdrs = [], copy_hdrs=[], **kwargs):
    # bazel doesn't care for .mm files in the srcs attribute of C++ targets.
    # Copy all the .mm files to .mm.cc files and compile them with the C++
    # compiler, but passing the "-x objective-c++" language selection argument.
    copied_srcs = []
    for src in srcs:
        copy_name = name+"_copy_"+src
        copied_srcs += [":"+copy_name]
        if src.endswith(".mm"):
            copy(
                name = copy_name,
                src = src,
                dst = name+"/"+src+".cc",
            )
        else:
            copy(
                name = copy_name,
                src = src,
                dst = name+"/"+src,
            )
    filter_headers(
        name = name+"_headers",
        srcs = hdrs,
    )
    index = 0
    use_headers = [":"+name+"_headers"]
    for hdr in copy_hdrs:
        index+=1
        copied = "hdr_"+hdr
        use_headers += [":"+copied]
        copy(
            name = copied,
            src = hdr,
            dst = name+"/"+hdr,
        )
    native.cc_library(
        name = name,
        tags = ["manual"],
        srcs = copied_srcs,
        hdrs = use_headers,
        copts = ["-x","objective-c++"],
        **kwargs
    )
