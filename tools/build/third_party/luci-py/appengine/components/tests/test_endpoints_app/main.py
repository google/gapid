# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# Pylint doesn't like endpoints.
# pylint: disable=C0322,R0201

import endpoints
from protorpc import message_types
from protorpc import messages
from protorpc import remote

from components import auth
from components import endpoints_webapp2


package = 'testing_api'


class WhoResponse(messages.Message):
  identity = messages.StringField(1)
  ip = messages.StringField(2)


@auth.endpoints_api(name='testing_service', version='v1')
class TestingServiceApi(remote.Service):
  @auth.endpoints_method(
      message_types.VoidMessage,
      WhoResponse,
      name='who',
      http_method='GET')
  @auth.public
  def who(self, _request):
    return WhoResponse(
        identity=auth.get_current_identity().to_bytes(),
        ip=auth.ip_to_string(auth.get_peer_ip()))

  @auth.endpoints_method(
      message_types.VoidMessage,
      message_types.VoidMessage,
      name='forbidden',
      http_method='GET')
  @auth.require(lambda: False)
  def forbidden(self, _request):
    pass


app = endpoints_webapp2.api_server([TestingServiceApi])
