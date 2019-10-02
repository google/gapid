# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.


"""OAuth2 utilities for Swarming bots."""


import json
import logging
import time
import urllib2


def oauth2_access_token_from_url(url, headers):
  """Obtain an OAuth2 access token from the given URL.

  Returns tuple (oauth2 access token, expiration timestamp).

  Args:
    url: HTTP URL from which to retrieve the token. HTTPS support would require
        a bit more plumbing.
    headers: dict of HTTP headers to use for the request.
  """
  try:
    resp = json.load(
        urllib2.urlopen(urllib2.Request(url, headers=headers), timeout=20))
  except IOError as e:
    logging.error('Failed to grab OAuth2 access token: %s', e)
    raise
  return resp['access_token'], time.time() + resp['expires_in']
