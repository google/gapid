# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for working with IPv4 and IPv6 addresses."""

import collections


# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'IP',
  'ip_from_string',
  'ip_to_string',
  'is_in_subnet',
  'normalize_ip',
  'normalize_subnet',
  'Subnet',
  'subnet_from_string',
  'subnet_to_string',
]


# Parsed IPv4 or IPv6 address. 'bits' is 32 for IPv4 and 128 for IPv6,
# value is long int in native endianness.
IP = collections.namedtuple('IP', ['bits', 'value'])

# Parsed IPv4 or IPv6 subnet. 'bits' is 32 for IPv4 and 128 for IPv6,
Subnet = collections.namedtuple('Subnet', ['bits', 'base', 'mask'])


def ip_from_string(ipstr):
  """Parses IPv4 or IPv6 string and returns an instance of IP.

  This works around potentially different representations of the same value,
  like 1.1.1.1 vs 1.01.1.1 or hex case difference in IPv6.

  Raises ValueError if |ipstr| is not recognized as IPv4 or IPv6 address.
  """
  assert isinstance(ipstr, basestring), '%r is not a string' % (ipstr,)
  values = None
  try:
    if '.' in ipstr:
      # IPv4. Mixed v4-v6 addresses (e.g ::ffff:192.0.2.128) are not supported.
      values = [int(i) for i in ipstr.split('.')]
      if len(values) != 4:
        raise ValueError('expecting 4 bytes, got %d' % len(values))
      if not all(0 <= i <= 255 for i in values):
        raise ValueError('byte value is out of range')
      factor = 256
      bits = 32
    elif ':' in ipstr:
      # IPv6. '::' will be replaced with appropriate number of zeros.
      gaps = ipstr.count('::')
      if gaps > 1:
        raise ValueError('\'::\' can be used at most once')
      if not gaps:
        blocks = ipstr.split(':')
      else:
        idx = ipstr.index('::')
        before = ipstr[:idx].split(':') if ipstr[:idx] else []
        after = ipstr[idx+2:].split(':') if ipstr[idx+2:] else []
        if len(before) + len(after) >= 8:
          raise ValueError('too many sections')
        blocks = before + ['0'] * (8 - len(before) - len(after)) + after
      if len(blocks) != 8:
        raise ValueError('expecting 8 sections, got %d' % len(blocks))
      values = [int(i, 16) for i in blocks]
      if not all(0 <= i <= 65535 for i in values):
        raise ValueError('int16 value is out of range')
      factor = 65536
      bits = 128
    else:
      raise ValueError('not IPv4 or IPv6 address')
  except ValueError as e:
    raise ValueError('%r is not an IP address (%s)' % (ipstr, e))
  value = 0L
  for i in values:
    value = value * factor + i
  return IP(bits, value)


def ip_to_string(ip):
  """Given an instance of IP returns a string representing it."""
  if ip.bits == 32:
    factor = 256
    count = 4
    conv = str
    sep = '.'
  elif ip.bits == 128:
    factor = 65536
    count = 8
    conv = lambda x: '%x' % x
    sep = ':'
  else:
    raise ValueError('Unknown type of IP with bits %d' % ip.bits)
  fields = []
  value = ip.value
  for _ in xrange(count):
    value, mod = divmod(value, factor)
    fields.append(conv(mod))
  if value:
    raise ValueError('IP int is longer than expected')
  return sep.join(reversed(fields))


def normalize_ip(ip_str):
  """Converts IP string to a normal form, e.g 1.01.1.1 -> 1.1.1.1.

  Raises ValueError if ip_str is not a valid IP address.
  """
  return ip_to_string(ip_from_string(ip_str))


def subnet_from_string(subnet):
  """Given a string 'xxx.xxx.xxx.xxx/xx' return Subnet instance.

  Raises ValueError if |subnet| is not recognized as IPv4 or IPv6 subnet.
  """
  # Accept single IPs too.
  if '/' not in subnet:
    base_ip = ip_from_string(subnet)
    mask = (1 << base_ip.bits) - 1
    return Subnet(base_ip.bits, base_ip.value, mask)

  try:
    ip_str, mask_bits = subnet.split('/', 1)
    mask_bits = int(mask_bits)
  except ValueError:
    raise ValueError('Not a valid subnet string')

  base_ip = ip_from_string(ip_str)
  if mask_bits < 0 or mask_bits > base_ip.bits:
    raise ValueError('Not a valid subnet string')

  # |mask| is 0xffff000 where number of '1' bits in 'ffff' part is equal to
  # |mask_bits|. We build 0xffffffff and then zero least significant bits.
  full = (1 << base_ip.bits) - 1
  ignore_bits = base_ip.bits - mask_bits
  mask = (full >> ignore_bits) << ignore_bits
  return Subnet(base_ip.bits, base_ip.value & mask, mask)


def subnet_to_string(subnet):
  """Given Subnet instance returns a string describing it."""
  # Count number of least significant zero bits in the mask.
  mask = subnet.mask
  i = 0
  while not (mask & 1) and i < subnet.bits:
    mask = mask >> 1
    i += 1
  base_ip = IP(subnet.bits, subnet.base)
  return '%s/%d' % (ip_to_string(base_ip), subnet.bits - i)


def normalize_subnet(subnet_str):
  """Converts subnet string to a normal form, e.g 1.01.1.1/08 -> 1.0.0.0/8.

  It will also zero non significant bits of an IP address used as a subnet base.

  Raises ValueError if subnet_str is not a valid subnet.
  """
  return subnet_to_string(subnet_from_string(subnet_str))


def is_in_subnet(ip, subnet):
  """True if given IP instance belongs to Subnet."""
  return ip.bits == subnet.bits and (ip.value & subnet.mask) == subnet.base
