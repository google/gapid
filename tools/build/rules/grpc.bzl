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

def _gen_java_source_impl(ctx):
  # Use .jar since .srcjar makes protoc think output will be a directory
  srcdotjar = ctx.actions.declare_file(ctx.label.name + "-src.jar")
  protos = [f for dep in ctx.attr.srcs for f in dep.proto.direct_sources]
  includes = [f for dep in ctx.attr.srcs for f in dep.proto.transitive_imports]

  arguments = [
    "--plugin=protoc-gen-grpc-java=" + ctx.executable._plugin.path,
    "--grpc-java_out=" + srcdotjar.path,
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
    outputs = [srcdotjar],
    tools = [ctx.executable._protoc, ctx.executable._plugin],
    executable = ctx.executable._protoc,
    arguments = arguments,
    use_default_shell_env = True,
  )

  ctx.actions.run_shell(
    command = "cp \"" + srcdotjar.path + "\" \"" + ctx.outputs.srcjar.path + "\"",
    inputs = [srcdotjar],
    outputs = [ctx.outputs.srcjar],
  )

_gen_java_source = rule(
  attrs = {
    "srcs": attr.label_list(
      mandatory = True,
      non_empty = True,
      providers = ["proto"],
    ),
    "_protoc": attr.label(
      default = Label("@com_google_protobuf//:protoc"),
      executable = True,
      cfg = "host",
    ),
    "_plugin": attr.label(
      default = Label("@com_github_grpc_java//:protoc-gen-java"),
      executable = True,
      cfg = "host",
    ),
  },
  outputs = {
    "srcjar": "%{name}.srcjar",
  },
  implementation = _gen_java_source_impl,
)

def _gen_cc_source_impl(ctx):
  protos = [f for dep in ctx.attr.srcs for f in dep.proto.direct_sources]
  includes = [f for dep in ctx.attr.srcs for f in dep.proto.transitive_imports]

  proto_root = ""
  if ctx.label.workspace_root:
    proto_root = "/" + ctx.label.workspace_root

  out_files = []
  out_files += [ctx.actions.declare_file(
      proto.basename[:-len(".proto")] + ".pb.h",
      sibling = proto) for proto in protos]
  out_files += [ctx.actions.declare_file(
      proto.basename[:-len(".proto")] + ".pb.cc",
      sibling = proto) for proto in protos]
  out_files += [ctx.actions.declare_file(
      proto.basename[:-len(".proto")] + ".grpc.pb.h",
      sibling = proto) for proto in protos]
  out_files += [ctx.actions.declare_file(
      proto.basename[:-len(".proto")] + ".grpc.pb.cc",
      sibling = proto) for proto in protos]
  dir_out = str(ctx.genfiles_dir.path + proto_root)

  arguments = [
    "--cpp_out=" + dir_out,
    "--plugin=protoc-gen-plugin=" + ctx.executable._plugin.path,
    "--plugin_out=" + dir_out,
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

  ctx.action(
    inputs = protos + includes,
    outputs = out_files,
    tools = [ctx.executable._protoc, ctx.executable._plugin],
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

_gen_cc_source = rule(
  attrs = {
    "srcs": attr.label_list(
      mandatory = True,
      non_empty = True,
      providers = ["proto"],
    ),
    "_protoc": attr.label(
      default = Label("@com_google_protobuf//:protoc"),
      executable = True,
      cfg = "host",
    ),
    "_plugin": attr.label(
      default = Label("@com_github_grpc_grpc//:grpc_cpp_plugin"),
      executable = True,
      cfg = "host",
    ),
  },
  implementation = _gen_cc_source_impl,
)

def java_grpc_library(name, srcs, **kwargs):
  _gen_java_source(
    name = name + "-src",
    srcs = srcs,
    visibility = ["//visibility:private"]
  )

  native.java_library(
    name = name,
    srcs = [name + "-src"],
    **kwargs
  )


def cc_grpc_library(name, srcs, **kwargs):
  _gen_cc_source(
    name = name + "_src",
    srcs = srcs,
    visibility = ["//visibility:private"]
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
