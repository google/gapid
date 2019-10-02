// Copyright 2016 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// TODO(kjlubick): add tests for this code

this.swarming = this.swarming || {};
this.swarming.alias = this.swarming.alias || (function(){
  var ANDROID_ALIASES = {
    "angler": "Nexus 6p",
    "athene": "Moto G4",
    "blueline": "Pixel 3",
    "bullhead": "Nexus 5X",
    "crosshatch": "Pixel 3 XL",
    "darcy": "NVIDIA Shield [2017]",
    "dragon": "Pixel C",
    "flo": "Nexus 7 [2013]",
    "flounder": "Nexus 9",
    "foster": "NVIDIA Shield [2015]",
    "fugu": "Nexus Player",
    "gce_x86": "Android on GCE",
    "goyawifi": "Galaxy Tab 3",
    "grouper": "Nexus 7 [2012]",
    "hammerhead": "Nexus 5",
    "herolte": "Galaxy S7 [Global]",
    "heroqlteatt": "Galaxy S7 [AT&T]",
    "j5xnlte": "Galaxy J5",
    "m0": "Galaxy S3",
    "mako": "Nexus 4",
    "manta": "Nexus 10",
    "marlin": "Pixel XL",
    "sailfish": "Pixel",
    "shamu": "Nexus 6",
    "sprout": "Android One",
    "starlte": "Galaxy S9",
    "taimen": "Pixel 2 XL",
    "walleye": "Pixel 2",
    "zerofltetmo": "Galaxy S6",
  };

  var UNKNOWN = "unknown";

  var GPU_ALIASES = {
    "1002":      "AMD",
    "1002:6613": "AMD Radeon R7 240",
    "1002:6646": "AMD Radeon R9 M280X",
    "1002:6779": "AMD Radeon HD 6450/7450/8450",
    "1002:679e": "AMD Radeon HD 7800",
    "1002:6821": "AMD Radeon HD 8870M",
    "1002:683d": "AMD Radeon HD 7770/8760",
    "1002:9830": "AMD Radeon HD 8400",
    "1002:9874": "AMD Carrizo",
    "1a03":      "ASPEED",
    "1a03:2000": "ASPEED Graphics Family",
    "102b":      "Matrox",
    "102b:0522": "Matrox MGA G200e",
    "102b:0532": "Matrox MGA G200eW",
    "102b:0534": "Matrox G200eR2",
    "10de":      "NVIDIA",
    "10de:08a4": "NVIDIA GeForce 320M",
    "10de:08aa": "NVIDIA GeForce 320M",
    "10de:0a65": "NVIDIA GeForce 210",
    "10de:0fe9": "NVIDIA GeForce GT 750M Mac Edition",
    "10de:0ffa": "NVIDIA Quadro K600",
    "10de:104a": "NVIDIA GeForce GT 610",
    "10de:11c0": "NVIDIA GeForce GTX 660",
    "10de:1244": "NVIDIA GeForce GTX 550 Ti",
    "10de:1401": "NVIDIA GeForce GTX 960",
    "10de:1ba1": "NVIDIA GeForce GTX 1070",
    "10de:1cb3": "NVIDIA Quadro P400",
    "8086":      "Intel",
    "8086:0046": "Intel Ironlake HD Graphics",
    "8086:0102": "Intel Sandy Bridge HD Graphics 2000",
    "8086:0116": "Intel Sandy Bridge HD Graphics 3000",
    "8086:0166": "Intel Ivy Bridge HD Graphics 4000",
    "8086:0412": "Intel Haswell HD Graphics 4600",
    "8086:041a": "Intel Haswell HD Graphics",
    "8086:0a16": "Intel Haswell HD Graphics 4400",
    "8086:0a26": "Intel Haswell HD Graphics 5000",
    "8086:0a2e": "Intel Haswell Iris Graphics 5100",
    "8086:0d26": "Intel Haswell Iris Pro Graphics 5200",
    "8086:0f31": "Intel Bay Trail HD Graphics",
    "8086:1616": "Intel Broadwell HD Graphics 5500",
    "8086:161e": "Intel Broadwell HD Graphics 5300",
    "8086:1626": "Intel Broadwell HD Graphics 6000",
    "8086:162b": "Intel Broadwell Iris Graphics 6100",
    "8086:1912": "Intel Skylake HD Graphics 530",
    "8086:191e": "Intel Skylake HD Graphics 515",
    "8086:1926": "Intel Skylake Iris 540/550",
    "8086:193b": "Intel Skylake Iris Pro 580",
    "8086:22b1": "Intel Braswell HD Graphics",
    "8086:3ea5": "Intel Coffee Lake Iris Plus Graphics 655",
    "8086:5912": "Intel Kaby Lake HD Graphics 630",
    "8086:591e": "Intel Kaby Lake HD Graphics 615",
    "8086:5926": "Intel Kaby Lake Iris Plus Graphics 640",
  }

  // Taken from http://developer.android.com/reference/android/os/BatteryManager.html
  var BATTERY_HEALTH_ALIASES = {
    1: "Unknown",
    2: "Good",
    3: "Overheated",
    4: "Dead",
    5: "Over Voltage",
    6: "Unspecified Failure",
    7: "Too Cold",
  }

  var BATTERY_STATUS_ALIASES = {
    1: "Unknown",
    2: "Charging",
    3: "Discharging",
    4: "Not Charging",
    5: "Full",
  }

  // Created from experience and cross-referenced with
  // https://www.theiphonewiki.com/wiki/Models
  var DEVICE_ALIASES = {
    "iPad4,1":   "iPad Air",
    "iPad5,1":   "iPad mini 4",
    "iPad6,3":   "iPad Pro [9.7 in]",
    "iPhone7,2": "iPhone 6",
    "iPhone9,1": "iPhone 7",
  }

  // For consistency, all aliases are displayed like:
  // Nexus 5X (bullhead)
  // This regex matches a string like "ALIAS (ORIG)", with ORIG as group 1.
  var ALIAS_REGEXP = /.+ \((.*)\)/;

  var alias = {};

  alias.DIMENSIONS_WITH_ALIASES = ["device_type", "gpu", "battery_health", "battery_status", "device"];

  alias.android = function(dt) {
    return ANDROID_ALIASES[dt] || UNKNOWN;
  };

  alias.battery_health = function(bh) {
    return BATTERY_HEALTH_ALIASES[bh] || UNKNOWN;
  };

  alias.battery_status = function(bs) {
    return BATTERY_STATUS_ALIASES[bs] || UNKNOWN;
  };

  alias.device = function(dt) {
    return DEVICE_ALIASES[dt] || UNKNOWN;
  };

  alias.gpu = function(gpu) {
    if (!gpu || !gpu.split) {
      return UNKNOWN;
    }
    gpu = gpu.split("-")[0];
    return GPU_ALIASES[gpu] || UNKNOWN;
  };


  // alias.apply tries to alias the string "orig" based on what "type" it is.
  // If type is in DIMENSIONS_WITH_ALIASES, the appropriate alias (e.g. gpu)
  // is automatically applied.  Otherwise, "type" is treated as the alias.
  // If type is known, but there is no matching alias (e.g. for gpu: FOOBAR)
  // the original will be returned.
  alias.apply = function(orig, type) {
    var aliaser = aliasMap[type];
    if (!aliaser) {
      // treat type as the actual alias
      return type + " ("+orig+")";
    }
    var alias = aliaser(orig);
    if (alias !== UNKNOWN) {
      return alias + " ("+orig+")";
    }
    return orig;
  };

  alias.has = function(type) {
    return !!aliasMap[type];
  };

  // alias.unapply will return the base dimension/state with its alias removed
  // if it had one.  This is handy for sorting and filtering.
  alias.unapply = function(str) {
    var match = ALIAS_REGEXP.exec(str);
    if (match) {
      return match[1];
    }
    return str;
  };

  var aliasMap = {
    "battery_health": alias.battery_health,
    "battery_status": alias.battery_status,
    "device": alias.device,
    "device_type": alias.android,
    "gpu": alias.gpu,
  }

  return alias;
})();
