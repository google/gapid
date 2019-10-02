#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Wrapper around GAE SDK tools to simplify working with multi module apps."""

__version__ = '1.2'

import atexit
import code
import optparse
import os
import signal
import sys
import tempfile
import urllib2

try:
  import readline
except ImportError:
  readline = None

# In case gae.py was run via symlink, find the original file since it's where
# third_party libs are. Handle a chain of symlinks too.
SCRIPT_PATH = os.path.abspath(__file__)
IS_SYMLINKED = False
while True:
  try:
    SCRIPT_PATH = os.path.abspath(
        os.path.join(os.path.dirname(SCRIPT_PATH), os.readlink(SCRIPT_PATH)))
    IS_SYMLINKED = True
  except OSError:
    break

ROOT_DIR = os.path.dirname(os.path.dirname(SCRIPT_PATH))
sys.path.insert(0, ROOT_DIR)
sys.path.insert(0, os.path.join(ROOT_DIR, '..', 'third_party_local'))

import colorama
from depot_tools import subcommand

from tool_support import gae_sdk_utils
from tools import calculate_version
from tools import log_since


def _print_version_log(app, to_version):
  """Queries the server active version and prints the log between the active
  version and the new version.
  """
  from_versions = set(service['id'] for service in app.get_actives())
  if len(from_versions) > 1:
    print >> sys.stderr, (
        'Error: found multiple modules with different active versions. Use '
        '"gae active" to get the curent list of active version. Please use the '
        'Web UI to fix. Aborting.')
    return 1
  if from_versions:
    from_version = list(from_versions)[0]
    start = int(from_version.split('-', 1)[0])
    end = int(to_version.split('-', 1)[0])
    if start < end:
      pseudo_revision, mergebase = calculate_version.get_remote_pseudo_revision(
          app.app_dir, 'origin/master')
      logs, _ = log_since.get_logs(
          app.app_dir, pseudo_revision, mergebase, start, end)
      print('\nLogs between %s and %s:' % (from_version, to_version))
      print('%s\n' % logs)


##


def CMDappcfg_login(parser, args):
  """Sets up authentication for appcfg.py usage [DEPRECATED]."""
  app, _, _ = parser.parse_args(args)
  print (
      'Since appcfg.py doesn\'t support explicit login command, we\'ll run '
      'innocent "list_version" instead. It will trigger appcfg\'s login flow. '
      '\n'
      'It\'s fine if "list_version" call itself fails - at this point we have '
      'the necessary credentials cached and other subcommands should be able '
      'to use them.\n')
  gae_sdk_utils.appcfg_login(app)
  return 0


def CMDactive(parser, args):
  """Prints the active versions on the server.

  This is an approximation of querying which version is the default.
  """
  parser.add_option(
      '-b', '--bare', action='store_true',
      help='Only print the version(s), nothing else')
  app, options, _modules = parser.parse_args(args)
  data = app.get_actives()
  if options.bare:
    print('\n'.join(sorted(set(i['id'] for i in data))))
    return 0
  print('%s:' % app.app_id)
  for service in data:
    print(
        '  %s: %s by %s at %s' % (
          service['service'], service['id'], service['deployer'],
          service['creationTime']))
  return 0


def CMDapp_dir(parser, args):
  """Prints a root directory of the application."""
  # parser.app_dir is None if app root directory discovery fails. Fail the
  # command even before invoking CLI parser, or it will ask to pass --app_dir to
  # 'app-dir' subcommand, which is ridiculous.
  if not parser.app_dir:
    print >> sys.stderr, 'Can\'t discover an application root directory.'
    return 1
  parser.add_tag_option()
  app, _, _ = parser.parse_args(args)
  print app.app_dir
  return 0


@subcommand.usage('[version_id version_id ...]')
def CMDcleanup(parser, args):
  """Removes old versions of GAE application modules.

  Removes the specified versions from all app modules. If no versions are
  provided via command line, will ask interactively.

  When asking interactively, uses EDITOR environment variable to edit the list
  of versions. Otherwise uses notepad.exe on Windows, or vi otherwise.
  """
  parser.add_force_option()
  parser.allow_positional_args = True
  app, options, versions_to_remove = parser.parse_args(args)

  if not versions_to_remove:
    # List all deployed versions, dump them to a temp file to be edited.
    versions = app.get_uploaded_versions()
    fd, path = tempfile.mkstemp()
    atexit.register(lambda: os.remove(path))
    with os.fdopen(fd, 'w') as f:
      header = (
        '# Remove lines that correspond to versions\n'
        '# you\'d like to delete from \'%s\'.\n')
      f.write(header % app.app_id + '\n'.join(versions) + '\n')

    # Let user remove versions that are no longer needed.
    editor = os.environ.get(
        'EDITOR', 'notepad.exe' if sys.platform == 'win32' else 'vi')
    exit_code = os.system('%s %s' % (editor, path))
    if exit_code:
      print('Aborted.')
      return exit_code

    # Read back the file that now contains only versions to keep.
    keep = []
    with open(path, 'r') as f:
      for line in f:
        line = line.strip()
        if not line or line.startswith('#'):
          continue
        if line not in versions:
          print >> sys.stderr, 'Unknown version: %s' % line
          return 1
        if line not in keep:
          keep.append(line)

    # Calculate a list of versions to remove.
    versions_to_remove = [v for v in versions if v not in keep]
    if not versions_to_remove:
      print('Nothing to do.')
      return 0

  # Deleting a version is a destructive operation, confirm.
  if not options.force:
    ok = gae_sdk_utils.confirm(
        'Delete the following versions?', app, versions_to_remove)
    if not ok:
      print('Aborted.')
      return 1

  for version in versions_to_remove:
    print('Deleting %s...' % version)
    app.delete_version(version)

  return 0


@subcommand.usage('[extra arguments for dev_appserver.py]')
def CMDdevserver(parser, args):
  """Runs the app locally via dev_appserver.py."""
  parser.allow_positional_args = True
  parser.disable_interspersed_args()
  parser.add_option(
      '-o', '--open', action='store_true',
      help='Listen to all interfaces (less secure)')
  app, options, args = parser.parse_args(args)
  # Let dev_appserver.py handle Ctrl+C interrupts.
  signal.signal(signal.SIGINT, signal.SIG_IGN)
  return app.run_dev_appserver(args, options.open)


@subcommand.usage('[module_id version_id]')
def CMDshell(parser, args):
  """Opens interactive remote shell with app's GAE environment.

  Connects to a specific version of a specific module (an active version of
  'default' module by default). The app must have 'remote_api: on' builtin
  enabled in app.yaml.

  Always uses password based authentication.
  """
  parser.allow_positional_args = True
  parser.add_option(
      '-H', '--host', help='Only necessary if not hosted on .appspot.com')
  parser.add_option(
      '--local', action='store_true',
      help='Operates locally on an empty dev instance')
  app, options, args = parser.parse_args(args)

  module = 'default'
  version = None
  if len(args) == 2:
    module, version = args
  elif len(args) == 1:
    module = args[0]
  elif args:
    parser.error('Unknown args: %s' % args)

  if module not in app.modules:
    parser.error('No such module: %s' % module)

  if not options.host and not options.local:
    prefixes = filter(None, (version, module, app.app_id))
    options.host = '%s.appspot.com' % '-dot-'.join(prefixes)

  # Ensure remote_api is initialized and GAE sys.path is set.
  gae_sdk_utils.setup_env(
      app.app_dir, app.app_id, version, module, remote_api=True)

  if options.host:
    # Open the connection.
    from google.appengine.ext.remote_api import remote_api_stub
    try:
      print('If asked to login, run:\n')
      print(
          'gcloud auth application-default login '
          '--scopes=https://www.googleapis.com/auth/appengine.apis,'
          'https://www.googleapis.com/auth/userinfo.email\n')
      remote_api_stub.ConfigureRemoteApiForOAuth(
          options.host, '/_ah/remote_api')
    except urllib2.URLError:
      print >> sys.stderr, 'Failed to access %s' % options.host
      return 1
    remote_api_stub.MaybeInvokeAuthentication()

  def register_sys_path(*path):
    abs_path = os.path.abspath(os.path.join(*path))
    if os.path.isdir(abs_path) and not abs_path in sys.path:
      sys.path.insert(0, abs_path)

  # Simplify imports of app modules (with dependencies). This code is optimized
  # for layout of apps that use 'components'.
  register_sys_path(app.app_dir)
  register_sys_path(app.app_dir, 'third_party')
  register_sys_path(app.app_dir, 'components', 'third_party')

  # Import some common modules into interactive console namespace.
  def setup_context():
    # pylint: disable=unused-variable
    from google.appengine.api import app_identity
    from google.appengine.api import memcache
    from google.appengine.api import urlfetch
    from google.appengine.ext import ndb
    return locals().copy()
  context = setup_context()

  # Fancy readline support.
  if readline is not None:
    readline.parse_and_bind('tab: complete')
    history_file = os.path.expanduser(
        '~/.config/gae_tool/remote_api_%s' % app.app_id)
    if not os.path.exists(os.path.dirname(history_file)):
      os.makedirs(os.path.dirname(history_file))
    atexit.register(lambda: readline.write_history_file(history_file))
    if os.path.exists(history_file):
      readline.read_history_file(history_file)

  prompt = [
    'App Engine interactive console for "%s".' % app.app_id,
    'Available symbols:',
  ]
  prompt.extend(sorted('  %s' % symbol for symbol in context))
  code.interact('\n'.join(prompt), None, context)
  return 0


@subcommand.usage('[version_id]')
def CMDswitch(parser, args):
  """Switches default version of all app modules.

  The version must be uploaded already. If no version is provided via command
  line, will ask interactively.
  """
  parser.add_switch_option()
  parser.add_force_option()
  parser.allow_positional_args = True
  app, options, version = parser.parse_args(args)
  if len(version) > 1:
    parser.error('Unknown args: %s' % version[1:])
  version = None if not version else version[0]

  # Interactively pick a version if not passed via command line.
  if not version:
    versions = app.get_uploaded_versions()
    if not versions:
      print('Upload a version first.')
      return 1

    print('Specify a version to switch to:')
    for version in versions:
      print('  %s' % version)

    version = (
        raw_input('Switch to version [%s]: ' % versions[-1]) or versions[-1])
    if version not in versions:
      print('No such version.')
      return 1

  _print_version_log(app, version)
  # Switching a default version is disruptive operation. Require confirmation.
  if (not options.force and
      not gae_sdk_utils.confirm('Switch default version?', app, version)):
    print('Aborted.')
    return 1
  app.set_default_version(version)
  return 0


@subcommand.usage('[module_id module_id ...]')
def CMDupload(parser, args):
  """Uploads a new version of specific (or all) modules of an app.

  Note that module yamls are expected to be named module-<module name>.yaml

  Version name looks like <number>-<commit sha1>[-tainted-<who>], where:
    number      git commit number, monotonically increases with each commit
    commit sha1 upstream commit hash the branch is based of
    tainted     git repo has local modifications compared to upstream branch
    who         username who uploads the tainted version

  Doesn't make it a default unless --switch is specified. Use 'switch'
  subcommand to change default serving version.
  """
  parser.add_tag_option()
  parser.add_option(
      '-x', '--switch', action='store_true',
      help='Switch version after uploading new code')
  parser.add_switch_option()
  parser.add_force_option()
  parser.allow_positional_args = True
  app, options, modules = parser.parse_args(args)

  for module in modules:
    if module not in app.modules:
      parser.error('No such module: %s' % module)

  # Additional chars is for the app_id as well as 5 chars for '-dot-'.
  version = calculate_version.calculate_version(
    app.app_dir, options.tag, len(app.app_id)+5)

  # Updating indexes, queues, etc is a disruptive operation. Confirm.
  if not options.force:
    approved = gae_sdk_utils.confirm(
        'Upload new version, update indexes, queues and cron jobs?',
        app, version, modules, default_yes=True)
    if not approved:
      print('Aborted.')
      return 1

  app.update(version, modules)

  print('-' * 80)
  print('New version:')
  print('  %s' % version)
  print('Uploaded as:')
  print('  https://%s-dot-%s.appspot.com' % (version, app.app_id))
  print('Manage at:')
  print('  https://console.cloud.google.com/appengine/versions?project=' +
        app.app_id)
  print('-' * 80)

  if not options.switch:
    return 0
  if 'tainted-' in version:
    print('')
    print >> sys.stderr, 'Can\'t use --switch with a tainted version!'
    return 1
  _print_version_log(app, version)
  print('Switching as default version')
  app.set_default_version(version)
  return 0


def CMDversion(parser, args):
  """Prints version name that correspond to current state of the checkout.

  'update' subcommand uses this version when uploading code to GAE.

  Version name looks like <number>-<commit sha1>[-tainted-<who>], where:
    number      git commit number, monotonically increases with each commit
    commit sha1 upstream commit hash the branch is based of
    tainted     git repo has local modifications compared to upstream branch
    who         username who uploads the tainted version
  """
  parser.add_tag_option()
  app, options, _ = parser.parse_args(args)
  # Additional chars is for the app_id as well as 5 chars for '-dot-'.
  print(calculate_version.calculate_version(
    app.app_dir, options.tag, len(app.app_id)+5))
  return 0


class OptionParser(optparse.OptionParser):
  """OptionParser with some canned options."""

  def __init__(self, app_dir, **kwargs):
    optparse.OptionParser.__init__(
        self,
        version=__version__,
        description=sys.modules['__main__'].__doc__,
        **kwargs)
    self.default_app_dir = app_dir
    self.allow_positional_args = False

  def add_tag_option(self):
    self.add_option('-t', '--tag', help='Tag to attach to a tainted version')

  def add_switch_option(self):
    self.add_option(
        '-n', '--no-log', action='store_true',
        help='Do not print logs from the current server active version to the '
             'one being switched to')

  def add_force_option(self):
    self.add_option(
        '-f', '--force', action='store_true',
        help='Do not ask for confirmation')

  def parse_args(self, *args, **kwargs):
    gae_sdk_utils.add_sdk_options(self, self.default_app_dir)
    options, args = optparse.OptionParser.parse_args(self, *args, **kwargs)
    if not self.allow_positional_args and args:
      self.error('Unknown arguments: %s' % args)
    app = gae_sdk_utils.process_sdk_options(self, options)
    return app, options, args


def _find_app_dir(search_dir):
  """Locates GAE app root directory (or returns None if not found).

  Starts by examining search_dir, then its parent, and so on, until it discovers
  git repository root or filesystem root.

  A directory is a suspect for an app root if it looks like an app root (has
  app.yaml or some of its subdir have app.yaml), but its parent directory does
  NOT look like an app root.

  It allows to detect multi-module Go apps. Their default module directory
  usually contains app.yaml, and this directory by itself looks like one-module
  GAE app. By looking at the parent we can detect that it's indeed just one
  module of multi-module app.

  This logic gives false positives if multiple different one-module GAE apps are
  located in sibling directories of some root directory (e.g. appengine/<app1>,
  appengine/<app2). To prevent this directory to be incorrectly used as an app
  root, we forbid root directories of this kind to directly contains apps.

  A root directory is denoted either by presence of '.git' subdir, or 'ROOT'
  file.
  """
  def is_root(p):
    return (
        os.path.isdir(os.path.join(p, '.git')) or
        os.path.isfile(os.path.join(p, 'ROOT')) or
        os.path.dirname(p) == p)

  cached_check = {}
  def is_app_dir(p):
    if p not in cached_check:
      cached_check[p] = not is_root(p) and gae_sdk_utils.is_app_dir(p)
    return cached_check[p]

  while not is_root(search_dir):
    parent = os.path.dirname(search_dir)
    if is_app_dir(search_dir) and not is_app_dir(parent):
      return search_dir
    search_dir = parent
  return None


def main(args):
  # gae.py may be symlinked into app's directory or its subdirectory (to avoid
  # typing --app-dir all the time). If linked into subdirectory, discover root
  # by locating app.yaml. It is used for Python GAE apps and one-module Go apps
  # that have all YAMLs in app root dir.
  default_app_dir = None
  if IS_SYMLINKED:
    script_dir = os.path.dirname(os.path.abspath(__file__))
    default_app_dir = _find_app_dir(script_dir)

  # If not symlinked into an app directory, try to discover app root starting
  # from cwd.
  default_app_dir = default_app_dir or _find_app_dir(os.getcwd())

  colorama.init()
  dispatcher = subcommand.CommandDispatcher(__name__)
  try:
    return dispatcher.execute(OptionParser(default_app_dir), args)
  except gae_sdk_utils.Error as e:
    print >> sys.stderr, str(e)
    return 1
  except KeyboardInterrupt:
    # Don't dump stack traces on Ctrl+C, it's expected flow in some commands.
    print >> sys.stderr, '\nInterrupted'
    return 1


if __name__ == '__main__':
  sys.exit(main(sys.argv[1:]))
