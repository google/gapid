# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Access control groups for ereporter2."""

from components import auth
from . import config


RECIPIENTS_AUTH_GROUP = config.config.RECIPIENTS_AUTH_GROUP
VIEWERS_AUTH_GROUP = config.config.VIEWERS_AUTH_GROUP


def get_ereporter2_recipients():
  """Returns list of emails to send reports to."""
  return [
    x.name for x in auth.list_group(RECIPIENTS_AUTH_GROUP).members if x.is_user
  ]


def is_ereporter2_viewer():
  """True if current user is in recipients list, viewer list or is an admin."""
  if auth.is_admin() or auth.is_group_member(VIEWERS_AUTH_GROUP):
    return True
  ident = auth.get_current_identity()
  return ident.is_user and ident.name in get_ereporter2_recipients()


def is_ereporter2_editor():
  """Only auth admins or recipients can edit the silencing filters."""
  return auth.is_admin() or auth.is_group_member(RECIPIENTS_AUTH_GROUP)
