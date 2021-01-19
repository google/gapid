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

_BUILD_FILE_ATTRS = {
  "build_file": attr.label(
      allow_single_file = True,
  ),
  "build_file_content": attr.string(),
}

def _add_build_file(ctx):
    if ctx.attr.build_file:
      ctx.symlink(ctx.attr.build_file, "BUILD.bazel")
      return True
    if ctx.attr.build_file_content:
      ctx.file("BUILD.bazel", ctx.attr.build_file_content)
      return True
    return False

def github_http_args(organization, project, branch, commit):
  ref = ""
  if commit:
    ref = commit
  elif branch:
    ref = branch
  else:
    fail("You must specify either commit or branch")
  return struct(
    url = "https://codeload.github.com/{organization}/{project}/zip/{ref}".format(
      organization = organization,
      project = project,
      ref = ref
    ),
    type = "zip",
    strip_prefix = "{project}-{ref}".format(
      project = project,
      ref = ref
    ),
  )

def _github_repository_impl(ctx):
  args = github_http_args(
    organization = ctx.attr.organization,
    project = ctx.attr.project,
    branch = ctx.attr.branch,
    commit = ctx.attr.commit,
  )
  ctx.download_and_extract(
    url = args.url,
    type = args.type,
    stripPrefix = args.strip_prefix,
    sha256 = ctx.attr.sha256,
  )
  _apply_patch(ctx)
  _add_build_file(ctx)

_github_repository = repository_rule(
    _github_repository_impl,
    attrs = dict(_BUILD_FILE_ATTRS,
        organization = attr.string(mandatory = True),
        project = attr.string(mandatory = True),
        branch = attr.string(),
        commit = attr.string(),
        sha256 = attr.string(),
        patch_file = attr.string(),
    ),
)

def github_repository(name, path="", **kwargs):
  if path:
    print("Using local copy for {} at {}".format(name, path))
    native.local_repository(name = name, path = path)
  else:
    _github_repository(name=name, **kwargs)

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

def _apply_patch(ctx):
  if ctx.attr.patch_file:
    ctx.symlink(Label(ctx.attr.patch_file), "patch_to_repository.patch")
    cmd = "cd \"{}\" && /usr/bin/patch -p1 -i patch_to_repository.patch".format(ctx.path("."))
    print("Applying patch: " + cmd)
    bash_exe = "bash"
    if ctx.os.name.startswith("windows"):
      bash_exe = ctx.os.environ["BAZEL_SH"] if "BAZEL_SH" in ctx.os.environ else "c:/tools/msys64/usr/bin/bash.exe"
    result = ctx.execute([bash_exe, "--login", "-c", cmd])
    if result.return_code:
        fail("Failed to apply patch: (%d)\n%s" % (result.return_code, result.stderr))
    else:
        print("Patch applied successfully")

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
