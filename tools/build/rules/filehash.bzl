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

def _filehash_impl(ctx):
    args = [
        "-in", ctx.file.template.path,
        "-replace", ctx.attr.replace,
        "-out", ctx.outputs.out.path,
    ]
    args += [f.path for f in ctx.files.srcs]
    ctx.actions.run(
        inputs = ctx.files.srcs + [ctx.file.template],
        outputs = [ctx.outputs.out],
        arguments = args,
        progress_message = "Hashing files into %s" % ctx.outputs.out.short_path,
        executable = ctx.executable._filehash,
        use_default_shell_env = True,
    )

filehash = rule(
    _filehash_impl,
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
        "template": attr.label(
            mandatory = True,
            allow_single_file = True,
        ),
        "replace": attr.string(mandatory = True),
        "out": attr.output(mandatory = True),
        "_filehash": attr.label(
            cfg = "host",
            executable = True,
            allow_files = True,
            default = Label("//cmd/filehash:filehash"),
        ),
    },
    output_to_genfiles = True,
)
