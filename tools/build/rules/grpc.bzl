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

# ================================================================================
# TODO - Update to latest grpc-java, build it from source and use it's .bzl rules.
# ================================================================================

load(":common.bzl", "copy")

def _path_ignoring_repository(f):
    if (len(f.owner.workspace_root) == 0):
        return f.short_path
    return f.path[f.path.find(f.owner.workspace_root)+len(f.owner.workspace_root)+1:]

def _gensource_impl(ctx):
    # Use .jar since .srcjar makes protoc think output will be a directory
    srcdotjar = ctx.new_file(ctx.label.name + "-src.jar")
    srcs = [f for dep in ctx.attr.srcs for f in dep.proto.direct_sources]
    includes = [f for dep in ctx.attr.srcs for f in dep.proto.transitive_imports]

    ctx.actions.run(
        inputs = [ctx.executable._java_plugin] + srcs + includes,
        outputs = [srcdotjar],
        executable = ctx.executable._protoc,
        arguments = [
            "--plugin=protoc-gen-grpc-java=" + ctx.executable._java_plugin.path,
            "--grpc-java_out=" + srcdotjar.path
        ] + [
            "-I{0}={1}".format(_path_ignoring_repository(include), include.path) for include in includes
        ] + [src.short_path for src in srcs],
        use_default_shell_env = True,
    )

    ctx.actions.run_shell(
        command = "cp \"" + srcdotjar.path + "\" \"" + ctx.outputs.srcjar.path + "\"",
        inputs = [srcdotjar],
        outputs = [ctx.outputs.srcjar],
    )

_gensource = rule(
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
        "_java_plugin": attr.label(
            default = Label("@com_github_grpc_java//:protoc-gen-java"),
            executable = True,
            cfg = "host",
        ),
    },
    outputs = {
        "srcjar": "%{name}.srcjar",
    },
    implementation = _gensource_impl,
)

def java_grpc_library(name, srcs, **kwargs):
  _gensource(
      name = name + "-src",
      srcs = srcs,
      visibility = ["//visibility:private"]
  )

  native.java_library(
      name = name,
      srcs = [name + "-src"],
      **kwargs
  )
