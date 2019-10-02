#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from components.auth import ipaddr
from test_support import test_case


class IpAddrTest(test_case.TestCase):
  def test_ip_from_string_v4_ok(self):
    self.assertEqual(
        ipaddr.IP(32, 0), ipaddr.ip_from_string('0.0.0.0'))
    self.assertEqual(
        ipaddr.IP(32, 0xffffffff), ipaddr.ip_from_string('255.255.255.255'))
    self.assertEqual(
        ipaddr.IP(32, 0x7f000001), ipaddr.ip_from_string('127.0.0.1'))
    self.assertEqual(
        ipaddr.IP(32, 0x7f000001), ipaddr.ip_from_string('127.000.000.001'))

  def test_ip_from_string_v4_bad(self):
    with self.assertRaises(ValueError):
      ipaddr.ip_from_string('')
    with self.assertRaises(ValueError):
      ipaddr.ip_from_string('0.0.0')
    with self.assertRaises(ValueError):
      ipaddr.ip_from_string('127.0.0.a')
    with self.assertRaises(ValueError):
      ipaddr.ip_from_string('256.0.0.1')

  def test_ip_from_string_v6_ok(self):
    self.assertEqual(
        ipaddr.IP(128, 0), ipaddr.ip_from_string('0:0:0:0:0:0:0:0'))
    self.assertEqual(
        ipaddr.IP(128, 2**128 - 1),
        ipaddr.ip_from_string('ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff'))
    self.assertEqual(
        ipaddr.IP(128, 1), ipaddr.ip_from_string('0:0:0:0:0:0:0:1'))
    self.assertEqual(
        ipaddr.IP(128, 0xffff0000000000000000000000000000L),
        ipaddr.ip_from_string('ffff:0:0:0:0:0:0:0'))

  def test_ip_from_string_v6_bad(self):
    with self.assertRaises(ValueError):
      ipaddr.ip_from_string('0:0:0:0:0:0:0')
    with self.assertRaises(ValueError):
      ipaddr.ip_from_string('0:0:0:0:0:0:0:00gg')

  def test_ip_from_string_v6_omitting_zeros_ok(self):
    self.assertEqual(ipaddr.IP(128, 1), ipaddr.ip_from_string('::1'))
    self.assertEqual(ipaddr.IP(128, 0), ipaddr.ip_from_string('::0'))
    self.assertEqual(ipaddr.IP(128, 0), ipaddr.ip_from_string('::'))
    self.assertEqual(
        ipaddr.ip_from_string('ffff:ffff:ffff:0:ffff:ffff:ffff:ffff'),
        ipaddr.ip_from_string('ffff:ffff:ffff::ffff:ffff:ffff:ffff'))
    self.assertEqual(
        ipaddr.ip_from_string('ffff:ffff:0:0:0:0:0:ffff'),
        ipaddr.ip_from_string('ffff:ffff::ffff'))
    self.assertEqual(
        ipaddr.ip_from_string('ffff:0:0:0:0:0:0:0'),
        ipaddr.ip_from_string('ffff::'))


  def test_ip_from_string_v6_omitting_zeros_bad(self):
    with self.assertRaises(ValueError):
      ipaddr.ip_from_string('::1::')
    with self.assertRaises(ValueError):
      ipaddr.ip_from_string('0:0:0:0:0:0:0::0')

  def test_ip_to_string_v4_ok(self):
    call = lambda val: ipaddr.ip_to_string(ipaddr.IP(32, val))
    self.assertEqual('0.0.0.0', call(0))
    self.assertEqual('255.255.255.255', call(2**32 - 1))
    self.assertEqual('0.0.0.255', call(255))
    self.assertEqual('127.0.0.1', call(0x7f000001))

  def test_ip_to_string_v4_bad(self):
    with self.assertRaises(ValueError):
      ipaddr.ip_to_string(ipaddr.IP(8, 0))
    with self.assertRaises(ValueError):
      ipaddr.ip_to_string(ipaddr.IP(32, 2**32))

  def test_ip_to_string_v6_ok(self):
    call = lambda val: ipaddr.ip_to_string(ipaddr.IP(128, val))
    self.assertEqual('0:0:0:0:0:0:0:0', call(0))
    self.assertEqual('ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff', call(2**128-1))
    self.assertEqual('0:0:0:0:0:0:0:ffff', call(0xffff))
    self.assertEqual(
        'ffff:0:0:0:0:0:0:0', call(0xffff0000000000000000000000000000L))

  def test_ip_to_string_v6_bad(self):
    with self.assertRaises(ValueError):
      ipaddr.ip_to_string(ipaddr.IP(128, 2**128))

  def test_normalize_ip(self):
    self.assertEqual('127.0.0.1', ipaddr.normalize_ip('127.000.000.001'))
    self.assertEqual(
        '0:0:0:0:0:0:0:1', ipaddr.normalize_ip('00:00:00:00:00:00:00:1'))

  def test_subnet_from_string_v4(self):
    self.assertEqual(
        ipaddr.Subnet(32, 0x7f000001, 0xffffffff),
        ipaddr.subnet_from_string('127.0.0.1'))
    self.assertEqual(
        ipaddr.Subnet(32, 0xfffefdfc, 0xffffffff),
        ipaddr.subnet_from_string('255.254.253.252/32'))
    self.assertEqual(
        ipaddr.Subnet(32, 0xfffefd00, 0xffffff00),
        ipaddr.subnet_from_string('255.254.253.252/24'))

  def test_subnet_from_string_v6(self):
    self.assertEqual(
        ipaddr.Subnet(128, 1, 0xffffffffffffffffffffffffffffffffL),
        ipaddr.subnet_from_string('0:0:0:0:0:0:0:1'))
    self.assertEqual(
        ipaddr.Subnet(
            128,
            0xfffffffefffdfffcfffbfffafff0fff9L,
            0xffffffffffffffffffffffffffffffffL),
        ipaddr.subnet_from_string(
            'ffff:fffe:fffd:fffc:fffb:fffa:fff0:fff9/128'))
    self.assertEqual(
        ipaddr.Subnet(
            128,
            0xfffffffefffdfffcfffbfffa00000000L,
            0xffffffffffffffffffffffff00000000L),
        ipaddr.subnet_from_string('ffff:fffe:fffd:fffc:fffb:fffa:fff0:fff9/96'))

  def test_subnet_from_string_bad(self):
    with self.assertRaises(ValueError):
      ipaddr.subnet_from_string('256.0.0.1')
    with self.assertRaises(ValueError):
      ipaddr.subnet_from_string('127.0.0.1/abc')
    with self.assertRaises(ValueError):
      ipaddr.subnet_from_string('256.0.0.1/32')
    with self.assertRaises(ValueError):
      ipaddr.subnet_from_string('127.0.0.1/33')

  def test_subnet_to_string_v4(self):
    call = lambda base, mask: (
        ipaddr.subnet_to_string(ipaddr.Subnet(32, base, mask)))
    self.assertEqual('127.0.0.1/32', call(0x7f000001, 0xffffffff))
    self.assertEqual('0.0.0.0/0', call(0, 0))
    self.assertEqual('255.254.253.0/24', call(0xfffefd00, 0xffffff00))

  def test_subnet_to_string_v6(self):
    call = lambda base, mask: (
        ipaddr.subnet_to_string(ipaddr.Subnet(128, base, mask)))
    self.assertEqual(
        '0:0:0:0:0:0:0:1/128',
        call(1, 0xffffffffffffffffffffffffffffffffL))
    self.assertEqual(
        'ffff:fffe:fffd:fffc:fffb:fffa:fff0:fff9/128',
        call(
            0xfffffffefffdfffcfffbfffafff0fff9L,
            0xffffffffffffffffffffffffffffffffL))
    self.assertEqual(
        'ffff:fffe:fffd:fffc:fffb:fffa:0:0/96',
        call(
            0xfffffffefffdfffcfffbfffa00000000L,
            0xffffffffffffffffffffffff00000000L))

  def test_subnet_to_string_bad(self):
    with self.assertRaises(ValueError):
      ipaddr.subnet_to_string(ipaddr.Subnet(15, 0, 0))

  def test_normalize_subnet(self):
    self.assertEqual('1.0.0.0/8', ipaddr.normalize_subnet('1.01.1.1/08'))
    self.assertEqual(
        'ff00:0:0:0:0:0:0:0/8',
        ipaddr.normalize_subnet('ffff:ffff:ffff:ffff:ffff:ffff:ffff:0000/008'))

  def test_is_in_subnet(self):
    call = lambda ip, subnet: (
        ipaddr.is_in_subnet(
            ipaddr.ip_from_string(ip),
            ipaddr.subnet_from_string(subnet)))

    self.assertTrue(call('127.0.0.1', '127.0.0.1/32'))
    self.assertTrue(call('192.168.0.25', '192.168.0.0/24'))
    self.assertFalse(call('192.168.0.25', '192.168.1.0/24'))
    self.assertFalse(call('192.168.0.25', '192.168.0.0/31'))
    self.assertTrue(call('255.255.255.255', '0.0.0.0/0'))

    self.assertTrue(call('0:0:0:0:0:0:0:1', '0:0:0:0:0:0:0:1/128'))
    self.assertTrue(call(
        'ffff:fffe:fffd:fffc:fffb:fffa:fff0:1234',
        'ffff:fffe:fffd:fffc:fffb:fffa:fff0:0/112'))
    self.assertFalse(call(
        'ffff:fffe:fffd:fffc:fffb:fffa:fff1:1234',
        'ffff:fffe:fffd:fffc:fffb:fffa:fff0:0/112'))
    self.assertFalse(call(
        'ffff:fffe:fffd:fffc:fffb:fffa:fff0:2',
        'ffff:fffe:fffd:fffc:fffb:fffa:fff0:0/127'))

    self.assertFalse(call('0:0:0:0:0:0:0:0', '0.0.0.0/32'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
