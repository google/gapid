# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Discovery service implementation."""

import service_pb2
import service_prpc_pb2


class Discovery(object):
  DESCRIPTION = service_prpc_pb2.DiscoveryServiceDescription

  def __init__(self):
    self._response = service_pb2.DescribeResponse()
    self._files = set()

  def Describe(self, _request, _ctx):
    return self._response

  def add_service(self, service_description):
    file_desc = service_description['file_descriptor']
    full_name = service_description['service_descriptor'].name
    if file_desc.package:
      full_name = '%s.%s' % (file_desc.package, full_name)
    self._response.services.append(full_name)

    for f in service_description['file_descriptor_set'].file:
      if f.name not in self._files:
        self._response.description.file.add().CopyFrom(f)
        self._files.add(f.name)
