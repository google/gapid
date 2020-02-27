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

def _extract_impl(ctx):
    outs = []
    base = ""
    if ctx.attr.dir:
        base = ctx.attr.dir + "/"
    for entry in ctx.attr.entries:
        out = ctx.actions.declare_file(base + entry)
        to =  out.path[:-len(entry)]
        outs += [out]
        ctx.actions.run_shell(
            inputs = [ctx.file.zip],
            outputs = [out],
            command = "unzip -q -DD -d {} {} {}".format(to, ctx.file.zip.path, entry),
        )
    return struct(
        files = depset(outs)
    )

extract = rule(
    _extract_impl,
    attrs = {
        "zip": attr.label(
            allow_single_file = True,
            mandatory = True,
        ),
        "entries": attr.string_list(mandatory=True),
        "dir": attr.string(),
    },
)
