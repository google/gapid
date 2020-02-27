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

def _generate_impl(ctx):
    ctx.file_action(
        output = ctx.outputs.output,
        content = ctx.attr.content,
    )

generate = rule(
    _generate_impl,
    attrs = {
        "output" : attr.output(mandatory=True),
        "content" : attr.string(mandatory=True),
    },
)

def _copy(ctx, src, dst):
    ctx.actions.run_shell(
        command = "cp \"" + src.path + "\" \"" + dst.path + "\"",
        inputs = [src],
        outputs = [dst]
    )

def _copy_impl(ctx):
    _copy(ctx, ctx.file.src, ctx.outputs.dst)

copy = rule(
    _copy_impl,
    attrs = {
        "src": attr.label(
            allow_single_file = True,
            mandatory = True,
        ),
        "dst": attr.output(),
    },
    executable = False,
)

def _copy_to_impl(ctx):
    filtered = []
    if not ctx.attr.extensions:
        filtered = ctx.files.srcs
    else:
        for src in ctx.files.srcs:
            if src.extension in ctx.attr.extensions:
                filtered += [src]
    outs = []
    for src in filtered:
        dstname = ctx.attr.rename.get(src.basename, default = src.basename)
        dst = ctx.actions.declare_file(ctx.attr.to + "/" + dstname)
        outs += [dst]
        _copy(ctx, src, dst)

    return struct(
        files = depset(outs),
    )

copy_to = rule(
    _copy_to_impl,
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
        "extensions": attr.string_list(),
        "rename": attr.string_dict(),
        "to": attr.string(
            mandatory=True,
        ),
    },
)

def _dirname_ignoring_repository(f):
    if (len(f.owner.workspace_root) == 0):
        return f.dirname
    return f.dirname[f.dirname.find(f.owner.workspace_root)+len(f.owner.workspace_root)+1:]

def _copy_tree_impl(ctx):
    outs = []
    for src in ctx.files.srcs:
        dir = _dirname_ignoring_repository(src)
        if dir.startswith(ctx.attr.strip):
            dir = dir[len(ctx.attr.strip):]
        if ctx.attr.to:
            dir = ctx.attr.to + "/" + dir
        path = dir + "/" + ctx.attr.rename.get(src.basename, default = src.basename)
        dst = ctx.actions.declare_file(path)
        outs += [dst]
        _copy(ctx, src, dst)

    return struct(
        files = depset(outs),
    )

copy_tree = rule(
    _copy_tree_impl,
    attrs = {
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
        ),
        "strip": attr.string(),
        "rename": attr.string_dict(),
        "to": attr.string(),
    },
)

# Implementation of the copy_exec rule.
def _copy_exec_impl(ctx):
    if len(ctx.files.srcs) != 1:
        fail("copy_exec rule with multiple inputs")

    src = ctx.files.srcs[0]
    extension = src.extension
    if not extension == "":
        extension = "." + extension
    if ctx.label.name.endswith(extension):
        extension = ""
    out = ctx.actions.declare_file(ctx.label.name + extension)

    _copy(ctx, src, out)

    return struct(
        files = depset([out]),
        runfiles = ctx.runfiles(collect_data = True),
        executable = out,
    )

# A copy rule that can be used on an executable. Propagates the extension and
# runfiles to the output.
copy_exec = rule(
    _copy_exec_impl,
    attrs = {
        # Needs to be a list and called srcs, so collect_data above will work.
        # If more than one file is given, the rule will fail.
        "srcs": attr.label_list(
            allow_files = True,
            mandatory = True,
            allow_empty = False,
        ),
    },
    executable = True,
)
