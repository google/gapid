# Copyright (C) 2018 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load(":common.bzl", "copy")

def _filter_headers_impl(ctx):
    outs = []
    for src in ctx.files.srcs:
        path = src.short_path
        if path.endswith(".h") or path.endswith(".inc"):
            outs += [src]
    return struct(
        files = depset(outs),
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

def mm_library(name, srcs = [], hdrs = [], copy_hdrs = [], copts = [], **kwargs):
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
    use_headers = [":"+name+"_headers"]
    for hdr in copy_hdrs:
        copied = "hdr_"+hdr
        use_headers += [":"+copied]
        copy(
            name = copied,
            src = hdr,
            dst = name+"/"+hdr,
        )
    native.cc_library(
        name = name,
        srcs = copied_srcs,
        hdrs = use_headers,
        copts = ["-x","objective-c++"] + copts,
        **kwargs
    )
