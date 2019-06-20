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

def _stringgen_impl(ctx):
    go = ctx.actions.declare_file(ctx.label.name+".go")
    api = ctx.actions.declare_file(ctx.label.name+".api")
    table = ctx.actions.declare_file("en-us.stb")
    ctx.actions.run(
        inputs = [ctx.file.input],
        outputs = [go, api, table],
        progress_message = "Stringgen %s" % ctx.file.input.short_path,
        executable = ctx.executable._stringgen,
        arguments = [
            "--def-go", go.path,
            "--def-api", api.path,
            "--pkg", table.dirname,
            ctx.file.input.path
        ],
        use_default_shell_env = True,
    )
    return [
        DefaultInfo(files = depset([go, api, table])),
        OutputGroupInfo(
            go = depset([go]),
            api = depset([api]),
            table = depset([table]),
        )
    ]

"""Builds a stringgen source converter rule"""
stringgen = rule(
    _stringgen_impl,
    attrs = {
        "input": attr.label(
            allow_single_file = True,
            mandatory = True,
        ),
        "_stringgen": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/stringgen:stringgen"),
        ),
    },
    output_to_genfiles = True,
)
