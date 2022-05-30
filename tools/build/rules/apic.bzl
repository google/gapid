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

load("@io_bazel_rules_go//go:def.bzl", "go_context")
load(":gapil.bzl", "ApicTemplate")
load("@bazel_tools//tools/cpp:toolchain_utils.bzl", "find_cpp_toolchain")

def api_search_path(inputs):
    roots = {}
    for dep in inputs:
        if dep.root.path:
            roots[dep.root.path] = True
    return ",".join(["."] + roots.keys())

def _apic_binary_impl(ctx):
    api = ctx.attr.api
    apilist = api.includes.to_list()
    generated = depset()

    outputs = [ctx.actions.declare_file(ctx.label.name + ".bapi")]
    generated += outputs

    ctx.actions.run(
        inputs = apilist,
        outputs = outputs,
        arguments = [
            "binary",
            "--search",
            api_search_path(apilist),
            "--output",
            outputs[0].path,
            api.main.path,
        ],
        mnemonic = "apic",
        progress_message = "apic binary " + api.main.short_path,
        executable = ctx.executable._apic,
        use_default_shell_env = True,
    )

    return [
        DefaultInfo(files = depset(generated)),
    ]

"""Adds an API binary rule"""
apic_binary = rule(
    _apic_binary_impl,
    attrs = {
        "api": attr.label(
            allow_files = False,
            mandatory = True,
            providers = [
                "main",
                "includes",
            ],
        ),
        "_apic": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/apic:apic"),
        ),
    },
)

def _apic_compile_impl(ctx):
    apis = ctx.attr.apis
    apilist = []
    for api in ctx.attr.apis:
        apilist += api.includes.to_list()

    outputs = [ctx.actions.declare_file(ctx.label.name + ".cpp")]

    ctx.actions.run(
        inputs = apilist,
        outputs = outputs,
        arguments = [
            "compile",
            "--search", api_search_path(apilist),
            "--namespace", ctx.attr.namespace,
            "--output", outputs[0].path,
        ] + [ api.main.path for api in apis ],
        mnemonic = "apic",
        progress_message = "apic compiling apis",
        executable = ctx.executable._apic,
        use_default_shell_env = True,
    )

    return [
        DefaultInfo(files = depset(outputs)),
    ]

"""Adds an API compile rule"""
apic_compile = rule(
    _apic_compile_impl,
    attrs = {
        "apis": attr.label_list(
            allow_files = False,
            mandatory = True,
            providers = [
                "main",
                "includes",
            ],
        ),
        "namespace": attr.string(
            mandatory = True,
        ),
        "_apic": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/apic:apic"),
        ),
    },
)

def _apic_template_impl(ctx):
    api = ctx.attr.api
    apiname = api.apiname
    apilist = api.includes.to_list()
    generated = []
    for template in ctx.attr.templates:
        template = template[ApicTemplate]
        templatelist = template.uses.to_list()
        outputs = [ctx.actions.declare_file(out.format(api = apiname)) for out in template.outputs]
        generated += outputs
        ctx.actions.run(
            inputs = apilist + templatelist,
            outputs = outputs,
            arguments = [
                "template",
                "--dir",
                outputs[0].dirname,
                "--search",
                api_search_path(apilist),
                api.main.path,
                template.main.path,
            ],
            mnemonic = "apic",
            progress_message = "apic generating " + api.main.short_path + " with " + template.main.short_path,
            executable = ctx.executable._apic,
            use_default_shell_env = True,
        )
    return [
        DefaultInfo(files = depset(generated)),
    ]

"""Adds an API template rule"""
apic_template = rule(
    _apic_template_impl,
    attrs = {
        "api": attr.label(
            allow_files = False,
            mandatory = True,
            providers = [
                "apiname",
                "main",
                "includes",
            ],
        ),
        "templates": attr.label_list(
            allow_files = False,
            mandatory = True,
            providers = [ApicTemplate],
        ),
        "_apic": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//cmd/apic:apic"),
        ),
    },
    output_to_genfiles = True,
)
