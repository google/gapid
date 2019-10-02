# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Set of functions to work with GAE SDK tools."""

import collections
import glob
import json
import logging
import os
import re
import subprocess
import sys
import time


# 'setup_gae_sdk' loads 'yaml' module and modifies this variable.
yaml = None


# If True, all code here will use gcloud SDK (not GAE SDK). It assumes 'gcloud'
# tool is in PATH and the SDK has all necessary GAE components installed.
#
# This flag is temporary. Once gcloud support is fully implemented and gcloud is
# available on bots, this will become the default.
USE_GCLOUD = os.getenv('LUCI_PY_USE_GCLOUD') == '1'


# Directory with this file.
TOOLS_DIR = os.path.dirname(os.path.abspath(__file__))

# Name of a directory with Python GAE SDK.
PYTHON_GAE_SDK = 'google_appengine'

# Name of a directory with Go GAE SDK.
GO_GAE_SDK = 'go_appengine'

# Value of 'runtime: ...' in app.yaml -> SDK to use.
#
# TODO(vadimsh): Can be removed if using 'gcloud'.
RUNTIME_TO_SDK = {
  'go': GO_GAE_SDK,
  'python27': PYTHON_GAE_SDK,
}

# Path to a current SDK, set in setup_gae_sdk.
_GAE_SDK_PATH = None


class Error(Exception):
  """Base class for a fatal error."""


class BadEnvironmentError(Error):
  """Raised when required tools or environment are missing."""


class UnsupportedModuleError(Error):
  """Raised when trying to deploy MVM or Flex module."""


class LoginRequiredError(Error):
  """Raised by Application methods if use has to go through login flow."""


def find_gcloud():
  """Searches for 'gcloud' binary in PATH and returns absolute path to it.

  Raises BadEnvironmentError error if it's not there.
  """
  for path in os.environ['PATH'].split(os.pathsep):
    exe_file = os.path.join(path, 'gcloud')  # <sdk_root>/bin/gcloud
    if os.path.isfile(exe_file) and os.access(exe_file, os.X_OK):
      return os.path.realpath(exe_file)
  raise BadEnvironmentError(
      'Can\'t find "gcloud" in PATH. Install the Google Cloud SDK from '
      'https://cloud.google.com/sdk/')


def find_gae_sdk(sdk_name=PYTHON_GAE_SDK, search_dir=TOOLS_DIR):
  """Returns the path to GAE SDK if found, else None.

  TODO(vadimsh): Replace with find_gae_sdk_gcloud, get rid of arguments.
  """
  if USE_GCLOUD:
    return find_gae_sdk_gcloud()
  return find_gae_sdk_appcfg(sdk_name, search_dir)


def find_gae_sdk_gcloud():
  """Returns the path to GAE portion of Google Cloud SDK or None if not found.

  This is '<sdk_root>/platform/google_appengine'. It is documented here:
  https://cloud.google.com/appengine/docs/standard/python/tools/localunittesting

  It is shared between Python and Go flavors of GAE.
  """
  try:
    gcloud = find_gcloud()
  except BadEnvironmentError:
    return None
  # 'gcloud' is <sdk_root>/bin/gcloud.
  sdk_root = os.path.dirname(os.path.dirname(gcloud))
  return os.path.join(sdk_root, 'platform', 'google_appengine')


def find_gae_sdk_appcfg(sdk_name, search_dir):
  """Searches for appcfg.py to figure out where non-gcloud GAE SDK is.

  TODO(vadimsh): To be removed once gcloud is fully supported.
  """
  # First search up the directories up to root.
  while True:
    attempt = os.path.join(search_dir, sdk_name)
    if os.path.isfile(os.path.join(attempt, 'appcfg.py')):
      return attempt
    prev_dir = search_dir
    search_dir = os.path.dirname(search_dir)
    if search_dir == prev_dir:
      break
  # Next search PATH.
  markers = ['appcfg.py']
  if sdk_name == GO_GAE_SDK:
    markers.append('goroot')
  for item in os.environ['PATH'].split(os.pathsep):
    if not item:
      continue
    item = os.path.normpath(os.path.abspath(item))
    if all(os.path.exists(os.path.join(item, m)) for m in markers):
      return item
  return None


def find_app_yamls(app_dir):
  """Searches for app.yaml and module-*.yaml in app_dir or its subdirs.

  Recognizes Python and Go GAE apps.

  Returns:
    List of abs path to module yamls.

  Raises:
    ValueError if not a valid GAE app.
  """
  # Look in the root first. It's how python apps and one-module Go apps look.
  yamls = []
  app_yaml = os.path.join(app_dir, 'app.yaml')
  if os.path.isfile(app_yaml):
    yamls.append(app_yaml)
  yamls.extend(glob.glob(os.path.join(app_dir, 'module-*.yaml')))
  if yamls:
    return yamls

  # Look in per-module subdirectories. Only Go apps are structured that way.
  # See https://cloud.google.com/appengine/docs/go/#Go_Organizing_Go_apps.
  for subdir in os.listdir(app_dir):
    subdir = os.path.join(app_dir, subdir)
    if not os.path.isdir(subdir):
      continue
    app_yaml = os.path.join(subdir, 'app.yaml')
    if os.path.isfile(app_yaml):
      yamls.append(app_yaml)
    yamls.extend(glob.glob(os.path.join(subdir, 'module-*.yaml')))
  if not yamls:
    raise ValueError(
        'Not a GAE application directory, no module *.yamls found: %s' %
        app_dir)

  # There should be one and only one app.yaml.
  app_yamls = [p for p in yamls if os.path.basename(p) == 'app.yaml']
  if not app_yamls:
    raise ValueError(
        'Not a GAE application directory, no app.yaml found: %s' % app_dir)
  if len(app_yamls) > 1:
    raise ValueError(
        'Not a GAE application directory, multiple app.yaml found (%s): %s' %
        (app_yamls, app_dir))
  return yamls


def is_app_dir(path):
  """Returns True if |path| is structure like GAE app directory."""
  try:
    find_app_yamls(path)
    return True
  except ValueError:
    return False


def get_module_runtime(yaml_path):
  """Finds 'runtime: ...' property in module YAML (or None if missing)."""
  # 'yaml' module is not available yet at this point (it is loaded from SDK).
  with open(yaml_path, 'rt') as f:
    m = re.search(r'^runtime\:\s+(.*)$', f.read(), re.MULTILINE)
  return m.group(1) if m else None


def get_app_runtime(yaml_paths):
  """Examines all app's yamls making sure they specify single runtime.

  Raises:
    ValueError if multiple (or unknown) runtimes are specified.
  """
  runtimes = sorted(set(get_module_runtime(p) for p in yaml_paths))
  if len(runtimes) != 1:
    raise ValueError('Expecting single runtime, got %s' % ', '.join(runtimes))
  if runtimes[0] not in RUNTIME_TO_SDK:
    raise ValueError('Unknown runtime \'%s\' in %s' % (runtimes[0], yaml_paths))
  return runtimes[0]


def setup_gae_sdk(sdk_path):
  """Modifies sys.path and to be able to use Python portion of GAE SDK.

  Once this is called, other functions from this module know where to find GAE
  SDK and any AppEngine included module can be imported. The change is global
  and permanent.
  """
  global _GAE_SDK_PATH
  if _GAE_SDK_PATH:
    raise ValueError('setup_gae_sdk was already called.')
  _GAE_SDK_PATH = sdk_path

  sys.path.insert(0, sdk_path)
  # Sadly, coverage may inject google.protobuf in the path. Forcibly expulse it.
  if 'google' in sys.modules:
    del sys.modules['google']

  import dev_appserver
  dev_appserver.fix_sys_path()
  for i in sys.path[:]:
    if 'jinja2-2.6' in i:
      sys.path.remove(i)

  # Make 'yaml' variable (defined on top of this module) point to loaded module.
  global yaml
  import yaml as yaml_module
  yaml = yaml_module


ModuleFile = collections.namedtuple('ModuleFile', ['path', 'data'])


class Application(object):
  """Configurable GAE application.

  Can be used to query and change GAE application configuration (default
  serving version, uploaded versions, etc.). Built on top of appcfg.py calls.
  """

  def __init__(self, app_dir, app_id=None, verbose=False):
    """Args:
      app_dir: application directory (should contain app.yaml).
      app_id: application ID to use, or None to use one from app.yaml.
      verbose: if True will run all appcfg.py operations in verbose mode.
    """
    if not _GAE_SDK_PATH:
      raise ValueError('Call setup_gae_sdk first')

    self._gae_sdk = _GAE_SDK_PATH
    self._app_dir = os.path.abspath(app_dir)
    self._app_id = app_id
    self._verbose = verbose

    # Module ID -> (path to YAML, deserialized content of module YAML).
    self._modules = {}
    for yaml_path in find_app_yamls(self._app_dir):
      with open(yaml_path) as f:
        data = yaml.load(f)
        module_id = data.get('service', data.get('module', 'default'))
        if module_id in self._modules:
          raise ValueError(
              'Multiple *.yaml files define same module %s: %s and %s' %
              (module_id, yaml_path, self._modules[module_id].path))
        self._modules[module_id] = ModuleFile(yaml_path, data)

    self.dispatch_yaml = os.path.join(app_dir, 'dispatch.yaml')
    if not os.path.isfile(self.dispatch_yaml):
      self.dispatch_yaml = None

    if 'default' not in self._modules:
      raise ValueError('Default module is missing')
    if not self.app_id:
      raise ValueError('application id is neither specified in default module, '
                       'nor provided explicitly')

  @property
  def app_dir(self):
    """Absolute path to application directory."""
    return self._app_dir

  @property
  def app_id(self):
    """Application ID as passed to constructor, or as read from app.yaml."""
    return self._app_id or self._modules['default'].data.get('application')

  @property
  def modules(self):
    """List of module IDs that this application contain."""
    return self._modules.keys()

  @property
  def module_yamls(self):
    """List of paths to all module YAMLs (include app.yaml as a first item)."""
    # app.yaml first (correspond to 'default' module), then everything else.
    yamls = self._modules.copy()
    return [yamls.pop('default').path] + [m.path for m in yamls.itervalues()]

  @property
  def default_module_dir(self):
    """Absolute path to a directory with app.yaml of the default module.

    It's different from app_dir for Go apps. dev_appserver.py searches for
    cron.yaml, index.yaml etc in this directory.
    """
    return os.path.dirname(self._modules['default'].path)

  def run_cmd(self, cmd, cwd=None):
    """Runs subprocess, capturing the output.

    Doesn't close stdin, since gcloud may be asking for user input. If this is
    undesirable (e.g when gae.py is used from scripts), close 'stdin' of gae.py
    process itself.
    """
    logging.debug('Running %s', cmd)
    proc = subprocess.Popen(
        cmd,
        cwd=cwd or self._app_dir,
        stdout=subprocess.PIPE)
    output, _ = proc.communicate()
    if proc.returncode:
      sys.stderr.write('\n' + output + '\n')
      raise subprocess.CalledProcessError(proc.returncode, cmd, output)
    return output

  def run_appcfg(self, args):
    """Runs appcfg.py <args>, deserializes its output and returns it."""
    if USE_GCLOUD:
      raise Error('Attempting to run appcfg.py %s' % ' '.join(args))
    if not is_appcfg_oauth_token_cached():
      raise LoginRequiredError('Login first using \'gae.py appcfg_login\'.')
    cmd = [
      sys.executable,
      os.path.join(self._gae_sdk, 'appcfg.py'),
      '--skip_sdk_update_check',
      '--application', self.app_id,
    ]
    if self._verbose:
      cmd.append('--verbose')
    cmd.extend(args)
    return yaml.safe_load(self.run_cmd(cmd))

  def run_gcloud(self, args):
    """Runs 'gcloud <args> --project ... --format ...' and parses the output."""
    gcloud = find_gcloud()
    if not is_gcloud_auth_set():
      raise LoginRequiredError('Login first using \'gcloud auth login\'')
    raw = self.run_cmd(
        [gcloud] + args + ['--project', self.app_id, '--format', 'json'])
    try:
      return json.loads(raw)
    except ValueError:
      sys.stderr.write('Failed to decode gcloud output %r as JSON\n' % raw)
      raise

  def list_versions(self):
    """List all uploaded versions.

    Returns:
      Dict {module name -> [list of uploaded versions]}.
    """
    if not USE_GCLOUD:
      return self.run_appcfg(['list_versions'])
    data = self.run_gcloud(['app', 'versions', 'list'])
    per_module = collections.defaultdict(list)
    for deployment in data:
      service = deployment['service'].encode('utf-8')
      version_id = deployment['id'].encode('utf-8')
      per_module[service].append(version_id)
    return dict(per_module)

  def set_default_version(self, version, modules=None):
    """Switches default version of given |modules| to |version|."""
    if not USE_GCLOUD:
      self.run_appcfg([
        'set_default_version',
        '--module', ','.join(sorted(modules or self.modules)),
        '--version', version,
      ])
      return

    # There's 'versions migrate' command. Unfortunately it requires to enable
    # warmup requests for all modules if at least one module have them, which is
    # very inconvenient. Use 'services set-traffic' instead that is free of this
    # weird restriction. If a gradual traffic migration is desired, users can
    # click buttons in Cloud Console.
    for m in sorted(modules or self.modules):
      self.run_gcloud([
        'app', 'services', 'set-traffic',
        m, '--splits', '%s=1' % version,
        '--quiet'
      ])

  def delete_version(self, version, modules=None):
    """Deletes the specified version of the given module names."""
    if not USE_GCLOUD:
      # For some reason 'delete_version' call processes only one module at
      # a time, unlike all other related appcfg.py calls.
      for module in sorted(modules or self.modules):
        self.run_appcfg([
          'delete_version',
          '--module', module,
          '--version', version,
        ])
      return

    # If --service is not specified, gcloud deletes the version from all
    # modules. That's what we want if modules is None. --quiet is needed to
    # skip "Do you want to continue?". We've already asked in gae.py.
    if modules is None:
      self.run_gcloud(['app', 'versions', 'delete', version, '--quiet'])
    else:
      # Otherwise delete service-by-service.
      for m in sorted(modules):
        self.run_gcloud([
          'app', 'versions', 'delete', version, '--service', m, '--quiet'
        ])

  def update(self, version, modules=None):
    """Deploys new version of the given module names.

    Supports only GAE Standard currently.
    """
    mods = []
    try:
      for m in sorted(modules or self.modules):
        mod = self._modules[m]
        if mod.data.get('vm') and not USE_GCLOUD:
          raise UnsupportedModuleError(
              'MVM is only supported in gcloud mode: %s' % m)
        if mod.data.get('env') == 'flex' and not USE_GCLOUD:
          raise UnsupportedModuleError(
              'Flex is only supported in gcloud mode: %s' % m)
        if mod.data.get('runtime') == 'go' and not os.environ.get('GOROOT'):
          raise BadEnvironmentError('GOROOT must be set when deploying Go app')
        mods.append(mod)
    except KeyError as e:
      raise ValueError('Unknown module: %s' % e)

    # Always make 'default' the first module to be uploaded. It is magical,
    # deploying it first "enables" the application, or so it seems.
    mods.sort(key=lambda x: '' if x == 'default' else x)

    if not USE_GCLOUD:
      self.run_appcfg(
          ['update'] + [m.path for m in mods] + ['--version', version])
      self._appcfg_update_indexes()
      self._appcfg_update_queues()
      self._appcfg_update_cron()
      self._appcfg_update_dispatch()
      return

    # Will contain paths to module YAMLs and to all extra YAMLs, like cron.yaml.
    yamls = []

    # 'gcloud' barfs at 'application' and 'version' fields in app.yaml. Hack
    # them away. Eventually all app.yaml must be updated to not specify
    # 'application' or 'version'.
    hacked = []
    for m in mods:
      stripped = m.data.copy()
      stripped.pop('application', None)
      stripped.pop('version', None)
      if stripped == m.data:
        yamls.append(m.path)  # the original YAML is good enough
      else:
        # Need to write a hacked version, in same directory, so all paths are
        # relative.
        logging.error(
            'Please remove "application" and "version" keys from %s', m.path)
        fname = os.path.basename(m.path)
        hacked_path = os.path.join(os.path.dirname(m.path), '._gae_py_' + fname)
        with open(hacked_path, 'w') as f:
          json.dump(stripped, f)  # JSON is YAML, so whatever
        yamls.append(hacked_path)
        hacked.append(hacked_path)  # to know what to delete later

    # Deploy all other stuff too. 'app deploy' is a polyglot.
    possible_extra = [
      os.path.join(self.default_module_dir, 'index.yaml'),
      os.path.join(self.default_module_dir, 'queue.yaml'),
      os.path.join(self.default_module_dir, 'cron.yaml'),
      os.path.join(self.default_module_dir, 'dispatch.yaml'),
    ]
    for extra in possible_extra:
      if extra and os.path.isfile(extra):
        yamls.append(extra)

    try:
      self.run_gcloud(
          ['app', 'deploy'] + yamls +
          [
            '--version', version, '--quiet',
            '--no-promote', '--no-stop-previous-version',
          ])
    finally:
      for h in hacked:
        os.remove(h)

  def _appcfg_update_indexes(self):
    """Deploys new index.yaml."""
    assert not USE_GCLOUD
    if os.path.isfile(os.path.join(self.default_module_dir, 'index.yaml')):
      self.run_appcfg(['update_indexes', self.default_module_dir])

  def _appcfg_update_queues(self):
    """Deploys new queue.yaml."""
    assert not USE_GCLOUD
    if os.path.isfile(os.path.join(self.default_module_dir, 'queue.yaml')):
      self.run_appcfg(['update_queues', self.default_module_dir])

  def _appcfg_update_cron(self):
    """Deploys new cron.yaml."""
    assert not USE_GCLOUD
    if os.path.isfile(os.path.join(self.default_module_dir, 'cron.yaml')):
      self.run_appcfg(['update_cron', self.default_module_dir])

  def _appcfg_update_dispatch(self):
    """Deploys new dispatch.yaml."""
    assert not USE_GCLOUD
    if self.dispatch_yaml:
      self.run_appcfg(['update_dispatch', self.app_dir])

  def spawn_dev_appserver(self, args, open_ports=False, **kwargs):
    """Launches subprocess with dev_appserver.py.

    Args:
      args: extra arguments to dev_appserver.py.
      open_ports: if True will bind TCP ports to 0.0.0.0 interface.
      kwargs: passed as is to subprocess.Popen.

    Returns:
      Instance of subprocess.Popen.
    """
    cmd = [
      sys.executable,
      os.path.join(self._gae_sdk, 'dev_appserver.py'),
      '--application', self.app_id,
      '--skip_sdk_update_check=yes',
      '--require_indexes=yes',
    ] + self.module_yamls
    if self.dispatch_yaml:
      cmd += [self.dispatch_yaml]
    cmd += args
    if open_ports:
      cmd.extend(('--host', '0.0.0.0', '--admin_host', '0.0.0.0'))
    if self._verbose:
      cmd.extend(('--log_level', 'debug'))
    return subprocess.Popen(cmd, cwd=self.app_dir, **kwargs)

  def run_dev_appserver(self, args, open_ports=False):
    """Runs the application locally via dev_appserver.py.

    Args:
      args: extra arguments to dev_appserver.py.
      open_ports: if True will bind TCP ports to 0.0.0.0 interface.

    Returns:
      dev_appserver.py exit code.
    """
    return self.spawn_dev_appserver(args, open_ports).wait()

  def get_uploaded_versions(self, modules=None):
    """Returns list of versions that are deployed to all given |modules|.

    If a version is deployed only to one module, it won't be listed. Versions
    are sorted by a version number, oldest first.
    """
    # Build a mapping: version -> list of modules that have it.
    versions = collections.defaultdict(list)
    for module, version_list in self.list_versions().iteritems():
      for version in version_list:
        versions[version].append(module)

    # Keep only versions that are deployed to all requested modules.
    modules = modules or self.modules
    actual_versions = [
      version for version, modules_with_it in versions.iteritems()
      if set(modules_with_it).issuperset(modules)
    ]

    # Sort by version number (best effort, nonconforming version names will
    # appear first in the list).
    def extract_version_num(version):
      parts = version.split('-', 1)
      try:
        parts[0] = int(parts[0])
      except ValueError:
        pass
      return tuple(parts)
    return sorted(actual_versions, key=extract_version_num)

  def get_actives(self, modules=None):
    """Returns active version(s)."""
    data = self.run_gcloud(['app', 'versions', 'list', '--hide-no-traffic'])
    # TODO(maruel): Handle when traffic_split != 1.0.
    # TODO(maruel): There's a lot more data, decide what is generally useful in
    # there.
    return [
      {
        'creationTime': service['version']['createTime'],
        'deployer': service['version']['createdBy'],
        'id': service['id'],
        'service': service['service'],
      } for service in data if not modules or service['service'] in modules
    ]


def setup_env(app_dir, app_id, version, module_id, remote_api=False):
  """Setups os.environ so GAE code works.

  Must be called only after SDK path has been initialized with setup_gae_sdk.
  """
  # GCS library behaves differently when running under remote_api. It uses
  # SERVER_SOFTWARE to figure this out. See cloudstorage/common.py, local_run().
  if remote_api:
    os.environ['SERVER_SOFTWARE'] = 'remote_api'
  else:
    os.environ['SERVER_SOFTWARE'] = 'Development yo dawg/1.0'
  if app_dir:
    app_id = app_id or Application(app_dir).app_id
    version = version or 'default-version'
  if app_id:
    os.environ['APPLICATION_ID'] = app_id
  if version:
    os.environ['CURRENT_VERSION_ID'] = '%s.%d' % (
        version, int(time.time()) << 28)
  if module_id:
    os.environ['CURRENT_MODULE_ID'] = module_id


def add_sdk_options(parser, default_app_dir):
  """Adds common command line options used by tools that wrap GAE SDK.

  Args:
    parser: OptionParser to add options to.
    default_app_dir: default value for --app-dir option.
  """
  parser.add_option(
      '-s', '--sdk-path',
      help='Path to AppEngine SDK. Will try to find by itself.')
  parser.add_option(
      '-p', '--app-dir',
      default=default_app_dir,
      help='Path to application directory with app.yaml.')
  parser.add_option('-A', '--app-id', help='Defaults to name in app.yaml.')
  parser.add_option('-v', '--verbose', action='store_true')


def process_sdk_options(parser, options):
  """Handles values of options added by 'add_sdk_options'.

  Modifies global process state by configuring logging and path to GAE SDK.

  Args:
    parser: OptionParser instance to use to report errors.
    options: parsed options, as returned by parser.parse_args.

  Returns:
    New instance of Application configured based on passed options.
  """
  logging.basicConfig(level=logging.DEBUG if options.verbose else logging.ERROR)

  if not options.app_dir:
    parser.error('--app-dir option is required')
  app_dir = os.path.abspath(options.app_dir)

  try:
    runtime = get_app_runtime(find_app_yamls(app_dir))
  except (Error, ValueError) as exc:
    parser.error(str(exc))

  sdk_path = options.sdk_path or find_gae_sdk(RUNTIME_TO_SDK[runtime], app_dir)
  if not sdk_path:
    parser.error('Failed to find the AppEngine SDK. Pass --sdk-path argument.')

  setup_gae_sdk(sdk_path)

  try:
    return Application(app_dir, options.app_id, options.verbose)
  except (Error, ValueError) as e:
    parser.error(str(e))


def confirm(text, app, version, modules=None, default_yes=False):
  """Asks a user to confirm the action related to GAE app.

  Args:
    text: actual text of the prompt.
    app: instance of Application.
    version: version or a list of versions to operate upon.
    modules: list of modules to operate upon (or None for all).

  Returns:
    True on approval, False otherwise.
  """
  print(text)
  print('  Directory: %s' % os.path.basename(app.app_dir))
  print('  App ID:    %s' % app.app_id)
  print('  Version:   %s' % version)
  print('  Modules:   %s' % ', '.join(modules or app.modules))
  if default_yes:
    return raw_input('Continue? [Y/n] ') not in ('n', 'N')
  else:
    return raw_input('Continue? [y/N] ') in ('y', 'Y')


def is_gcloud_auth_set():
  """Returns false if 'gcloud auth login' needs to be run."""
  try:
    # This returns an email address of currently active account or empty string
    # if no account is active.
    output = subprocess.check_output([
      find_gcloud(), 'auth', 'list',
      '--filter=status:ACTIVE', '--format=value(account)',
    ])
    return bool(output.strip())
  except subprocess.CalledProcessError as exc:
    logging.error('Failed to check active gcloud account: %s', exc)
    return False


# TODO(vadimsh): Can be removed if using 'gcloud'.
def _appcfg_oauth2_tokens():
  return os.path.join(os.path.expanduser('~'), '.appcfg_oauth2_tokens')


# TODO(vadimsh): Can be removed if using 'gcloud'.
def is_appcfg_oauth_token_cached():
  """Returns true if ~/.appcfg_oauth2_tokens exists."""
  return os.path.exists(_appcfg_oauth2_tokens())


# TODO(vadimsh): Can be removed if using 'gcloud'.
def appcfg_login(app):
  """Starts appcfg.py's login flow."""
  if not _GAE_SDK_PATH:
    raise ValueError('Call setup_gae_sdk first')
  if os.path.exists(_appcfg_oauth2_tokens()):
    os.remove(_appcfg_oauth2_tokens())
  # HACK: Call a command with no side effect to launch the flow.
  subprocess.call([
    sys.executable,
    os.path.join(_GAE_SDK_PATH, 'appcfg.py'),
    '--application', app.app_id,
    '--noauth_local_webserver',
    'list_versions',
  ], cwd=app.app_dir)


def setup_gae_env():
  """Sets up App Engine Python test environment by modifying sys.path."""
  sdk_path = find_gae_sdk()
  if not sdk_path:
    raise BadEnvironmentError('Couldn\'t find GAE SDK.')
  setup_gae_sdk(sdk_path)
