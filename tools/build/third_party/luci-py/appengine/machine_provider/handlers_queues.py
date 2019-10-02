# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Task queues for fulfilling lease requests."""

import json
import logging

from google.appengine.ext import ndb
import webapp2

from components import decorators
from components import net
from components import pubsub
from components.machine_provider import rpc_messages

import metrics
import models


def maybe_notify_backend(message, hostname, policies):
  """Informs the backend of the status of a request if there's a Pub/Sub topic.

  Args:
    message: The message string to send.
    hostname: The hostname of the machine this message concerns.
    policies: A dict representation of an rpc_messages.Policies instance.
  """
  if policies.get('backend_topic'):
    topic = pubsub.full_topic_name(
        policies['backend_project'], policies['backend_topic'])
    attributes = {
        attribute['key']: attribute['value']
        for attribute in policies['backend_attributes']
    }
    attributes['hostname'] = hostname
    pubsub.publish(topic, message, attributes)
    # There are relatively few backends, so it's safe to include the
    # backend topic/project as the value for the target field.
    metrics.pubsub_messages_sent.increment(fields={'target': topic})


def maybe_notify_lessee(request, response):
  """Informs the lessee of the status of a request if there's a Pub/Sub topic.

  Args:
    request: A dict representation of an rpc_messages.LeaseRequest instance.
    response: A dict representation of an rpc_messages.LeaseResponse instance.
  """
  if request.get('pubsub_topic'):
    pubsub.publish(
        pubsub.full_topic_name(
            request['pubsub_project'], request['pubsub_topic']),
        json.dumps(response),
        {},
    )
    metrics.pubsub_messages_sent.increment(fields={'target': 'lessee'})


class LeaseRequestFulfiller(webapp2.RequestHandler):
  """Worker for fulfilling lease requests."""

  @decorators.require_taskqueue('fulfill-lease-request')
  def post(self):
    """Fulfill a lease request.

    Params:
      policies: JSON-encoded string representation of the
        rpc_messages.Policies governing this machine.
      request_json: JSON-encoded string representation of the
        rpc_messages.LeaseRequest being fulfilled.
      response_json: JSON-encoded string representation of the
        rpc_messages.LeaseResponse being delivered.
    """
    policies = json.loads(self.request.get('policies'))
    request = json.loads(self.request.get('request_json'))
    response = json.loads(self.request.get('response_json'))

    maybe_notify_backend('LEASED', response['hostname'], policies)
    maybe_notify_lessee(request, response)


@ndb.transactional(xg=True)
def reclaim(machine_key):
  """Reclaims a machine.

  Args:
    machine_key: ndb.Key for a models.CatalogMachineEntry.
  """
  machine = machine_key.get()
  if not machine:
    return

  lease = models.LeaseRequest.get_by_id(machine.lease_id)
  lease.machine_id = None
  lease.response.hostname = None
  machine.key.delete()
  lease.put()


class MachineReclaimer(webapp2.RequestHandler):
  """Worker for reclaiming machines."""

  @decorators.require_taskqueue('reclaim-machine')
  def post(self):
    """Reclaim a machine.

    Params:
      hostname: Hostname of the machine being reclaimed.
      machine_key: URL-safe ndb.Key for a models.CatalogMachineEntry.
      policies: JSON-encoded string representation of the
        rpc_messages.Policies governing this machine.
      request_json: JSON-encoded string representation of the
        rpc_messages.LeaseRequest being fulfilled.
      response_json: JSON-encoded string representation of the
        rpc_messages.LeaseResponse being delivered.
    """
    hostname = self.request.get('hostname')
    machine_key = ndb.Key(urlsafe=self.request.get('machine_key'))
    policies = json.loads(self.request.get('policies'))
    request = json.loads(self.request.get('request_json'))
    response = json.loads(self.request.get('response_json'))

    assert machine_key.kind() == 'CatalogMachineEntry', machine_key

    maybe_notify_backend('RECLAIMED', hostname, policies)
    maybe_notify_lessee(request, response)

    reclaim(machine_key)
    metrics.lease_requests_expired.increment()


def create_queues_app():
  return webapp2.WSGIApplication([
      ('/internal/queues/fulfill-lease-request', LeaseRequestFulfiller),
      ('/internal/queues/reclaim-machine', MachineReclaimer),
  ])
