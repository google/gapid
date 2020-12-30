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
  protos = [f for dep in ctx.attr.srcs for f in dep[ProtoInfo].direct_sources]
  dsi = [f for dep in ctx.attr.srcs for f in dep[ProtoInfo].transitive_descriptor_sets.to_list()]

  args = ctx.actions.args()
  args.add(ctx.executable._plugin, format = "--plugin=protoc-gen-rpc-plugin=%s")
  args.add(srcdotjar, format = "--rpc-plugin_out=%s")
  args.add_joined("--descriptor_set_in", dsi,
      join_with = ctx.host_configuration.host_path_separator,
      uniquify = True,
  )
  args.add_all(protos)

  ctx.actions.run(
    inputs = protos + dsi,
    outputs = [srcdotjar],
    tools = [ctx.executable._protoc, ctx.executable._plugin],
    executable = ctx.executable._protoc,
    arguments = [args],
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
      allow_empty = False,
      providers = [ProtoInfo],
    ),
    "_protoc": attr.label(
      default = Label("@com_google_protobuf//:protoc"),
      executable = True,
      cfg = "host",
    ),
    "_plugin": attr.label(
      default = Label("@com_github_grpc_java//compiler:grpc_java_plugin"),
      executable = True,
      cfg = "host",
    ),
  },
  outputs = {
    "srcjar": "%{name}.srcjar",
  },
  implementation = _gen_java_source_impl,
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
