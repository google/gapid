#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Wrapper around GKE (Google Container Engine) SDK tools to simplify working
with container deployments.

Core to this is a services configuration YAML file. This lists all of the
GKE services that get deployed. A file declares clusters, each of which
is a single GKE pod deployment. An example YAML follows:

project: my-project
clusters:
  my-service:
    path: relpath/to/Dockerfile/directory
    name: gke-cluster-name
    zone: cluster-zone
    replicas: number-of-replicas
    collect_gopath: _gopath

Breaking this down, this declares that the YAML applies to the Google cloud
project called "my-project". It defines one microservice, "my-service". The
service is rooted at the specified relative path and uses the GKE cluster
named "gke-cluster-name" in zone "cluster-zone".

When deployed, the cluster will request "number-of-replicas" replicas.

Finally, we collect GOPATH in a subdirectory "_gopath" underneath of "path".
This means that all source folders in GOPATH with ".go" files in them are
copied (hardlink, so fast) into that directory. This is necessary so Docker
can deploy against the local checkout of those files instead of using "go get"
to fetch them from HEAD.

We use deployment annotations to track deployment status:
- luci.managedBy
  This is set to "luci-gke-py" to assert that a given image is managed by
  this "gke.py" tool.
- luci-gke-py/stable-image
  This is set to be the image name of the last stable image. This can be
  updated via "commit" to the current deployed image.
- luci-gke-py/latest-image
  This is set to be the name of the latest image that was built.

Our general command sequences will be:

UPLOAD:
  - Build and push the new, unique "gcr.io" image.
  - Set "latest-image" tag on the deployment to point to the image. This
    doesn't change deployment, but marks the last built image for a future
    command.
  - If the deployment doesn't already exist (first time only), a zero-replica
    initial deployment will be created.

SWITCH:
  - Set the deployment's image to the image marked as "latest-image".
  - Uses "scale" to set the number of replicas.

ROLLBACK:
  - Issues a straightforward "kubectl rollout undo" command to revert to the
    previous deployment configuration.

PROMOTE:
  - Promotes the "latest-image" tag to "stable-image".
"""

__version__ = '1.0'

import argparse
import collections
import contextlib
import logging
import os
import shutil
import subprocess
import sys
import tempfile
import time

SCRIPT_PATH = os.path.abspath(__file__)
ROOT_DIR = os.path.dirname(os.path.dirname(SCRIPT_PATH))
sys.path.insert(0, ROOT_DIR)
sys.path.insert(0, os.path.join(ROOT_DIR, '..', 'third_party_local'))

# Load and import the AppEngine SDK environment.
from tools import calculate_version
from tool_support import gae_sdk_utils
gae_sdk_utils.setup_gae_env()

import yaml


# True if running on Windows.
IS_WINDOWS = os.name == 'nt'


# Enable "yaml" to dump ordered dict.
def dict_representer(dumper, data):
  return dumper.represent_dict((k, v) for k, v in data.iteritems()
                               if v is not None)
yaml.Dumper.add_representer(collections.OrderedDict, dict_representer)


def run_command(cmd, cwd=None):
  logging.debug('Running command: %s', cmd)
  return subprocess.call(
      cmd,
      stdin=sys.stdin,
      stdout=sys.stdout,
      stderr=sys.stderr,
      cwd=cwd)


def check_output(cmd, cwd=None):
  logging.debug('Running command (cwd=%r): %s', cwd, cmd)
  stdout = subprocess.check_output(cmd, cwd=cwd)
  logging.debug('Command returned with output:\n%s', stdout)
  return stdout.strip()


def is_same_fs(a, b):
  dev_a = os.stat(a).st_dev
  dev_b = os.stat(b).st_dev
  return dev_a == dev_b


def collect_gopath(base, name, gopath):
  dst = os.path.join(base, name)
  logging.debug('Collecting GOPATH from %r into %r.', gopath, dst)
  if os.path.exists(dst):
    shutil.rmtree(dst)
  os.makedirs(dst)

  # Our goal is to recursively copy ONLY Go directories. We detect these as
  # directories with at least one ".go" file in them. Any other directory, we
  # ignore.
  #
  # We'll also hardlink the actual files so we don't waste space.
  #
  # Unfortunately, there's not really a good OS or Python facility to do this.
  # It's simple enough, though, so we'll just do it ourselves.
  #
  # Also, in true emulation of GOPATH, we want any earlier Go directories to
  # override later ones. We'll track Go directories (relative) and not copy
  # other directories if they happen to have duplicates.
  seen = set()
  for p in gopath.split(os.pathsep):
    go_src = os.path.join(p, 'src')
    if not os.path.isdir(go_src):
      continue

    for root, dirs, files in os.walk(go_src):
      # Do not recurse into our destination directory, which is likely itself
      # on GOPATH.
      if os.path.samefile(root, dst):
        del(dirs[:])
        continue
      if not any(f.endswith('.go') for f in files):
        continue

      # Determine if we're on the same filesystem. If we are, we can make this
      # really fast by using hardlinks.
      link_cmd = (os.link) if is_same_fs(root, dst) else shutil.copy2

      rel = os.path.relpath(root, go_src)
      if rel in seen:
        continue
      seen.add(rel)

      rel_dst = os.path.join(dst, 'src', rel)
      if not os.path.isdir(rel_dst):
        os.makedirs(rel_dst)
      logging.debug('Copying Go source %r => %r', root, rel_dst)
      for f in files:
        link_cmd(os.path.join(root, f), os.path.join(rel_dst, f))


@contextlib.contextmanager
def tempdir():
  path = None
  try:
    path = tempfile.mkdtemp(dir=os.getcwd())
    yield path
  finally:
    if path is not None:
      shutil.rmtree(path)


class Configuration(object):
  """Cluster is the configuration for a single GKE application.
  """

  # Config is the configuration schema.
  Config = collections.namedtuple('Config', (
      'project', 'clusters',
  ))
  Cluster = collections.namedtuple('Cluster', (
      'name', 'path', 'zone', 'replicas', 'collect_gopath'))

  def __init__(self, config):
    self.config = config

  def __getattr__(self, key):
    return getattr(self.config, key)

  @staticmethod
  def load_with_defaults(typ, d):
    inst = typ(**{v: None for v in typ._fields})
    inst = inst._replace(**d)
    return inst

  @classmethod
  def write_template(cls, path):
    # Generate a template config and write it.
    cfg = cls.Config(
      project='project',
      clusters={
        'cluster-key': cls.Cluster(
          name='cluster-name',
          zone='cluster-zone',
          path='relpath/to/docker/root',
          replicas=1,
          collect_gopath=None,
        )._asdict(),
      },
    )
    with open(path, 'w') as fd:
      yaml.dump(cfg._asdict(), fd, default_flow_style=False)

  @classmethod
  def load(cls, path):
    if not os.path.isfile(path):
      cls.write_template(path)
      raise ValueError('Missing configuration path. A template was generated '
                       'at: %r' % (path,))

    with open(path, 'r') as fd:
      d = yaml.load(fd)

    # Load the JSON into our namedtuple schema.
    cfg = cls.load_with_defaults(cls.Config, d)
    cfg = cfg._replace(clusters={k: cls.load_with_defaults(cls.Cluster, v)
                                 for k, v in cfg.clusters.iteritems()})

    return cls(cfg)


class Application(object):
  """Cluster is the configuration for a single GKE application.

  This includes the application's name, project, deployment parameters.

  An application consists of:
  - A path to its root Dockerfile.
  - Deployment cluster parameters (project, zone, etc.), loaded from JSON.
  """

  def __init__(self, config_dir, config, cluster_key):
    self.timestamp = int(time.time())
    self.config_dir = config_dir
    self.cfg = config
    self.cluster_key = cluster_key

  @property
  def project(self):
    return self.cfg.project
  @property
  def cluster(self):
    return self.cfg.clusters[self.cluster_key]
  @property
  def name(self):
    return self.cluster.name
  @property
  def zone(self):
    return self.cluster.zone
  @property
  def app_dir(self):
    return os.path.join(self.config_dir, self.cluster.path)

  def ensure_dockerfile(self):
    dockerfile_path = os.path.join(self.app_dir, 'Dockerfile')
    if not os.path.isfile(dockerfile_path):
      raise ValueError('No Dockerfile at %r.' % (dockerfile_path,))
    return dockerfile_path

  @property
  def kubectl_context(self):
    """Returns the name of the "kubectl" context for this Application.

    This name is a "gcloud" implementation detail, and may change in the future.
    Knowing the name, we can check for credential provisioning locally, reducing
    the common case overhead of individual "kubectl" invocations.
    """
    return 'gke_%s_%s_%s' % (self.project, self.zone, self.name)

  def calculate_version(self, tag=None):
    return calculate_version.calculate_version(self.app_dir, tag)

  def image_tag(self, version):
    # We add a timestamp b/c otherwise Docker can't distinguish two images with
    # the same tag, and won't update.
    return 'gcr.io/%s/%s:%s-%d' % (
        self.project, self.name, version, self.timestamp)

  @property
  def deployment_name(self):
    return self.cluster_key

  @property
  def container_name(self):
    return self.cluster_key


class Kubectl(object):
  """Wrapper around the "kubectl" tool.
  """

  _ANNOTATION_MANAGED_BY = 'luci.managedBy'
  _ANNOTATION_MANAGED_BY_ME = 'luci-gke-py'
  _ANNOTATION_STABLE = 'luci-gke-py/stable-image'
  _ANNOTATION_LATEST = 'luci-gke-py/latest-image'

  def __init__(self, app, needs_refresh):
    self.app = app
    self._needs_refresh = needs_refresh
    self._verified_app_context = False

  @property
  def executable(self):
    return 'kubectl'

  def run(self, cmd, **kwargs):
    args = [
        self.executable,
        '--context', self.ensure_app_context(),
    ]
    args.extend(cmd)
    return run_command(args, **kwargs)

  def check_output(self, cmd, **kwargs):
    args = [
        self.executable,
        '--context', self.ensure_app_context(),
    ]
    args.extend(cmd)
    return check_output(args, **kwargs)

  def check_gcloud(self, cmd, **kwargs):
    args = [
        'gcloud',
        '--project', self.app.project,
    ]
    args.extend(cmd)
    return check_output(args, **kwargs)

  @staticmethod
  def _strip_quoted_output(v):
    if v.startswith("'") and v.endswith("'"):
      v = v[1:-1]
    return v

  def is_deployed(self):
    # Query for deployments w/ the specified name. If an empty string is
    # returned, we are not deployed.
    query = '{.items[?(@.metadata.name == "%s")].metadata.name}' % (
        self.app.deployment_name,)
    output = self.check_output([
        'get',
        'deployments',
        '--output', "jsonpath='%s'" % (query,),
    ])
    output = self._strip_quoted_output(output)
    return bool(output)

  def get_deployment_annotation(self, name):
    output = self.check_output([
        'get',
        'deployments/%s' % (self.app.deployment_name,),
        '--output', 'jsonpath=\'{.metadata.annotations.%s}\'' % (name,),
    ])
    return self._strip_quoted_output(output)

  def set_deployment_annotation(self, name, value):
    self.run([
      'annotate',
      '--overwrite',
      'deployments',
      self.app.deployment_name,
      "%s=%s" % (name, value),
    ])

  def get_latest_image(self):
    return self.get_deployment_annotation(self._ANNOTATION_LATEST)

  def set_latest_image(self, image_tag):
    self.set_deployment_annotation(self._ANNOTATION_LATEST, image_tag)

  def get_stable_image(self):
    return self.get_deployment_annotation(self._ANNOTATION_STABLE)

  def set_stable_image(self, image_tag):
    self.set_deployment_annotation(self._ANNOTATION_STABLE, image_tag)

  def describe_deployment(self):
    print self.check_output([
        'describe',
        'deployments/%s' % (self.app.deployment_name,)])

  def maybe_push_new_deployment(self, image_tag):
    # If we're not deployed, push a brand-new deployment YAML.
    if self.is_deployed():
      return False

    logging.info('No current deployment, creating a new one...')
    with tempdir() as tdir:
      deployment_yaml_path = os.path.join(tdir, 'deployment.yaml')
      self.write_deployment_yaml(deployment_yaml_path, image_tag)

      # Deploy to Kubernetes.
      self.run(['apply', '-f', deployment_yaml_path])
    return True

  def set_deployment_image(self, image_tag):
    # Set the deployment's image to the new image version.
    self.run([
      'set',
      'image',
      'deployments/%s' % (self.app.deployment_name,),
      '%s=%s' % (self.app.container_name, image_tag),
    ])

  def undo_rollout(self):
    """Issues a simple "kubectl rollback" command against the deployment."""
    self.run(['rollout', 'undo'])

  def scale_replicas(self, count):
    self.run([
      'scale',
      'deployments/%s' % (self.app.deployment_name,),
      '--replicas=%d' % (count,),
    ])

  def _has_app_context(self):
    if self._verified_app_context:
      return True

    # If this command returns non-zero and has non-empty output, we know that
    # the context is available.
    logging.debug('Checking if "gcloud" credentials are available.')
    stdout = check_output([
          self.executable,
          'config',
          'view',
          '--output', 'jsonpath=\'{.users[?(@.name == "%s")].name}\'' % (
              self.app.kubectl_context,),
        ],
    )
    self._verified_app_context = bool(self._strip_quoted_output(stdout))
    return self._verified_app_context

  def ensure_app_context(self):
    """Sets the current "kubectl" context to point to the current application.

    Kubectl can contain multiple context configurations. We want to explicitly
    specify the context each time we execute a command.

    Worst-case, we need to use "gcloud" to provision the context, which includes
    round-trips from remote services. Best-case, we're already provisioned with
    the context and can just use it.

    Returns (str): The name of the Kubernetes context, suitable for passing
      to the "--context" command-line parameter.
    """
    if self._needs_refresh or not self._has_app_context():
      self.check_gcloud([
          'container',
          'clusters',
          'get-credentials',
          self.app.name,
          '--zone', self.app.zone,
      ])
      self._needs_refresh = False
      if not self._has_app_context():
        raise Exception('Kubernetes context missing after provisioning.')
    return self.app.kubectl_context

  def write_deployment_yaml(self, path, image_tag):
    v = collections.OrderedDict()
    v['apiVersion'] = 'apps/v1beta1'
    v['kind'] = 'Deployment'
    v['metadata'] = collections.OrderedDict((
        ('name', self.app.deployment_name),
        ('annotations', collections.OrderedDict((
          (self._ANNOTATION_MANAGED_BY, self._ANNOTATION_MANAGED_BY_ME),
          (self._ANNOTATION_STABLE, image_tag),
          (self._ANNOTATION_LATEST, image_tag),
        ))),
    ))

    v['spec'] = collections.OrderedDict()
    v['spec']['replicas'] = 0

    # Set the stategy to "Recreate". This means that, rather than perform a
    # rolling update, old repica sets are deleted prior to new ones being
    # instantiated.
    v['spec']['strategy'] = collections.OrderedDict()
    v['spec']['strategy']['type'] = 'Recreate'

    v['spec']['template'] = collections.OrderedDict()
    v['spec']['template']['metadata'] = collections.OrderedDict()
    v['spec']['template']['metadata']['labels'] = collections.OrderedDict((
        ('luci/project', self.app.project),
        ('luci/cluster', self.app.cluster_key),
    ))

    v['spec']['template']['spec'] = collections.OrderedDict()
    v['spec']['template']['spec']['containers'] = [
        collections.OrderedDict((
            ('name', self.app.container_name),
            ('image', image_tag),
        )),
    ]

    with open(path, 'w') as fd:
      yaml.dump(v, fd, default_flow_style=False)


def subcommand_kubectl(args, kctl):
  """Runs a Kubernetes command in the context of the configured Application.
  """
  cmd = args.args
  if cmd and cmd[0] == '--':
    cmd = cmd[1:]
  return kctl.run(cmd)


def subcommand_upload(args, kctl):
  """Deploys a Kubernetes instance."""

  # Determine our version and Docker image tag.
  version = kctl.app.calculate_version(tag=args.tag)
  image_tag = kctl.app.image_tag(version)

  # If we need to collect GOPATH, do it.
  if kctl.app.cluster.collect_gopath:
    collect_gopath(
        kctl.app.app_dir,
        kctl.app.cluster.collect_gopath,
        os.environ.get('GOPATH', ''))

  # Build our Docker image.
  kctl.check_gcloud(
      ['docker', '--', 'build', '-t', image_tag, '.'],
      cwd=kctl.app.app_dir)

  # Push the Docker image to the project's registry.
  kctl.check_gcloud(
      ['docker', '--', 'push', image_tag])

  # If the deployment doesn't exist, deploy it now.
  if not kctl.maybe_push_new_deployment(image_tag):
    # Not a new deployment, so just set the "latest" annotation.
    logging.debug('Setting latest image tag to: %r', image_tag)
    kctl.set_latest_image(image_tag)
  return 0


def subcommand_switch(_args, kctl):
  # If the deployment doesn't exist, deploy it now.
  latest = kctl.get_latest_image()
  logging.info('Switching image to: %r', latest)
  kctl.set_deployment_image(latest)

  logging.info('Adjusting replicas to: %d', kctl.app.cluster.replicas)
  kctl.scale_replicas(kctl.app.cluster.replicas)
  return 0


def subcommand_rollback(args, kctl):
  if args.stable:
    # If the deployment doesn't exist, deploy it now.
    stable = kctl.get_stable_image()
    logging.info('Rolling back to stable image: %r', stable)
    kctl.set_deployment_image(stable)
    return 0

  kctl.undo_rollout()
  return 0


def subcommand_promote(_args, kctl):
  latest = kctl.get_latest_image()
  stable = kctl.get_stable_image()
  logging.info('Promoting stable image: %r => %r', stable, latest)
  kctl.set_stable_image(latest)
  return 0


def subcommand_status(_args, kctl):
  kctl.describe_deployment()

  stable = kctl.get_stable_image()
  print 'Stable image:', stable

  latest = kctl.get_latest_image()
  print 'Latest image:', latest

  if stable != latest:
    print 'NOTE: Stable and latest do not match. Run "promote" to commit.'
  else:
    print 'Stable and latest images match.'

  return 0


def main(args):
  if IS_WINDOWS:
    logging.error('This script does not currently work on Windows.')
    return 1

  parser = argparse.ArgumentParser()
  parser.add_argument(
      '-v', '--verbose',
      action='count',
      default=0,
      help='Increase logging verbosity. Can be specified multiple times.')
  parser.add_argument(
      '-r', '--force-refresh',
      action='store_true',
      help='Forcefully refresh GKE authentication.')
  parser.add_argument(
      '-C', '--config', required=True,
      help='Path to the cluster configuration JSON file.')
  subparsers = parser.add_subparsers()

  def add_cluster_key(subparser):
    subparser.add_argument(
        '-K', '--cluster-key', required=True,
        help='Key of the cluster within the config to work with.')

  # Subcommand: kubectl
  subparser = subparsers.add_parser('kubectl',
      help='Direct invocation of "kubectl" command using target context.')
  subparser.add_argument('args', nargs=argparse.REMAINDER,
      help='Arguments to pass to the "kubectl" invocation.')
  add_cluster_key(subparser)
  subparser.set_defaults(func=subcommand_kubectl)

  # Subcommand: upload
  subparser = subparsers.add_parser('upload',
      help='Build and upload a new instance to a Kubernetes cluster.')
  subparser.add_argument('-t', '--tag',
      help='Optional tag to add to the version.')
  add_cluster_key(subparser)
  subparser.set_defaults(func=subcommand_upload)

  # Subcommand: switch
  subparser = subparsers.add_parser('switch',
      help='Switch the current image to the latest uploaded image.')
  add_cluster_key(subparser)
  subparser.set_defaults(func=subcommand_switch)

  # Subcommand: rollback
  subparser = subparsers.add_parser('rollback',
      help='Issue a rollback command to the deployment.')
  subparser.add_argument('--stable', action='store_true',
      help='Rather than issue a rollback command, explicitly set the image '
           'back to the last "stable" tag. This sidesteps standard Kubernetes '
           'rollback, but can be useful if multiple deployments have been '
           'made in between a commit.')
  add_cluster_key(subparser)
  subparser.set_defaults(func=subcommand_rollback)

  # Subcommand: promote
  subparser = subparsers.add_parser('promote',
      help='Promote the latest deployed build to "stable".')
  add_cluster_key(subparser)
  subparser.set_defaults(func=subcommand_promote)

  # Subcommand: status
  subparser = subparsers.add_parser('status',
      help='Display information about the current deployment.')
  add_cluster_key(subparser)
  subparser.set_defaults(func=subcommand_status)

  args = parser.parse_args()

  if args.verbose == 0:
    logging.getLogger().setLevel(logging.WARNING)
  elif args.verbose == 1:
    logging.getLogger().setLevel(logging.INFO)
  else:
    logging.getLogger().setLevel(logging.DEBUG)

  config_path = os.path.abspath(args.config)
  config = Configuration.load(config_path)
  if args.cluster_key not in config.clusters:
    raise ValueError(
        'A cluster key is required (--cluster-key), one of: %s' % (
          ', '.join(sorted(config.clusters.keys())),)
    )

  app = Application(
      os.path.dirname(config_path),
      config,
      args.cluster_key)
  kctl = Kubectl(app, args.force_refresh)
  return args.func(args, kctl)


if __name__ == '__main__':
  logging.basicConfig(level=logging.DEBUG)
  sys.exit(main(sys.argv[1:]))
