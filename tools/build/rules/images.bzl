# Copyright (C) 2020 Google Inc.
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

def _img2h_impl(ctx):
    outs = []
    for img in ctx.files.srcs:
        n = img.basename
        out = ctx.actions.declare_file(n[0:len(n) - len(img.extension)] + "h")
        ctx.actions.run(
            inputs = [img],
            outputs = [out],
            arguments = ["-out", out.path, img.path],
            executable = ctx.executable._img2h,
            use_default_shell_env = True,
        )
        outs += [out]
    return [
       DefaultInfo(files = depset(outs)),
    ]

img2h = rule(
    _img2h_impl,
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
        "_img2h": attr.label(
            cfg = "host",
            executable = True,
            allow_files = True,
            default = Label("//cmd/img2h"),
        ),
    },
    output_to_genfiles = True,
)
