# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import rsa

from pyasn1.codec.der import decoder
from pyasn1.type import univ
from rsa import pkcs1


# python-rsa lib hashes all messages it signs. ADB does it already, we just
# need to slap a signature on top of already hashed message. Introduce "fake"
# hashing algo for this.
class _Accum(object):
  def __init__(self):
    self._buf = ''
  def update(self, msg):
    self._buf += msg
  def digest(self):
    return self._buf
pkcs1.HASH_METHODS['SHA-1-PREHASHED'] = _Accum
pkcs1.HASH_ASN1['SHA-1-PREHASHED'] = pkcs1.HASH_ASN1['SHA-1']


def _load_rsa_private_key(pem):
  """PEM encoded PKCS#8 private key -> rsa.PrivateKey."""
  # ADB uses private RSA keys in pkcs#8 format. 'rsa' library doesn't support
  # them natively. Do some ASN unwrapping to extract naked RSA key
  # (in der-encoded form). See https://www.ietf.org/rfc/rfc2313.txt.
  # Also http://superuser.com/a/606266.
  try:
    der = rsa.pem.load_pem(pem, 'PRIVATE KEY')
    keyinfo, _ = decoder.decode(der)
    if keyinfo[1][0] != univ.ObjectIdentifier(
        '1.2.840.113549.1.1.1'):  # pragma: no cover
      raise ValueError('Not a DER-encoded OpenSSL private RSA key')
    private_key_der = keyinfo[2].asOctets()
  except IndexError:  # pragma: no cover
    raise ValueError('Not a DER-encoded OpenSSL private RSA key')
  return rsa.PrivateKey.load_pkcs1(private_key_der, format='DER')


class PythonRSASigner(object):
  """Implements adb_protocol.AuthSigner using http://stuvel.eu/rsa."""
  @classmethod
  def FromRSAKeyPath(cls, rsa_key_path):
    with open(rsa_key_path + '.pub') as f:
      pub = f.read()
    with open(rsa_key_path) as f:
      priv = f.read()
    return cls(pub, priv)

  def __init__(self, pub=None, priv=None):
    self.priv_key = _load_rsa_private_key(priv)
    self.pub_key = pub

  def Sign(self, data):
    return rsa.sign(data, self.priv_key, 'SHA-1-PREHASHED')

  def GetPublicKey(self):
    return self.pub_key
