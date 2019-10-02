# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Wrapper around GCE REST API."""

import re

from components import net


AUTH_SCOPES = [
    'https://www.googleapis.com/auth/compute',
]

# custom-<# CPUs>-<memory MB>
CUSTOM_MACHINE_TYPE_RE = r'^custom-([0-9]+)-([0-9]+)$'


DISK_TYPES = {
    'local-ssd': {'ssd': True},
    'pd-ssd': {'ssd': True},
    'pd-standard': {'ssd': False},
}


MACHINE_TYPES = {
    'f1-micro':       {'cpus': 1,  'memory': 0.6},
    'g1-small':       {'cpus': 1,  'memory': 1.7},
    'n1-standard-1':  {'cpus': 1,  'memory': 3.75},
    'n1-standard-2':  {'cpus': 2,  'memory': 7.5},
    'n1-standard-4':  {'cpus': 4,  'memory': 15},
    'n1-standard-8':  {'cpus': 8,  'memory': 30},
    'n1-standard-16': {'cpus': 16, 'memory': 60},
    'n1-standard-32': {'cpus': 32, 'memory': 120},
    'n1-highcpu-2':   {'cpus': 2,  'memory': 1.8},
    'n1-highcpu-4':   {'cpus': 4,  'memory': 3.6},
    'n1-highcpu-8':   {'cpus': 8,  'memory': 7.2},
    'n1-highcpu-16':  {'cpus': 16, 'memory': 12.4},
    'n1-highcpu-32':  {'cpus': 32, 'memory': 28.8},
    'n1-highmem-2':   {'cpus': 2,  'memory': 13},
    'n1-highmem-4':   {'cpus': 4,  'memory': 26},
    'n1-highmem-8':   {'cpus': 8,  'memory': 52},
    'n1-highmem-16':  {'cpus': 16, 'memory': 104},
    'n1-highmem-32':  {'cpus': 32, 'memory': 208},
}


# TODO(vadimsh): Add a method to fetch list of available zones and use the
# result together with yield_instances_in_zones to emulate yield_instances. Once
# this is done, we can remove hacky HTTP 400 error handling there.


class Project(object):
  """Wrapper around GCE REST API endpoints for some project."""

  def __init__(self, project_id, service_account_key=None):
    """
    Args:
      project_id: Cloud Project ID.
      service_account_key: auth.ServiceAccountKey to use JSON service account,
          or None to use GAE app's service account.
    """
    assert is_valid_project_id(project_id), project_id
    self._project_id = project_id
    self._service_account_key = service_account_key

  @property
  def project_id(self):
    return self._project_id

  def call_api(
      self,
      endpoint,
      method='GET',
      payload=None,
      params=None,
      deadline=None,
      version='v1',
      service='compute'):
    """Sends JSON request (with retries) to GCE API endpoint.

    Args:
      endpoint: endpoint URL relative to the project URL (e.g. /regions).
      method: HTTP method to use, e.g. GET, POST, PUT.
      payload: object to serialize to JSON and put in request body.
      params: dict with query GET parameters (i.e. ?key=value&key=value).
      deadline: deadline for a single call attempt.
      version: API version to use.
      service: API service to call (compute or replicapool).

    Returns:
      Deserialized JSON response.

    Raises:
      net.Error on errors.
    """
    assert service in ('compute', 'replicapool')
    assert endpoint.startswith('/'), endpoint
    url = 'https://www.googleapis.com/%s/%s/projects/%s%s' % (
        service, version, self._project_id, endpoint)
    return net.json_request(
        url=url,
        method=method,
        payload=payload,
        params=params,
        scopes=AUTH_SCOPES,
        service_account_key=self._service_account_key,
        deadline=30 if deadline is None else deadline)

  def get_instance(self, zone, instance, fields=None):
    """Returns dict with info about an instance or None if no such instance.

    Args:
      zone: name of a zone, e.g. 'us-central1-a'.
      instance: name of an instance, e.g. 'slave123-c4'.
      fields: enumeration of dict fields to fetch (or None for all).

    Returns:
      See https://cloud.google.com/compute/docs/reference/v1/instances#resource.
    """
    assert is_valid_zone(zone), zone
    assert is_valid_instance(instance), instance
    try:
      return self.call_api(
          '/zones/%s/instances/%s' % (zone, instance),
          params={'fields': ','.join(fields)} if fields else None)
    except net.NotFoundError:  # pragma: no cover
      return None

  def yield_instances(self, instance_filter=None):
    """Yields dicts with all project instances across all zones.

    The format of the instance dict is defined here:
      https://cloud.google.com/compute/docs/reference/v1/instances#resource

    Returns instances in all possible states (instance['status'] attribute):
      PROVISIONING
      STAGING
      RUNNING
      STOPPING
      STOPPED
      TERMINATED

    Very slow call (can run for minutes). Should be used only from task queues.

    Args:
      instance_filter: optional filter to apply to instance names when scanning.
    """
    if instance_filter and set("\"\\'").intersection(instance_filter):
      raise ValueError('Invalid instance filter: %s' % instance_filter)
    page_token = None
    while True:
      params = {'maxResults': 250}
      if instance_filter:
        params['filter'] = 'name eq "%s"' % instance_filter
      if page_token:
        params['pageToken'] = page_token
      resp = self.call_api('/aggregated/instances', params=params, deadline=120)
      items = resp.get('items', {})
      for zone in sorted(items):
        for instance in items[zone].get('instances', []):
          yield instance
      page_token = resp.get('nextPageToken')
      if not page_token:
        break

  def yield_instances_in_zone(self, zone, instance_filter=None):
    """Yields dicts with all project instances in the specific zone.

    If the zone doesn't exist, silently yields nothing. We assume non-existing
    zones are empty.

    The format of the instance dict is defined here:
      https://cloud.google.com/compute/docs/reference/v1/instances#resource

    Returns instances in all possible states (instance['status'] attribute):
      PROVISIONING
      STAGING
      RUNNING
      STOPPING
      STOPPED
      TERMINATED

    Very slow call (can run for minutes). Should be used only from task queues.

    Args:
      zone: a zone to list instances in, e.g "us-central1-a".
      instance_filter: optional filter to apply to instance names when scanning.
    """
    if instance_filter and set("\"\\'").intersection(instance_filter):
      raise ValueError('Invalid instance filter: %s' % instance_filter)
    page_token = None
    while True:
      params = {'maxResults': 250}
      if instance_filter:
        params['filter'] = 'name eq "%s"' % instance_filter
      if page_token:
        params['pageToken'] = page_token
      try:
        resp = self.call_api(
            '/zones/%s/instances' % zone, params=params, deadline=120)
      except net.Error as exc:
        if not page_token and exc.status_code == 400:
          return  # no such zone, this is fine...
        raise
      for instance in resp.get('items', []):
        yield instance
      page_token = resp.get('nextPageToken')
      if not page_token:
        break

  def yield_instances_in_zones(self, zones, instance_filter=None):
    """Yields dicts with all project instances in the specific zones.

    Fetches listing sequentially zone-by-zone. We assume non-existing zones are
    empty.

    The format of the instance dict is defined here:
      https://cloud.google.com/compute/docs/reference/v1/instances#resource

    Returns instances in all possible states (instance['status'] attribute):
      PROVISIONING
      STAGING
      RUNNING
      STOPPING
      STOPPED
      TERMINATED

    Very slow call (can run for minutes). Should be used only from task queues.

    Args:
      zones: a list of zones to list instances in, e.g ["us-central1-a"].
      instance_filter: optional filter to apply to instance names when scanning.
    """
    for zone in zones:
      for instance in self.yield_instances_in_zone(zone, instance_filter):
        yield instance

  def set_metadata(self, zone, instance, fingerprint, items):
    """Initiates metadata update operation.

    Args:
      zone: name of a zone, e.g. 'us-central1-a'.
      instance: name of an instance, e.g. 'slave123-c4'.
      fingerprint: fingerprint of existing metadata.
      items: list of {'key': ..., 'value': ...} dicts with new metadata.

    Returns:
      ZoneOperation object that can be polled to wait for result.
    """
    assert is_valid_zone(zone), zone
    assert is_valid_instance(instance), instance
    op_info = self.call_api(
        endpoint='/zones/%s/instances/%s/setMetadata' % (zone, instance),
        method='POST',
        payload={
          'kind': 'compute#metadata',
          'fingerprint': fingerprint,
          'items': items,
        })
    return ZoneOperation(self, zone, op_info)

  def check_zone_operation(self, zone, operation):
    """Returns the result of a zone operation.

    Args:
      zone: name of a zone, e.g. 'us-central1-a'.
      operation: name of the operation.

    Returns:
      A compute#operation dict.
    """
    assert is_valid_zone(zone), zone
    return self.call_api('/zones/%s/operations/%s' % (zone, operation))

  def add_access_config(self, zone, instance, network_interface, external_ip):
    """Attaches external IP (given as IPv4 string) to an instance's NIC."""
    assert is_valid_zone(zone), zone
    assert is_valid_instance(instance), instance
    op_info = self.call_api(
        endpoint='/zones/%s/instances/%s/addAccessConfig' % (zone, instance),
        params={'networkInterface': network_interface},
        method='POST',
        payload={
          'kind': 'compute#accessConfig',
          'type': 'ONE_TO_ONE_NAT',
          'name': 'External NAT',
          'natIP': external_ip,
        })
    return ZoneOperation(self, zone, op_info)

  def list_addresses(self, region):
    """Yields dicts with reserved IPs in a region.

    Very slow call (can run for minutes). Should be used only from task queues.
    """
    assert is_valid_region(region), region
    page_token = None
    while True:
      params = {'maxResults': 250}
      if page_token:
        params['pageToken'] = page_token
      resp = self.call_api(
          '/regions/%s/addresses' % region, params=params, deadline=120)
      for addr in resp.get('items', []):
        yield addr
      page_token = resp.get('nextPageToken')
      if not page_token:
        break

  def create_instance_template(
      self, name, disk_size_gb, image, machine_type,
      auto_assign_external_ip=False, disk_type=None, metadata=None,
      network_url='', min_cpu_platform=None, service_accounts=None, tags=None):
    """
    Args:
      name: Name of the instance template.
      disk_size_gb: Disk size in GiB for instances created from this template.
      image: Image to use for instances created from this template.
      machine_type: GCE machine type for instances created from this template.
        e.g. n1-standard-8.
      auto_assign_external_ip: flag to enable external network with
        auto-assigned IP address.
      disk_type: Disk type for instances created from this template.
      metadata: List of {'key': ..., 'value': ...} dicts to attach as metadata
        to instances created from this template.
      network_url: name or URL of the network resource for this template.
      min_cpu_platfom: Minimum CPU platform for instances (e.g. Intel Skylake).
      service_accounts: List of {'email': ..., 'scopes': [...]} dicts to make
        available to instances created from this template.
      tags: List of strings to attach as tags to instances created from this
        template.

    Returns:
      A compute#operation dict.
    """
    tags = tags or []
    metadata = metadata or []
    network_interfaces = get_network_interfaces(self.project_id, network_url,
                                                auto_assign_external_ip)
    service_accounts = service_accounts or []

    payload = {
        'name': name,
        'properties': {
            'disks': [
              {
                  'autoDelete': True,
                  'boot': True,
                  'initializeParams': {
                      'diskSizeGb': disk_size_gb,
                      'sourceImage': image,
                  },
              },
            ],
            'machineType': machine_type,
            'metadata': {
                'items': metadata,
            },
            'networkInterfaces': network_interfaces,
            'serviceAccounts': service_accounts,
            'tags': {
                'items': tags,
            },
        },
    }

    # Empty 'diskType' field is rejected, need to omit it entirely.
    if disk_type:
      payload['properties']['disks'][0]['initializeParams']['diskType'] = (
          disk_type
      )
    # Empty 'minCpuPlatform' field is rejected, need to omit it entirely.
    if min_cpu_platform:
      payload['properties']['minCpuPlatform'] = min_cpu_platform

    return self.call_api(
        '/global/instanceTemplates',
        method='POST',
        payload=payload,
    )

  def create_instance_group_manager(
      self, name, instance_template, size, zone, base_name=None):
    """Creates an instance group manager from the given template.

    Args:
     name: Name of the instance group.
     instance_template: URL of the instance template to create instances from.
     size: Number of instances the group manager should maintain.
     zone: Zone to create the instance group in. e.g. us-central1-b.
     base_name: Base name for instances created by tihs instance group manager.
       Defaults to name.

    Returns:
      A compute#operation dict.
    """
    return self.call_api(
        '/zones/%s/instanceGroupManagers' % zone,
        method='POST',
        payload={
            'baseInstanceName': base_name or name,
            'instanceTemplate': instance_template,
            'name': name,
            'targetSize': size,
        },
    )

  def delete_instances(self, manager, zone, instance_urls):
    """Deletes the given GCE instances from the given instance group manager.

    Args:
      manager: Name of the instance group manager.
      zone: Zone to delete the instances in.
      instance_urls: List of URLs of instances to delete.

    Returns:
      JSON describing the result of the operation.
    """
    return self.call_api(
        '/zones/%s/instanceGroupManagers/%s/deleteInstances' % (
            zone,
            manager,
        ),
        method='POST',
        payload={'instances': instance_urls},
    )

  def get_instances_in_instance_group(
      self, name, zone, max_results=None, page_token=None):
    """Returns the instances in the specified GCE instance group.

    Args:
      name: Name of the instance group manager.
      zone: Zone the instance group manager exists in.
      max_results: If specified, maximum number of instances to return.
      page_token: If specified, token to use to return a specific page of
        instances.

    Returns:
      A compute#instanceGroupsListInstances dict.
    """
    params = {}
    if max_results:
      params['maxResults'] = max_results
    if page_token:
      params['pageToken'] = page_token
    return self.call_api(
        '/zones/%s/instanceGroups/%s/listInstances' % (zone, name),
        method='POST',
        params=params,
    )

  def get_instance_group_manager(self, name, zone):
    """Returns the specified GCE instance group manager.

    Returns:
      A compute#instanceGroupManager dict.

    Raises:
      net.NotFoundError: If the instance group manager does not exist.
    """
    return self.call_api('/zones/%s/instanceGroupManagers/%s' % (zone, name))

  def get_instance_group_managers(self, zone):
    """Returns the GCE instance group managers associated with this project.

    Args:
      zone: Zone to list the instance group managers in.

    Returns:
      A dict mapping instance group manager names to
      compute#instanceGroupManager dicts.
    """
    response = self.call_api('/zones/%s/instanceGroupManagers' % zone)
    return {manager['name']: manager for manager in response.get('items', [])}

  def get_instance_template(self, name):
    """Returns the specified GCE instance template.

    Returns:
      A compute#instanceTemplate dict.

    Raises:
      net.NotFoundError: If the instance template does not exist.
    """
    return self.call_api('/global/instanceTemplates/%s' % name)

  def get_instance_templates(self):
    """Returns the GCE instance templates associated with this project.

    Returns:
      A dict mapping instance template names to compute#instanceTemplate dicts.
    """
    response = self.call_api('/global/instanceTemplates')
    return {
        template['name']: template for template in response.get('items', [])
    }

  def get_managed_instances(self, manager, zone):
    """Returns the GCE instances managed by the given instance group manager.

    Args:
      manager: Name of the instance group manager.
      zone: Zone to list the managed instances in.

    Returns:
      A dict mapping instance names to dicts describing those managed instances.
    """
    response = self.call_api(
        '/zones/%s/instanceGroupManagers/%s/listManagedInstances' % (
            zone,
            manager,
        ),
        method='POST',
    )
    return {
        # Extract instance name from a link to the instance.
        instance['instance'].split('/')[-1]: instance
        for instance in response.get('managedInstances', [])
    }

  def resize_managed_instance_group(self, manager, zone, size):
    """Resizes the instance group managed by the given instance group manager.

    Args:
      manager: Name of the instance group manager.
      zone: Zone to resize the instance group in. e.g. us-central1-f
      size: Desired number of instances in the instance group.

    Returns:
      A list of error dicts containing "code", "location", and "message" keys.
    """
    response = self.call_api(
        '/zones/%s/instanceGroupManagers/%s/resize' % (zone, manager),
        method='POST',
        params={
            'size': size,
        },
    )
    return response.get('error', {}).get('errors', [])

  def get_snapshots(
      self, name=None, labels=None, max_results=None, page_token=None):
    """Returns the snapshots matching the specified name and labels.

    Args:
      name: Name of a snapshot. If unspecified, matches all names.
      labels: Dict of labels. If unspecified, matches all labels.
      max_results: If specified, maximum number of snapshots to return.
      page_token: If specified, token to use to return a specific page of
        snapshots.

    Returns:
      A compute#snapshotList dict.
    """
    labels = labels or {}
    params = {}
    filters = []
    if name:
      filters.append('(name = %s)' % name)
    for key, value in sorted(labels.iteritems()):
      filters.append('(labels.%s = %s)' % (key, value))
    if filters:
      # e.g. (name = snapshot-name) AND (label.version = latest)
      params['filter'] = ' AND '.join(filters)
    if max_results:
      params['maxResults'] = max_results
    if page_token:
      params['pageToken'] = page_token
    return self.call_api('/global/snapshots', params=params)

  def create_disk(self, name, snapshot, zone):
    """Creates a disk from the given snapshot.

    Args:
      name: Name of the disk to create.
      snapshot: Snapshot to use when creating the disk.
      zone: Zone to create the disk in. e.g. us-central1-f
    """
    return self.call_api(
        '/zones/%s/disks' % zone,
        method='POST',
        payload={
            'name': name,
            'sourceSnapshot': snapshot,
        },
    )

  def attach_disk(self, instance, disk, zone):
    """Attaches a disk to an instance.

    Args:
      instance: Name of the instance to attach the disk to.
      disk: Name of the disk to attach.
      zone: Zone the disk and instance already exist in. e.g. us-central1-f
    """
    return self.call_api(
        '/zones/%s/instances/%s/attachDisk' % (zone, instance),
        method='POST',
        payload={
            'autoDelete': True,
            'deviceName': disk,
            'source': 'projects/%s/zones/%s/disks/%s' % (
                self.project_id, zone, disk),
        },
    )


class ZoneOperation(object):
  """Asynchronous GCE operation returned by some Project methods.

  Usage:
    op = project.set_metadata(...)
    while not op.poll():
      <operation is not finished yet>
    if op.error:
      <operation failed>
  """

  def __init__(self, project, zone, info):
    self._project = project
    self._zone = zone
    self._info = info

  def poll(self):
    """Refetches operation status, returns True if operation is done."""
    if not self.done:
      self._info = self._project.call_api(
          '/zones/%s/operations/%s' % (self._zone, self._info['name']))
    return self.done

  @property
  def name(self):
    return self._info['name']

  @property
  def done(self):
    """True when operation completes (successfully or not)."""
    return self._info['status'] == 'DONE'

  @property
  def url(self):
    return self._info['selfLink']

  @property
  def error(self):
    """Error message on error or None on success or if not yet done."""
    errors = self._info.get('error', {}).get('errors')
    if not errors:
      return None
    return ' '.join(err.get('message', 'unknown') for err in errors)

  def has_error_code(self, code):
    """True if this operation has a suberror with given code."""
    errors = self._info.get('error', {}).get('errors')
    return any(err.get('code') == code for err in errors)


def is_valid_project_id(project_id):
  """True if string looks like a valid Cloud Project id."""
  return re.match(r'^(google.com:)?[a-z0-9\-]+$', project_id)


def is_valid_region(region):
  """True if string looks like a GCE region name."""
  return re.match(r'^[a-z0-9\-]+$', region)


def is_valid_zone(zone):
  """True if string looks like a GCE zone name."""
  return re.match(r'^[a-z0-9\-]+$', zone)


def is_valid_instance(instance):
  """True if string looks like a valid GCE instance name."""
  return re.match(r'^[a-z0-9\-_]+$', instance)


def is_valid_image(image):
  """True if string looks like a valid GCE image name."""
  return re.match(r'^[a-z0-9\-_]+$', image)


def is_valid_network(network):
  """True if string looks like a valid GCE network name."""
  return re.match(r'^[a-z0-9\-_]+$', network)


def get_image_url(project_id, image):
  """Returns full image URL given image name."""
  assert is_valid_project_id(project_id), project_id
  assert is_valid_image(image), image
  return (
      'https://www.googleapis.com/compute/v1/projects/%s/global/images/%s' % (
          project_id, image))


def get_network_interfaces(project_id, network_url, auto_assign_external_ip):
  """Returns list of network interfaces for configuring a network.

  Args:
    project_id: project ID
    network_url: name or URL of the network resource property.
    auto_assign_external_ip: flag to enable external network with auto-assigned
      IP address.

  Returns:
    network_interfaces: List of network_interface dicts to configure instance
      networks. For more details, see:
      https://cloud.google.com/compute/docs/reference/latest/instanceTemplates
  """
  network = network_url or get_network_url(project_id, 'default')
  network_interfaces = [{'network': network}]
  if auto_assign_external_ip:
    # This creates a single accessConfig instance and uses default values for
    # all fields to enable external network with auto-assigned IP.
    network_interfaces[0]['accessConfigs'] = [{'type': 'ONE_TO_ONE_NAT'}]
  return network_interfaces


def get_network_url(project_id, network):
  """Returns full network URL given network name."""
  assert is_valid_project_id(project_id), project_id
  assert is_valid_network(network), network
  return (
      'https://www.googleapis.com/compute/v1/projects/%s/global/networks/%s' % (
          project_id, network))


def get_zone_url(project_id, zone):
  """Returns full zone URL given zone name."""
  assert is_valid_project_id(project_id), project_id
  assert is_valid_zone(zone), zone
  return 'https://www.googleapis.com/compute/v1/projects/%s/zones/%s' % (
      project_id, zone)


def extract_zone(zone_url):
  """Given zone URL (as in instance['zone']) returns zone name."""
  zone = zone_url[zone_url.rfind('/')+1:]
  assert is_valid_zone(zone), zone
  return zone


def get_region_url(project_id, region):
  """Returns full region URL given region name."""
  assert is_valid_project_id(project_id), project_id
  assert is_valid_region(region), region
  return 'https://www.googleapis.com/compute/v1/projects/%s/regions/%s' % (
      project_id, region)


def extract_instance_name(url):
  """Given instance URL returns instance name."""
  return url.rsplit('/', 1)[-1]


def extract_region(region_url):
  """Given region URL (as in address['region']) returns region name."""
  region = region_url[region_url.rfind('/')+1:]
  assert is_valid_region(region), region
  return region


def region_from_zone(zone):
  """Given a zone name returns a region: us-central1-a -> us-central1."""
  assert is_valid_zone(zone), zone
  return zone[:zone.rfind('-')]


def machine_type_to_num_cpus(machine_type):
  """Given a machine type returns its number of CPUs."""
  if machine_type in MACHINE_TYPES:
    return MACHINE_TYPES[machine_type]['cpus']
  m = re.match(CUSTOM_MACHINE_TYPE_RE, machine_type)
  assert m, machine_type
  return int(m.group(1))


def machine_type_to_memory(machine_type):
  """Given a machine type returns its memory in GB."""
  if machine_type in MACHINE_TYPES:
    return MACHINE_TYPES[machine_type]['memory']
  m = re.match(CUSTOM_MACHINE_TYPE_RE, machine_type)
  assert m, machine_type
  return int(m.group(2)) / 1024
