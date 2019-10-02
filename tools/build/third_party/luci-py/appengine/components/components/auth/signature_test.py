#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import collections
import datetime
import random
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from components.auth import signature
from test_support import test_case


# Skip check_signature tests if PyCrypto is not available.
try:
  import Crypto
  has_pycrypto = True
except ImportError:
  has_pycrypto = False


FetchResult = collections.namedtuple('FetchResult', ['status_code', 'content'])


FAKE_SERVICE_ACCOUNT_CERTS = """
{
 "faffca3b64d5bb61da829ace9aed119dceb2f63c": "abc",
 "cba74246d54580c5ee7a6a778c997a7cb1abc918": "def"
}
"""

class SignatureTest(test_case.TestCase):
  def test_get_service_account_certificates(self):
    def do_fetch(url, **_kwargs):
      self.assertEqual(
        url,
        'https://www.googleapis.com/robot/v1/metadata/x509/'
        '123%40appspot.gserviceaccount.com')
      return FetchResult(200, FAKE_SERVICE_ACCOUNT_CERTS)
    self.mock(signature.urlfetch, 'fetch', do_fetch)

    certs = signature.get_service_account_certificates(
        '123@appspot.gserviceaccount.com')
    self.assertEqual(
        '123@appspot.gserviceaccount.com', certs.service_account_name)
    self.assertEqual(certs.to_jsonish()['certificates'], [
      {
        'key_name': u'cba74246d54580c5ee7a6a778c997a7cb1abc918',
        'x509_certificate_pem': u'def',
      },
      {
        'key_name': u'faffca3b64d5bb61da829ace9aed119dceb2f63c',
        'x509_certificate_pem': u'abc',
      },
    ])

  def test_caching(self):
    now = datetime.datetime(2014, 2, 2, 3, 4, 5)

    self.mock(random, 'random', lambda: 1.0)
    self.mock_now(now)

    def fetch():
      return signature._use_cached_or_fetch('cache_key', lambda: {})

    # Fetch one. It gets put into cache. On the second fetch get exact same one.
    certs = fetch()
    self.assertTrue(fetch() is certs)

    # Some time later cache expires. This happens slightly earlier than
    # _CERTS_CACHE_EXP_SEC due to randomized early expiration.
    self.mock_now(now, 3600 - 360 + 1)
    self.assertFalse(fetch() is certs)

  if has_pycrypto:
    def test_check_signature_correct(self):
      blob = '123456789'
      key_name, sig = signature.sign_blob(blob)
      certs = signature.get_own_public_certificates()
      self.assertTrue(certs.check_signature(blob, key_name, sig))
      # Again, to hit a code path that uses cached verifier.
      self.assertTrue(certs.check_signature(blob, key_name, sig))

    def test_check_signature_wrong(self):
      blob = '123456789'
      key_name, sig = signature.sign_blob(blob)
      sig = chr(ord(sig[0]) + 1) + sig[1:]
      certs = signature.get_own_public_certificates()
      self.assertFalse(certs.check_signature(blob, key_name, sig))

    def test_check_signature_missing_key(self):
      certs = signature.get_own_public_certificates()
      with self.assertRaises(signature.CertificateError):
        certs.check_signature('blob', 'wrong-key', 'sig')


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
