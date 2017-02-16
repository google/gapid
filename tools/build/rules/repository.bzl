_BUILD_FILE_ATTRS = {
  "build_file": attr.label(
      allow_files = True,
      single_file = True,
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

def _empty_repository_impl(ctx):
  if not _add_build_file(ctx):
    fail("You must specify either build_file or build_file_content'")

empty_repository = repository_rule(
    implementation = _empty_repository_impl,
    attrs = _BUILD_FILE_ATTRS,
)

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
    url=args.url,
    type=args.type,
    stripPrefix=args.strip_prefix,
  )
  _add_build_file(ctx)

_github_repository = repository_rule(
    implementation = _github_repository_impl,
    attrs = _BUILD_FILE_ATTRS + dict(
        organization = attr.string(mandatory = True),
        project = attr.string(mandatory = True),
        branch = attr.string(),
        commit = attr.string(),
    ),
)

def github_repository(name, path="", **kwargs):
  if path:
    print("Using local copy for {} at {}".format(name, path))
    native.local_repository(name = name, path = path)
  else:
    _github_repository(name=name, **kwargs)
