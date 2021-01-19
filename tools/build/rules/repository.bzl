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

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def github_http_args(organization, project, commit):
  return struct(
    url = "https://codeload.github.com/{organization}/{project}/zip/{commit}".format(
      organization = organization,
      project = project,
      commit = commit,
    ),
    type = "zip",
    strip_prefix = "{project}-{commit}".format(
      project = project,
      commit = commit
    ),
  )

def github_repository(name, organization, project, commit, **kwargs):
  args = github_http_args(organization, project, commit)
  patch_args = kwargs.pop("patch_args", [ "-p1" ]) # sensible default

  http_archive(
    name = name,
    url = args.url,
    type = args.type,
    strip_prefix = args.strip_prefix,
    patch_args = patch_args,
    **kwargs
  )

def maybe_repository(repo_rule, name, locals, **kwargs):
    if name in native.existing_rules():
        return

    if name not in locals:
        repo_rule(name = name, **kwargs)
        return

    build_file = kwargs.get("build_file")
    if build_file == None:
        native.local_repository(
            name = name,
            path = locals.get(name)
        )
    else:
        native.new_local_repository(
            name = name,
            path = locals.get(name),
            build_file = build_file
        )

# This is *not* a complete maven implementation, it's just good enough for our rules.

_MAVEN_URL = "https://repo.maven.apache.org/maven2"
_MAVEN_RULE = """java_import(
    name = "jar{}",
    jars = ["{}"],
    visibility = ["//visibility:public"],
)
"""

def _maven_download(ctx, baseUrl, artifact, type, sha256):
    ctx.download(
        url = baseUrl + type + ".jar",
        output = artifact + type + ".jar",
        sha256 = sha256,
    )
    return [_MAVEN_RULE.format(type, artifact + type + ".jar")]

def _maven_jar_impl(ctx):
    toks = ctx.attr.artifact.split(":")
    if len(toks) != 3:
        fail("Invalid maven artifact: " + ctx.attr.artifact)

    baseUrl = "{}/{}/{}/{}/{}-{}".format(
        _MAVEN_URL, toks[0].replace(".", "/"), # group
        toks[1], toks[2], toks[1], toks[2])    # 2x (artifact and version)

    parts = _maven_download(ctx, baseUrl, toks[1], "", ctx.attr.sha256)
    if ctx.attr.sha256_src:
        parts += _maven_download(ctx, baseUrl, toks[1], "-sources", ctx.attr.sha256_src)
    if ctx.attr.sha256_linux:
        parts += _maven_download(ctx, baseUrl, toks[1], "-natives-linux", ctx.attr.sha256_linux)
    if ctx.attr.sha256_windows:
        parts += _maven_download(ctx, baseUrl, toks[1], "-natives-windows", ctx.attr.sha256_windows)
    if ctx.attr.sha256_macos:
        parts += _maven_download(ctx, baseUrl, toks[1], "-natives-macos", ctx.attr.sha256_macos)

    ctx.file("BUILD.bazel",
        content = "\n".join(parts),
        executable = False,
    )

maven_jar = repository_rule(
    _maven_jar_impl,
    attrs = {
        "artifact": attr.string(mandatory = True),
        "sha256": attr.string(mandatory = True),
        "sha256_src": attr.string(mandatory = False),
        "sha256_linux": attr.string(mandatory = False),
        "sha256_windows": attr.string(mandatory = False),
        "sha256_macos": attr.string(mandatory = False),
    },
)
