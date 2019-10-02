# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Metrics to track with ts_mon and event_mon."""

import logging

from components.machine_provider import rpc_messages

import gae_ts_mon
import models


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
    'lease_requests_untriaged': gae_ts_mon.GaugeMetric(
        'machine_provider/lease_requests/untriaged',
        'Number of untriaged lease requests.',
        None,
    ),
}


lease_requests_deduped = gae_ts_mon.CounterMetric(
    'machine_provider/lease_requests/deduped',
    'Number of lease requests deduplicated.',
    None,
)


lease_requests_expired = gae_ts_mon.CounterMetric(
    'machine_provider/lease_requests/expired',
    'Number of lease requests expired.',
    None,
)


lease_requests_fulfilled = gae_ts_mon.CounterMetric(
    'machine_provider/lease_requests/fulfilled',
    'Number of lease requests fulfilled.',
    None,
)


lease_requests_fulfilled_time = gae_ts_mon.CumulativeDistributionMetric(
    'machine_provider/lease_requests/fulfilled/time',
    'Time taken to fulfill a lease request.',
    None,
    bucketer=gae_ts_mon.GeometricBucketer(growth_factor=10**0.04),
)


lease_requests_received = gae_ts_mon.CounterMetric(
    'machine_provider/lease_requests/received',
    'Number of lease requests received.',
    None,
)


pubsub_messages_sent = gae_ts_mon.CounterMetric(
    'machine_provider/pubsub_messages/sent',
    'Number of Pub/Sub messages sent.',
    [gae_ts_mon.StringField('target')],
)


def compute_global_metrics(): # pragma: no cover
  count = models.LeaseRequest.query(
      models.LeaseRequest.response.state==
          rpc_messages.LeaseRequestState.UNTRIAGED).count()
  logging.info('Untriaged lease requests: %d', count)
  GLOBAL_METRICS['lease_requests_untriaged'].set(
      count, target_fields=GLOBAL_TARGET_FIELDS)


def initialize(): # pragma: no cover
  gae_ts_mon.register_global_metrics(GLOBAL_METRICS.values())
  gae_ts_mon.register_global_metrics_callback(
      'callback', compute_global_metrics)
