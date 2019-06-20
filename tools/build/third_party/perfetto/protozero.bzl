# Copyright (C) 2019 Google Inc.
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

def _path_ignoring_repository(f):
    if (len(f.owner.workspace_root) == 0):
        return f.short_path
    return f.path[f.path.find(f.owner.workspace_root)+len(f.owner.workspace_root)+1:]

def _gen_cc_impl(ctx):
  protos = [f for dep in ctx.attr.deps for f in dep.proto.direct_sources]
  includes = [f for dep in ctx.attr.deps for f in dep.proto.transitive_imports]
  proto_root = ""
  if ctx.label.workspace_root:
    proto_root = "/" + ctx.label.workspace_root

  out_files = []
  out_files += [ctx.actions.declare_file(
      proto.basename[:-len(".proto")] + ".pbzero.h",
      sibling = proto) for proto in protos]
  out_files += [ctx.actions.declare_file(
      proto.basename[:-len(".proto")] + ".pbzero.cc",
      sibling = proto) for proto in protos]
  dir_out = str(ctx.genfiles_dir.path + proto_root)

  arguments = [
    "--plugin=protoc-gen-plugin=" + ctx.executable._plugin.path,
    "--plugin_out=wrapper_namespace=pbzero:" + dir_out
  ]

  for include in includes:
    directory = include.path
    if directory.startswith("external"):
      external_sep = directory.find("/")
      repository_sep = directory.find("/", external_sep + 1)
      arguments += ["--proto_path=" + directory[:repository_sep]]
    else:
      arguments += ["--proto_path=."]
  arguments += [proto.path for proto in protos]

  ctx.actions.run(
    inputs = protos + includes,
    outputs = out_files,
    tools =  [ctx.executable._protoc, ctx.executable._plugin],
    executable = ctx.executable._protoc,
    arguments = arguments,
    use_default_shell_env = True,
  )

  return [
    DefaultInfo(files = depset(out_files)),
    OutputGroupInfo(
      cc = depset([f for f in out_files if f.path.endswith(".cc")]),
      h = depset([f for f in out_files if f.path.endswith(".h")]),
    ),
  ]

_gen_cc = rule(
    attrs = {
        "deps": attr.label_list(
            mandatory = True,
            allow_empty = False,
            providers = ["proto"],
        ),
        "_protoc": attr.label(
            default = Label("@com_google_protobuf//:protoc"),
            executable = True,
            cfg = "host",
        ),
        "_plugin": attr.label(
            default = Label("@perfetto//:protozero_plugin"),
            executable = True,
            cfg = "host",
        ),

    },
    output_to_genfiles = True,
    implementation = _gen_cc_impl,
)

def cc_protozero_library(name, deps, **kwargs):
  _gen_cc(
    name = name + "_src",
    deps = deps,
  )

  native.filegroup(
    name = name + "_h",
    srcs = [":" + name + "_src"],
    output_group = "h",
  )

  native.cc_library(
    name = name,
    srcs = [":" + name + "_src"],
    hdrs = [":" + name + "_h"],
    **kwargs
  )
