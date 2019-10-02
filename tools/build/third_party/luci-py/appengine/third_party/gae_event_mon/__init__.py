# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# pylint: disable=line-too-long

import logging
import os
import sys

# Since event_mon depends on ts_mon, we reuse module setup (__init__.py) from
# the gae_ts_mon, which configures infra_libs as if it was imported directly.
# We expect the users of gae_event_mon to symlink both this and gae_ts_mon
# modules into their apps.
import gae_ts_mon

# Additional infra_libs configuration for event_mon.
import infra_libs.ts_mon.httplib2_utils
sys.modules['infra_libs'].InstrumentedHttp = (
    infra_libs.ts_mon.httplib2_utils.InstrumentedHttp)
sys.modules['infra_libs'].event_mon = sys.modules[__package__]
sys.modules['infra_libs.event_mon'] = sys.modules[__package__]

from google.appengine.api import modules
from google.appengine.api.app_identity import app_identity

def initialize(service_name):
  is_local_unittest = ('expect_tests' in sys.argv[0])
  if is_local_unittest:
    appengine_name = 'unittest'
    service_name = 'unittest'
    hostname = 'unittest'
  else:  # pragma: no cover
    appengine_name = app_identity.get_application_id()
    hostname = '%s, %s' % (modules.get_current_module_name(),
                           modules.get_current_version_name())

  # Only send events if we are running on the actual AppEngine.
  if os.environ.get('SERVER_SOFTWARE', '').startswith('Google App Engine'):
    run_type = 'prod'  # pragma: no cover
  else:
    run_type = 'dry'

  config.setup_monitoring(run_type, hostname, service_name, appengine_name)
  logging.info(
      'Initialized event_mon with run_type=%s, hostname=%s, service_name=%s, '
      'appengine_name=%s', run_type, hostname, service_name, appengine_name)

# The remaining lines are copied from infra_libs/event_mon/__init__.py.
from infra_libs.event_mon.config import add_argparse_options
from infra_libs.event_mon.config import close
from infra_libs.event_mon.config import set_default_event, get_default_event
from infra_libs.event_mon.config import process_argparse_options
from infra_libs.event_mon.config import setup_monitoring

from infra_libs.event_mon.monitoring import BUILD_EVENT_TYPES, BUILD_RESULTS
from infra_libs.event_mon.monitoring import EVENT_TYPES, TIMESTAMP_KINDS
from infra_libs.event_mon.monitoring import Event
from infra_libs.event_mon.monitoring import get_build_event
from infra_libs.event_mon.monitoring import send_build_event
from infra_libs.event_mon.monitoring import send_events
from infra_libs.event_mon.monitoring import send_service_event

from infra_libs.event_mon.protos.chrome_infra_log_pb2 import ChromeInfraEvent
from infra_libs.event_mon.protos.chrome_infra_log_pb2 import BuildEvent
from infra_libs.event_mon.protos.chrome_infra_log_pb2 import ServiceEvent
from infra_libs.event_mon.protos.chrome_infra_log_pb2 import InfraEventSource
from infra_libs.event_mon.protos.chrome_infra_log_pb2 import CodeVersion
from infra_libs.event_mon.protos.chrome_infra_log_pb2 import CQEvent
from infra_libs.event_mon.protos.chrome_infra_log_pb2 import MachineProviderEvent

from infra_libs.event_mon.protos.log_request_lite_pb2 import LogRequestLite
