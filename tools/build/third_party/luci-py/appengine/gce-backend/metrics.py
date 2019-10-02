# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Metrics to track with ts_mon and event_mon."""

import logging

import gae_event_mon
import gae_ts_mon

import config
import instance_group_managers


# Overrides to create app-global metrics.
GLOBAL_TARGET_FIELDS = {
  # Name of the module reporting the metric.
  'job_name': '',
  # Version of the app reporting the metric.
  'hostname': '',
  # ID of the instance reporting the metric.
  'task_num': 0,
}


GLOBAL_METRICS = {
    'config_max_instances': gae_ts_mon.GaugeMetric(
        'machine_provider/gce_backend/config/instances/max',
        'Maximum number of instances currently configured.',
        [gae_ts_mon.StringField('instance_template')]
    ),
    'config_min_instances': gae_ts_mon.GaugeMetric(
        'machine_provider/gce_backend/config/instances/min',
        'Minimum number of instances currently configured.',
        [gae_ts_mon.StringField('instance_template')]
    ),
    'instances': gae_ts_mon.GaugeMetric(
        'machine_provider/gce_backend/instances',
        'Current count of the number of instances.',
        [gae_ts_mon.StringField('instance_template'),
         gae_ts_mon.BooleanField('drained')]
    ),
}


config_valid = gae_ts_mon.BooleanMetric(
    'machine_provider/gce_backend/config/valid',
    'Whether or not the current config is valid.',
    [gae_ts_mon.StringField('config')],
)


instance_deletion_time = gae_ts_mon.CumulativeDistributionMetric(
    'machine_provider/gce_backend/instances/deletions/time',
    'Seconds between initiating deletion RPC and learning its result.',
    [gae_ts_mon.StringField('zone')],
#    units=ts_mon.MetricsDataUnits.SECONDS,
)


def compute_global_metrics(): # pragma: no cover
  for name, (minimum, maximum) in config.count_instances().iteritems():
    logging.info('%s min: %s', name, minimum)
    GLOBAL_METRICS['config_min_instances'].set(
        minimum,
        fields={
            'instance_template': name,
        },
        target_fields=GLOBAL_TARGET_FIELDS,
    )
    logging.info('%s max: %s', name, maximum)
    GLOBAL_METRICS['config_max_instances'].set(
        maximum,
        fields={
            'instance_template': name,
        },
        target_fields=GLOBAL_TARGET_FIELDS,
    )

  counts = instance_group_managers.count_instances()
  for name, (active, drained) in counts.iteritems():
    logging.info('%s active: %s', name, active)
    GLOBAL_METRICS['instances'].set(
        active,
        fields={
            'drained': False,
            'instance_template': name,
        },
        target_fields=GLOBAL_TARGET_FIELDS,
    )
    logging.info('%s drained: %s', name, drained)
    GLOBAL_METRICS['instances'].set(
        drained,
        fields={
            'drained': True,
            'instance_template': name,
        },
        target_fields=GLOBAL_TARGET_FIELDS,
    )


def initialize(): # pragma: no cover
  gae_ts_mon.register_global_metrics(GLOBAL_METRICS.values())
  gae_ts_mon.register_global_metrics_callback(
      'callback', compute_global_metrics)


def send_machine_event(state, hostname): # pragma: no cover
  """Sends an event_mon event about a GCE instance.

  Args:
    state: gae_event_mon.ChromeInfraEvent.GCEBackendMachineState.
    hostname: Name of the GCE instance this event is for.
  """
  state = gae_event_mon.MachineProviderEvent.GCEBackendMachineState.Value(state)
  event = gae_event_mon.Event('POINT')
  event.proto.event_source.host_name = hostname
  event.proto.machine_provider_event.gce_backend_state = state
  logging.info('Sending event: %s', event.proto)
  event.send()
