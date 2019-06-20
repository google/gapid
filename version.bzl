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

# True source of GAPID versions.
# Increment these numbers immediately after releasing a new version.
GAPID_VERSION_MAJOR="1"
GAPID_VERSION_MINOR="7"
GAPID_VERSION_POINT="0"

# See bazel.rc. Can be overriden on the command line with:
#   bazel build --define GAPID_BUILD_NUMBER=<#> --define GAPID_BUILD_SHA=<sha>
GAPID_BUILD_NUMBER="$(GAPID_BUILD_NUMBER)"
GAPID_BUILD_SHA="$(GAPID_BUILD_SHA)"

def _gapid_version(ctx):
    ctx.actions.expand_template(
        template = ctx.file.template,
        output = ctx.outputs.out,
        substitutions = {
            "@GAPID_VERSION_MAJOR@": GAPID_VERSION_MAJOR,
            "@GAPID_VERSION_MINOR@": GAPID_VERSION_MINOR,
            "@GAPID_VERSION_POINT@": GAPID_VERSION_POINT,
            "@GAPID_BUILD_NUMBER@": ctx.var.get("GAPID_BUILD_NUMBER"),
            "@GAPID_BUILD_SHA@": ctx.var.get("GAPID_BUILD_SHA"),
        }
    )

gapid_version = rule(
    implementation=_gapid_version,
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
