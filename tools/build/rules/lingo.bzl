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

def _lingo_impl(ctx):
    outs = [ctx.actions.declare_file(src.basename[:-6]+".go") for src in ctx.files.srcs]
    ctx.actions.run(
        inputs = ctx.files.srcs,
        outputs = outs,
        arguments = ["-base", outs[0].dirname] + [f.path for f in ctx.files.srcs],
        progress_message = "Lingo",
        executable = ctx.executable._lingo,
        use_default_shell_env = True,
    )
    return [
        DefaultInfo(
            files=depset(outs),
        ),
    ]

"""Builds a lingo source converter rule"""

lingo = rule(
    _lingo_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "_lingo": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/lingo:lingo"),
        ),
    },
    output_to_genfiles = True,
)
