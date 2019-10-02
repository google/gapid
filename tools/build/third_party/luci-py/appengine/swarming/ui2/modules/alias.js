// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// applyAlias will transform 'value' based on what 'key' is.
// This aliasing should be used for human-displayable only,
// not, for example, on the filters that are sent to the backend.
export function applyAlias(value, key) {
  if (!aliasMap[key] || value === 'none') {
    return value;
  }
  let alias = aliasMap[key][value];
  if (key === 'gpu') {
    // the longest gpu string looks like [pci id]-[driver version],
    // so we trim that off if needed.
    const trimmed = value.split('-')[0];
    alias = aliasMap[key][trimmed];
  }
  if (!alias) {
    return value;
  }
  return `${alias} (${value})`;
}

const ANDROID_ALIASES = {
  'angler': 'Nexus 6p',
  'athene': 'Moto G4',
  'blueline': 'Pixel 3',
  'bullhead': 'Nexus 5X',
  'crosshatch': 'Pixel 3 XL',
  'darcy': 'NVIDIA Shield [2017]',
  'dragon': 'Pixel C',
  'flo': 'Nexus 7 [2013]',
  'flounder': 'Nexus 9',
  'foster': 'NVIDIA Shield [2015]',
  'fugu': 'Nexus Player',
  'gce_x86': 'Android on GCE',
  'goyawifi': 'Galaxy Tab 3',
  'grouper': 'Nexus 7 [2012]',
  'hammerhead': 'Nexus 5',
  'herolte': 'Galaxy S7 [Global]',
  'heroqlteatt': 'Galaxy S7 [AT&T]',
  'j5xnlte': 'Galaxy J5',
  'm0': 'Galaxy S3',
  'mako': 'Nexus 4',
  'manta': 'Nexus 10',
  'marlin': 'Pixel XL',
  'sailfish': 'Pixel',
  'shamu': 'Nexus 6',
  'sprout': 'Android One',
  'starlte': 'Galaxy S9',
  'taimen': 'Pixel 2 XL',
  'walleye': 'Pixel 2',
  'zerofltetmo': 'Galaxy S6',
};

const GPU_ALIASES = {
  '1002':      'AMD',
  '1002:6613': 'AMD Radeon R7 240',
  '1002:6646': 'AMD Radeon R9 M280X',
  '1002:6779': 'AMD Radeon HD 6450/7450/8450',
  '1002:679e': 'AMD Radeon HD 7800',
  '1002:6821': 'AMD Radeon HD 8870M',
  '1002:683d': 'AMD Radeon HD 7770/8760',
  '1002:9830': 'AMD Radeon HD 8400',
  '1002:9874': 'AMD Carrizo',
  '1a03':      'ASPEED',
  '1a03:2000': 'ASPEED Graphics Family',
  '102b':      'Matrox',
  '102b:0522': 'Matrox MGA G200e',
  '102b:0532': 'Matrox MGA G200eW',
  '102b:0534': 'Matrox G200eR2',
  '10de':      'NVIDIA',
  '10de:08a4': 'NVIDIA GeForce 320M',
  '10de:08aa': 'NVIDIA GeForce 320M',
  '10de:0a65': 'NVIDIA GeForce 210',
  '10de:0fe9': 'NVIDIA GeForce GT 750M Mac Edition',
  '10de:0ffa': 'NVIDIA Quadro K600',
  '10de:104a': 'NVIDIA GeForce GT 610',
  '10de:11c0': 'NVIDIA GeForce GTX 660',
  '10de:1244': 'NVIDIA GeForce GTX 550 Ti',
  '10de:1401': 'NVIDIA GeForce GTX 960',
  '10de:1ba1': 'NVIDIA GeForce GTX 1070',
  '10de:1cb3': 'NVIDIA Quadro P400',
  '8086':      'Intel',
  '8086:0046': 'Intel Ironlake HD Graphics',
  '8086:0102': 'Intel Sandy Bridge HD Graphics 2000',
  '8086:0116': 'Intel Sandy Bridge HD Graphics 3000',
  '8086:0166': 'Intel Ivy Bridge HD Graphics 4000',
  '8086:0412': 'Intel Haswell HD Graphics 4600',
  '8086:041a': 'Intel Haswell HD Graphics',
  '8086:0a16': 'Intel Haswell HD Graphics 4400',
  '8086:0a26': 'Intel Haswell HD Graphics 5000',
  '8086:0a2e': 'Intel Haswell Iris Graphics 5100',
  '8086:0d26': 'Intel Haswell Iris Pro Graphics 5200',
  '8086:0f31': 'Intel Bay Trail HD Graphics',
  '8086:1616': 'Intel Broadwell HD Graphics 5500',
  '8086:161e': 'Intel Broadwell HD Graphics 5300',
  '8086:1626': 'Intel Broadwell HD Graphics 6000',
  '8086:162b': 'Intel Broadwell Iris Graphics 6100',
  '8086:1912': 'Intel Skylake HD Graphics 530',
  '8086:191e': 'Intel Skylake HD Graphics 515',
  '8086:1926': 'Intel Skylake Iris 540/550',
  '8086:193b': 'Intel Skylake Iris Pro 580',
  '8086:22b1': 'Intel Braswell HD Graphics',
  '8086:3ea5': 'Intel Coffee Lake Iris Plus Graphics 655',
  '8086:5912': 'Intel Kaby Lake HD Graphics 630',
  '8086:591e': 'Intel Kaby Lake HD Graphics 615',
  '8086:5926': 'Intel Kaby Lake Iris Plus Graphics 640',
};

const DEVICE_ALIASES = {
  'iPad4,1':   'iPad Air',
  'iPad5,1':   'iPad mini 4',
  'iPad6,3':   'iPad Pro [9.7 in]',
  'iPhone7,2': 'iPhone 6',
  'iPhone9,1': 'iPhone 7',
}

const aliasMap = {
  'device': DEVICE_ALIASES,
  'device_type': ANDROID_ALIASES,
  'gpu': GPU_ALIASES,
};

const oldStyle = /.+\((.+)\)/;

/** handle legacy filters goes through a list of filters and
 *  transforms any from the "old-style" (e.g. Polymer version)
 *  to the new version.
 *
 *  @param {Array<string>} filters - a list of colon-separated key-values.
 *
 *  @return {Array<string>} - the cleaned up filters.
 */
export function handleLegacyFilters(filters) {
  if (!filters) {
    return [];
  }
  return filters.map((f) => {
    const potentialKey = f.split(':')[0];
    if (aliasMap[potentialKey]) {
      // This could be new/correct-style:
      //   "gpu:10de:1cb3-415.27"
      // or the old-style with the alias:
      //   "gpu:NVIDIA Quadro P400 (10de:1cb3-415.27)"
      // If it's the old-style, convert it to new-style.
      const found = f.match(oldStyle);
      if (found) {
        return potentialKey + ':' + found[1];
      } else {
        return f;
      }
    } else {
      return f;
    }
  });
}

/** maybeApplyAlias will take a filter (e.g. foo:bar) and apply
 *  the alias to it inline, returning it to be displayed on the UI.
 *  This means we can display it in a human-friendly way, without
 *  the complexity of handling the alias when storing URL params
 *  or making API requests.
 */
export function maybeApplyAlias(filter) {
  const idx = filter.indexOf(':');
  if (idx < 0) {
    return filter;
  }
  const key = filter.substring(0, idx);
  const value = filter.substring(idx+1);
  // remove -tag for tasks if it exists
  const trimmed = key.split('-tag')[0];
  return `${key}:${applyAlias(value, trimmed)}`;
}
