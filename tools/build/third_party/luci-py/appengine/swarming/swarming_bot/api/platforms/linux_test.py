#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

import test_env_platforms
test_env_platforms.setup_test_env()

from utils import tools

import linux


EXYNOS_CPU_INFO = r"""
Processor : ARMv7 Processor rev 4 (v7l)
processor : 0
BogoMIPS  : 1694.10

processor : 1
BogoMIPS  : 1694.10

Features  : swp half thumb fastmult vfp edsp thumbee neon vfpv3 tls vfpv4 idiva idivt
CPU implementer : 0x41
CPU architecture: 7
CPU variant : 0x0
CPU part  : 0xc0f
CPU revision  : 4

Hardware  : SAMSUNG EXYNOS5 (Flattened Device Tree)
Revision  : 0000
Serial    : 0000000000000000
"""


CAVIUM_CPU_INFO = r"""
processor : 0
BogoMIPS  : 200.00
Features  : fp asimd evtstrm aes pmull sha1 sha2 crc32 atomics
CPU implementer : 0x43
CPU architecture: 8
CPU variant : 0x1
CPU part  : 0x0a1
CPU revision  : 1

processor : 1
BogoMIPS  : 200.00
Features  : fp asimd evtstrm aes pmull sha1 sha2 crc32 atomics
CPU implementer : 0x43
CPU architecture: 8
CPU variant : 0x1
CPU part  : 0x0a1
CPU revision  : 1
"""


MIPS64_CPU_INFO = r"""
chrome-bot@build23-b3:/tmp/1$ cat /proc/cpuinfo
system type             : Unsupported Board (CN6120p1.1-1000-NSP)
machine                 : Unknown
processor               : 0
cpu model               : Cavium Octeon II V0.1
BogoMIPS                : 2000.00
wait instruction        : yes
microsecond timers      : yes
tlb_entries             : 128
extra interrupt vector  : yes
hardware watchpoint     : yes, count: 2, address/irw mask: [0x0ffc, 0x0ffb]
isa                     : mips2 mips3 mips4 mips5 mips64r2
ASEs implemented        :
shadow register sets    : 1
kscratch registers      : 3
package                 : 0
core                    : 0
VCED exceptions         : not available
VCEI exceptions         : not available

processor               : 1
cpu model               : Cavium Octeon II V0.1
BogoMIPS                : 2000.00
wait instruction        : yes
microsecond timers      : yes
tlb_entries             : 128
extra interrupt vector  : yes
hardware watchpoint     : yes, count: 2, address/irw mask: [0x0ffc, 0x0ffb]
isa                     : mips2 mips3 mips4 mips5 mips64r2
ASEs implemented        :
shadow register sets    : 1
kscratch registers      : 3
package                 : 0
core                    : 1
VCED exceptions         : not available
VCEI exceptions         : not available
"""


class TestCPUInfo(unittest.TestCase):
  def get_cpuinfo(self, text):
    tools.clear_cache(linux.get_cpuinfo)
    linux._read_cpuinfo = lambda: text
    return linux.get_cpuinfo()

  def test_get_cpuinfo_exynos(self):
    self.assertEqual({
      u'flags': [
        u'edsp', u'fastmult', u'half', u'idiva', u'idivt', u'neon', u'swp',
        u'thumb', u'thumbee', u'tls', u'vfp', u'vfpv3', u'vfpv4',
      ],
      u'model': (0, 3087, 4),
      u'name': u'SAMSUNG EXYNOS5',
      u'revision': u'0000',
      u'serial': u'',
      u'vendor': u'ARMv7 Processor rev 4 (v7l)',
    }, self.get_cpuinfo(EXYNOS_CPU_INFO))

  def test_get_cpuinfo_cavium(self):
    self.assertEqual({
      u'flags': [
        u'aes', u'asimd', u'atomics', u'crc32', u'evtstrm',  u'fp',  u'pmull',
        u'sha1', u'sha2',
      ],
      u'model': (1, 161, 1),
      u'vendor': u'N/A',
    }, self.get_cpuinfo(CAVIUM_CPU_INFO))

  def test_get_cpuinfo_mips(self):
    self.assertEqual({
      u'flags': [u'mips2', u'mips3', u'mips4', u'mips5', u'mips64r2'],
      u'name': 'Cavium Octeon II V0.1',
    }, self.get_cpuinfo(MIPS64_CPU_INFO))

  @unittest.skipIf(sys.platform != 'linux2', 'linux only test')
  def test_get_num_processors(self):
    self.assertTrue(linux.get_num_processors() != 0)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
