load("//tools/build/rules:repository.bzl", "github_http_args")
load("@io_bazel_rules_go//go:def.bzl", "go_http_archive")

def github_go_repository(name, organization, project, commit="", branch="", path="", **kwargs):
  if path:
    print("Override with {}".format(path))
  else:
    github = github_http_args(
        organization = organization,
        project = project,
        commit = commit,
        branch = branch,
      )
    go_http_archive(
      name = name,
      url = github.url,
      type = github.type,
      strip_prefix = github.strip_prefix,
      **kwargs
    )