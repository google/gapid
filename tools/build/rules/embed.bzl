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

def _embed_impl(ctx):
    out = ctx.actions.declare_file(ctx.label.name + ".go")
    args = ["--out", out.path]
    if ctx.attr.package:
        args += ["--package", ctx.attr.package]
    if ctx.attr.root:
        args += ["--root", ctx.attr.root]
    args += [f.path for f in ctx.files.srcs]
    ctx.actions.run(
        inputs = ctx.files.srcs,
        outputs = [out],
        arguments = args,
        progress_message = "Embedding into %s" % out.short_path,
        executable = ctx.executable._embedder,
        use_default_shell_env = True,
    )
    return [DefaultInfo(files = depset([out]))]

"""Builds a go embedding file from a collection of data files"""

embed = rule(
    _embed_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "package": attr.string(),
        "root": attr.string(),
        "_embedder": attr.label(
            cfg = "host",
            executable = True,
            allow_files = True,
            default = Label("//cmd/embed:embed"),
        ),
    },
    output_to_genfiles = True,
)
