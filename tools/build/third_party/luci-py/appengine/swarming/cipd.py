# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""CIPD-specific code is concentrated here."""

import re

# Regular expressions and other validation rules below are copied from
# https://chromium.googlesource.com/infra/luci/luci-go/+/61fc2ffc/cipd/common/common.go

PACKAGE_NAME_RE = re.compile(r'^([a-z0-9_\-\.]+/)*[a-z0-9_\-\.]+$')
PACKAGE_NAME_TEMPLATE_RE = re.compile(r'^[a-z0-9_\-\.,\${}/=]+$')
INSTANCE_ID_RE = re.compile(r'^([0-9a-f]{40})|([a-zA-Z0-9_\-]{44,})$')
TAG_KEY_RE = re.compile(r'^[a-z0-9_\-]+$')
REF_RE = re.compile(r'^[a-z0-9_./\-]{1,256}$')
TAG_MAX_LEN = 400


def is_valid_package_name(package_name):
  """Returns True if |package_name| is a valid CIPD package name."""
  return (
      package_name and
      bool(PACKAGE_NAME_RE.match(package_name)) and
      all(c.replace('.', '') != '' for c in package_name.split('/')))


def is_valid_package_name_template(template):
  """Returns True if |package_name| is a valid CIPD package name template."""
  parts = re.split(r'\${(.+)}', template)
  return (template and
      template.count('${') == template.count('{') == template.count('}') and
      all(PACKAGE_NAME_TEMPLATE_RE.match(p) for p in parts if p))


def is_valid_version(version):
  """Returns True if |version| is a valid CIPD package version."""
  return bool(
      is_valid_instance_id(version) or
      is_valid_tag(version) or
      REF_RE.match(version)
  )


def is_valid_tag(tag):
  """True if string looks like a valid package instance tag."""
  if not tag or ':' not in tag or len(tag) > TAG_MAX_LEN:
    return False
  # Care only about the key. Value can be anything (including empty string).
  return bool(TAG_KEY_RE.match(tag.split(':', 1)[0]))


def is_valid_instance_id(version):
  """Returns True if |version| is an instance_id.

  I.e. it is either a 40-char long hex strings (for legacy SHA1 ids), or
  a sufficiently long urlsafe base64-encoded unpadded string (for newer ids, in
  particular SHA256 ones).
  """
  return bool(INSTANCE_ID_RE.match(version))


def is_pinned_version(version):
  """Returns True if |version| is pinned."""
  return is_valid_instance_id(version) or is_valid_tag(version)
