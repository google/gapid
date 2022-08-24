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

# True source of AGI versions.
# Increment these numbers immediately after releasing a new version.
AGI_VERSION_MAJOR="3"
AGI_VERSION_MINOR="2"
AGI_VERSION_POINT="0"

# See bazel.rc. Can be overriden on the command line with:
#   bazel build --define AGI_BUILD_NUMBER=<#> --define AGI_BUILD_SHA=<sha>
AGI_BUILD_NUMBER="$(AGI_BUILD_NUMBER)"
AGI_BUILD_SHA="$(AGI_BUILD_SHA)"

def _agi_version(ctx):
    ctx.actions.expand_template(
        template = ctx.file.template,
        output = ctx.outputs.out,
        substitutions = {
            "@AGI_VERSION_MAJOR@": AGI_VERSION_MAJOR,
            "@AGI_VERSION_MINOR@": AGI_VERSION_MINOR,
            "@AGI_VERSION_POINT@": AGI_VERSION_POINT,
            "@AGI_BUILD_NUMBER@": ctx.var.get("AGI_BUILD_NUMBER"),
            "@AGI_BUILD_SHA@": ctx.var.get("AGI_BUILD_SHA"),
        }
    )

agi_version = rule(
    implementation=_agi_version,
    attrs = {
        "template": attr.label(
            mandatory = True,
            allow_single_file = True,
        ),
        "out": attr.output(
            mandatory = True,
        ),
    },
    output_to_genfiles = True,
)
