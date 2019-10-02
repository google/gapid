#!/usr/bin/env python
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Given current git checkout state return version string to use for an app."""

import contextlib
import getpass
import logging
import optparse
import os
import string
import subprocess
import sys

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(ROOT_DIR, '..', 'third_party_local'))

from depot_tools import git_number
from depot_tools import git_common


# Defines an error when generating the version for app engine.
class VersionError(Exception):
  pass


def git(cmd, cwd):
  return subprocess.check_output(['git'] + cmd, cwd=cwd)


@contextlib.contextmanager
def chdir(path):
  orig = os.getcwd()
  try:
    os.chdir(path)
    yield
  finally:
    os.chdir(orig)


def get_head_pseudo_revision(root, remote):
  """Returns the pseudo revision number and commit hash describing
  the base upstream commit this branch is based on.

  The base upstream commit hash is determined by 'git merge-base'. See the man
  page for more information.

  The pseudo revision is calculated by the number of commits separating the base
  upstream commit from the rootest commit.  The earliest commit should be a root
  commit, e.g. a commit with no parent. A git tree can have multiple root
  commits when git repositories are merged together. The oldest one will be
  selected. The list of all root commits can be retrieved with:

    git rev-list --parents HEAD | egrep "^[a-f0-9]{40}$"

  Returns:
    tuple of:
    - pseudo revision number as a int
    - upstream commit hash this branch is based of.
  """
  mergebase = git(['merge-base', 'HEAD', remote], cwd=root).rstrip()
  with chdir(root):
    targets = git_common.parse_commitrefs(mergebase)
    git_number.load_generation_numbers(targets)
    git_number.finalize(targets)
    return git_number.get_num(targets[0]), mergebase


def get_remote_pseudo_revision(root, remote):
  """Returns the pseudo revision number and commit hash describing
  the base upstream commit the remote branch is based on.

  The base upstream commit hash is determined by 'git rev-parse'. See the man
  page for more information.

  See get_head_pseudo_revision for more info about the pseudo revision.

  Returns:
    tuple of:
    - pseudo revision number as a int
    - upstream commit hash this branch is based of.
  """
  mergebase = git(['rev-parse', remote], cwd=root).rstrip()
  with chdir(root):
    targets = git_common.parse_commitrefs(mergebase)
    git_number.load_generation_numbers(targets)
    git_number.finalize(targets)
    return git_number.get_num(targets[0]), mergebase


def is_pristine(root, mergebase):
  """Returns True if the tree is pristine relating to mergebase."""
  head = git(['rev-parse', 'HEAD'], cwd=root).rstrip()
  logging.info('head: %s, mergebase: %s', head, mergebase)

  if head != mergebase:
    return False

  # Look for local uncommitted diff.
  return not (
      git(['diff', '--ignore-submodules=none', mergebase], cwd=root) or
      git(['diff', '--ignore-submodules', '--cached', mergebase], cwd=root) or
      git(['status', '-s', '--porcelain=v2'], cwd=root))


def calculate_version(root, tag, additional_chars=0):
  """Returns a tag for a git checkout.

  Uses the pseudo revision number from the upstream commit this branch is based
  on, the abbreviated commit hash. Adds -tainted-<username>-<tag> if the code is
  not pristine. Use the branch name is tag is not specified. If version is over
  63 characters, truncation is done. 'additional_chars' indicates that
  additional characters will be added, and thus that the limit for version
  should actually be 63 - additional_chars.
  """
  pseudo_revision, mergebase = get_head_pseudo_revision(root, 'origin/master')
  pristine = is_pristine(root, mergebase)
  user = getpass.getuser()
  if not pristine and not tag:
    # Automatically add the branch name as a tag if possible.
    tag = _get_cleaned_git_branch_name(root)

  # Per https://tools.ietf.org/html/rfc1035#section-2.3.1 and
  # https://tools.ietf.org/html/rfc2181#section-11, labels in domains can't be
  # more than 63 chars - additional_chars.
  return _get_limited_version(pseudo_revision, mergebase, pristine, user, tag,
                             63 - additional_chars)


def _get_cleaned_git_branch_name(root):
  """Returns the current git branch name if appropriate to use as an AppEngine
  version tag.
  """
  branch = git(['rev-parse', '--abbrev-ref', 'HEAD'], cwd=root).strip()
  if branch == 'HEAD':
    return ''
  # The official regexp is '^(?:^(?!-)[a-z\d\-]{0,62}[a-z\d]$)$'.
  valid = string.ascii_lowercase + string.digits + '-'
  # Strip unexpected characters. At worst it'll be an empty string, which will
  # result in no tag.
  clean = branch.lower().replace('_', '-').strip('-').lower()
  return ''.join(b for b in clean if b in valid)


def _get_limited_version(
    pseudo_revision, mergebase, pristine, user, tag, limit):
  """Return version, limited to the given 'limit' number of chars.

  Arguments:
  - pseudo_revision (int): representing the number of commits since the root
        commit.
  - mergebase (str): commit hash of the root commit upstream this branch is
        based on.
  - pristine (bool): specify if HEAD is a commit in upstream.
  - user (str): current user name.
  - tag (str): suffix to append.
  - limit (int): maximum number of characters the returned string will have.

  If it's short enough, return it as-is. Otherwise:
  - trim '-tainted'.
  - trim username while keeping a minimum of 6 characters.
  - cut the tag off.
  """
  tainted_text = '-tainted-%s' % user if not pristine else ''
  version = _get_version(pseudo_revision, mergebase, tainted_text, tag)

  # If already under the limit, return what we have.
  orig_version_len = len(version)
  if orig_version_len <= limit:
    return version

  # Shorten tainted_text by excluding '-tainted' (saves 8 chars) and truncating
  # user as needed to fit in the char limit, though (somewhat arbitrarily) we
  # refuse to truncate the username to less than 6 chars in order to maintain
  # some chance of identifying the user who deployed (though it's ok if the
  # username is less than 6 chars to begin with).
  min_user_chars = 6

  # Try with full username (just removing '-tainted').
  version = _get_version(pseudo_revision, mergebase, '-%s' % user, tag)
  if len(version) <= limit:
    return version

  # That's still not enough. Truncate username (in addition to removing the
  # 8 chars of tainted), bailing if we try to go lower than min_user_chars.
  current_chars_without_user = orig_version_len - len(user) - 8
  chars_for_user = limit - current_chars_without_user
  tainted_text = '-%s' % user[:max(min_user_chars, chars_for_user)]
  version = _get_version(
      pseudo_revision, mergebase, tainted_text, tag).strip('-')
  if len(version) <= limit:
    return version

  # Regenerate an even smaller version by cutting off the tag.
  return version[:limit].rstrip('-')


def _get_version(pseudo_revision, mergebase, tainted_text, tag):
  """Returns version based on given args."""

  # Build version, trimming mergebase to 7 characters like 'git describe' does
  # (since 40 chars is overwhelming)!
  version = '%s-%s' % (pseudo_revision, mergebase[:7])
  version += tainted_text
  if tag:
    version += '-' + tag
  return version


def checkout_root(cwd):
  """Returns the root of the checkout."""
  return git(['rev-parse', '--show-toplevel'], cwd).rstrip()


def main():
  parser = optparse.OptionParser(description=sys.modules[__name__].__doc__)
  parser.add_option('-v', '--verbose', action='store_true')
  parser.add_option(
      '-t', '--tag', help='Tag to attach to a tainted version')
  options, args = parser.parse_args()
  logging.basicConfig(level=logging.DEBUG if options.verbose else logging.ERROR)

  if args:
    parser.error('Unknown arguments, %s' % args)

  root = checkout_root(os.getcwd())
  logging.info('Checkout root is %s', root)
  try:
    print calculate_version(root, options.tag)
  except VersionError as e:
    sys.stderr.write(str(e))
    return 1

  return 0


if __name__ == '__main__':
  sys.exit(main())
