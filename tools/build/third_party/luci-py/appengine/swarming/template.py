# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import os

from components import template
from server import config

ROOT_DIR = os.path.dirname(os.path.abspath(__file__))


def bootstrap():
  template.bootstrap({
    'swarming': os.path.join(ROOT_DIR, 'templates'),
    'wcui': os.path.join(ROOT_DIR, 'ui2', 'dist')
    })


def reset():
  template.reset()


def render(name, params=None):
  """Shorthand to render a template."""
  out = {
    'google_analytics': config.settings().google_analytics,
  }
  out.update(params or {})
  return template.render(name, out)
