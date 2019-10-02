# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""GPU specific utility functions."""


AMD = u'1002'
ASPEED = u'1a03'
INTEL = u'8086'
MAXTROX = u'102b'
NVIDIA = u'10de'


_VENDOR_MAPPING = {
  AMD: (
    u'AMD',
    {
      # http://developer.amd.com/resources/ati-catalyst-pc-vendor-id-1002-li/
      u'6613': u'Radeon R7 240',   # The table is incorrect
      u'6646': u'Radeon R9 M280X',
      u'6779': u'Radeon HD 6450/7450/8450',
      u'679e': u'Radeon HD 7800',
      u'6821': u'Radeon R8 M370X', # 'HD 8800M' or 'R7 M380' based on rev_id
      u'683d': u'Radeon HD 7700',
      u'9830': u'Radeon HD 8400',
      u'9874': u'Carrizo',
    }),
  ASPEED: (
    u'ASPEED',
    {
      # https://pci-ids.ucw.cz/read/PC/1a03/2000
      # It seems all ASPEED graphics cards use the same device id
      # (for driver reasons?)
      u'2000': u'ASPEED Graphics Family',
    }),
  INTEL: (
    u'Intel',
    {
      # http://downloadmirror.intel.com/23188/eng/config.xml
      u'0046': u'Ironlake HD Graphics',
      u'0102': u'Sandy Bridge HD Graphics 2000',
      u'0116': u'Sandy Bridge HD Graphics 3000',
      u'0166': u'Ivy Bridge HD Graphics 4000',
      u'0412': u'Haswell HD Graphics 4600',
      u'041a': u'Haswell HD Graphics',
      u'0a16': u'Intel Haswell HD Graphics 4400',
      u'0a26': u'Haswell HD Graphics 5000',
      u'0a2e': u'Haswell Iris Graphics 5100',
      u'0d26': u'Haswell Iris Pro Graphics 5200',
      u'0f31': u'Bay Trail HD Graphics',
      u'1616': u'Broadwell HD Graphics 5500',
      u'161e': u'Broadwell HD Graphics 5300',
      u'1626': u'Broadwell HD Graphics 6000',
      u'162b': u'Broadwell Iris Graphics 6100',
      u'1912': u'Skylake HD Graphics 530',
      u'191e': u'Skylake HD Graphics 515',
      u'1926': u'Skylake Iris 540/550',
      u'193b': u'Skylake Iris Pro 580',
      u'22b1': u'Braswell HD Graphics',
      u'5912': u'Kaby Lake HD Graphics 630',
      u'591e': u'Kaby Lake HD Graphics 615',
      u'5926': u'Kaby Lake Iris Plus Graphics 640',
    }),
  MAXTROX: (
    u'Matrox',
    {
      u'0522': u'MGA G200e',
      u'0532': u'MGA G200eW',
      u'0534': u'G200eR2',
    }),
  NVIDIA: (
    u'Nvidia',
    {
      # ftp://download.nvidia.com/XFree86/Linux-x86_64/352.21/README/README.txt
      u'06fa': u'Quadro NVS 450',
      u'08a4': u'GeForce 320M',
      u'08aa': u'GeForce 320M',
      u'0a65': u'GeForce 210',
      u'0df8': u'Quadro 600',
      u'0fd5': u'GeForce GT 650M',
      u'0fe9': u'GeForce GT 750M Mac Edition',
      u'0ffa': u'Quadro K600',
      u'104a': u'GeForce GT 610',
      u'11c0': u'GeForce GTX 660',
      u'1244': u'GeForce GTX 550 Ti',
      u'1401': u'GeForce GTX 960',
      u'1ba1': u'GeForce GTX 1070',
      u'1cb3': u'Quadro P400',
    }),
}


def vendor_name_to_id(ven_name):
  """Uses an internal lookup table to return the vendor ID for known
  vendor names.

  Returns:
    a string, or None if unknown.
  """
  # macOS 10.13 doesn't provide the vendor ID any more, so support reverse
  # lookups on vendor name.
  lower_name = ven_name.lower()
  for k, v in _VENDOR_MAPPING.iteritems():
    if lower_name == v[0].lower():
      return k
  return None


def ids_to_names(ven_id, ven_name, dev_id, dev_name):
  """Uses an internal lookup table to return canonical names when known.

  Returns:
    tuple(vendor name, device name).
  """
  m = _VENDOR_MAPPING.get(ven_id)
  if not m:
    return ven_name, dev_name
  return m[0], m[1].get(dev_id, dev_name)
