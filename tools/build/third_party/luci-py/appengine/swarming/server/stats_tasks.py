# coding: utf-8
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Generates statistics for the tasks. Contains the backend code."""


### Public API


def cron_generate_stats():
  """Returns the number of minutes processed."""
  # TODO(maruel): https://crbug.com/864722
  return 0
