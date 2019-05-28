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

load("@io_bazel_rules_go//go:def.bzl",
    "go_context",
    "GoLibrary",
)

ApicTemplate = provider()

def _api_library_impl(ctx):
    includes = depset(
        [ctx.file.api] + ctx.files.includes,
        transitive = [dep.includes for dep in ctx.attr.deps])
    return struct(
        apiname = ctx.attr.apiname,
        main = ctx.file.api,
        includes = includes,
    )

"""Adds an API library rule"""
api_library = rule(
    _api_library_impl,
    attrs = {
        "apiname": attr.string(mandatory=True),
        "api": attr.label(
            single_file = True,
            allow_files = True,
        ),
        "includes": attr.label_list(allow_files = True),
        "deps": attr.label_list(
            allow_files = False,
            providers = [
                "apiname",
                "main",
                "includes",
            ],
        ),
    },
)

def _api_template_impl(ctx):
    go = go_context(ctx)
    library = go.new_library(go)
    return [
        library,
        go.library_to_source(go, ctx.attr, library, False),
        ApicTemplate(
            main = ctx.file.template,
            uses = depset([ctx.file.template] + ctx.files.includes),
            outputs = ctx.attr.outputs,
        ),
    ]

"""Adds an API template library rule"""
api_template = rule(
    _api_template_impl,
    attrs = {
        "template": attr.label(
            single_file = True,
            allow_files = True,
        ),
        "includes": attr.label_list(allow_files = True),
        "outputs": attr.string_list(),
        "deps": attr.label_list(providers = [GoLibrary]),
        "_go_context_data": attr.label(default=Label("@io_bazel_rules_go//:go_context_data")),
    },
    toolchains = ["@io_bazel_rules_go//go:toolchain"],
)
