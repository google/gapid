load(":common.bzl", "copy")

def _filter_headers_impl(ctx):
    outs = depset()
    for src in ctx.files.srcs:
        path = src.short_path
        if path.endswith(".h") or  path.endswith(".inl"):
            outs += [src]
    return struct(
        files = outs,
    )

filter_headers = rule(
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
    },
    implementation = _filter_headers_impl,
)

def mm_library(name, srcs=[], hdrs = [], copy_hdrs=[], **kwargs):
    copied_srcs = []
    for src in srcs:
        copy_name = name+"_copy_"+src
        copied_srcs += [":"+copy_name]
        copy(
            name = copy_name,
            src = src,
            dst = name+"/"+src+".cc",
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