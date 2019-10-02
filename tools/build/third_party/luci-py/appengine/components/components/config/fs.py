# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""fs.Provider reads configs from the filesystem."""

import logging
import os

from google.appengine.ext import ndb

from components.config import common

# SEPARATOR separates config set name and config file path.
# A config set name cannot contain 'CONFIGS' because of the capilization.
SEPARATOR = 'CONFIGS'


class Provider(object):
  """Filesystem-based configuration provider.

  Some functions are made NDB tasklets because they follow common interface.
  See api._get_config_provider for context.
  """

  def __init__(self, root):
    assert root
    self.root = root

  @ndb.tasklet
  def get_async(self, config_set, path, dest_type=None, **kwargs):
    """Reads a (revision, config) from a file, where revision is always None.

    Kwargs are not used, but reported as warnings.
    """
    assert config_set
    assert path
    if kwargs:
      logging.warning(
          'config: parameters %r are ignored in the filesystem mode',
          kwargs.keys())
    filename = os.path.join(
        self.root,
        config_set.replace('/', os.path.sep),
        SEPARATOR,
        path.replace('/', os.path.sep))
    filename = os.path.abspath(filename)
    assert filename.startswith(os.path.abspath(self.root)), filename
    content = None
    if os.path.exists(filename):
      with open(filename, 'rb') as f:
        content = f.read()
    config = common._convert_config(content, dest_type)
    raise ndb.Return(None, config)

  def get_project_ids(self):
    # A project_id cannot contain a slash, so recursion is not needed.
    projects_dir = os.path.join(self.root, 'projects')
    if not os.path.isdir(projects_dir):
      return
    for pid in os.listdir(projects_dir):
      if os.path.isdir(os.path.join(projects_dir, pid)):
        yield pid

  @ndb.tasklet
  def get_projects_async(self):
    projects = [{'id': pid} for pid in sorted(self.get_project_ids())]
    # TODO(nodir): read project names from projects/<pid>:project.cfg
    raise ndb.Return(projects)

  def get_project_refs(self, project_id):
    assert project_id
    assert os.path.sep not in project_id, project_id
    project_path = os.path.join(self.root, 'projects', project_id) + os.path.sep
    refs_dir = project_path + 'refs'
    if not os.path.isdir(refs_dir):
      return
    for dirpath, dirs, _ in os.walk(refs_dir):
      if SEPARATOR in dirs:
        # This is a leaf of the ref tree.
        dirs.remove(SEPARATOR)  # Do not go deeper.
        yield dirpath[len(project_path):]

  @ndb.tasklet
  def get_project_configs_async(self, path):
    """Reads a config file in all projects.

    Returns:
      {config_set -> (revision, content)} map, where revision is always None.
    """
    assert path
    config_sets = ['projects/%s' % pid for pid in self.get_project_ids()]
    result = {}
    for config_set in config_sets:
      rev, content = yield self.get_async(config_set, path)
      if content is not None:
        result[config_set] = (rev, content)
    raise ndb.Return(result)

  @ndb.tasklet
  def get_ref_configs_async(self, path):
    """Reads a config file in all refs of all projects.

    Returns:
      {config_set -> (revision, content)} map, where revision is always None.
    """
    assert path
    result = {}
    for pid in self.get_project_ids():
      for ref in self.get_project_refs(pid):
        config_set = 'projects/%s/%s' % (pid, ref)
        rev, content = yield self.get_async(config_set, path)
        if content is not None:
          result[config_set] = (rev, content)
    raise ndb.Return(result)

  @ndb.tasklet
  def get_config_set_location_async(self, _config_set):
    """Returns URL of where configs for given config set are stored.

    Returns:
      Always None for file system.
    """
    raise ndb.Return(None)


def get_provider():  # pragma: no cover
  return Provider(common.CONSTANTS.CONFIG_DIR)
