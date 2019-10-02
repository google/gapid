// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

export function botData(url, opts) {
  let botId = url.match('/bot/(.+)/get');
  botId = botId[1] || 'running';
  return botDataMap[botId];
}

export const botDataMap = {
  'running': {
    "authenticated_as": "bot:running.chromium.org",
    "dimensions": [
      {
        "value": [
          "vpython"
        ],
        "key": "caches"
      },
      {
        "value": [
          "8"
        ],
        "key": "cores"
      },
      {
        "value": [
          "x86",
          "x86-64",
          "x86-64-E3-1230_v5"
        ],
        "key": "cpu"
      },
      {
        "value": [
          "0"
        ],
        "key": "gce"
      },
      {
        "value": [
          "10de",
          "10de:1cb3",
          "10de:1cb3-25.21.14.1678"
        ],
        "key": "gpu"
      },
      {
        "value": [
          "build16-a9"
        ],
        "key": "id"
      },
      {
        "value": [
          "high"
        ],
        "key": "integrity"
      },
      {
        "value": [
          "en_US.cp1252"
        ],
        "key": "locale"
      },
      {
        "value": [
          "n1-standard-8"
        ],
        "key": "machine_type"
      },
      {
        "value": [
          "Windows",
          "Windows-10",
          "Windows-10-16299.309"
        ],
        "key": "os"
      },
      {
        "value": [
          "Skia"
        ],
        "key": "pool"
      },
      {
        "value": [
          "2.7.13"
        ],
        "key": "python"
      },
      {
        "value": [
          "1"
        ],
        "key": "rack"
      },
      {
        "value": [
          "4085-c81638b"
        ],
        "key": "server_version"
      },
      {
        "value": [
          "us",
          "us-mtv",
          "us-mtv-chops",
          "us-mtv-chops-a",
          "us-mtv-chops-a-9"
        ],
        "key": "zone"
      }
    ],
    "task_id": "42fb00e06d95be11",
    "external_ip": "70.32.137.220",
    "is_dead": false,
    "quarantined": false,
    "deleted": false,
    "state": "{\"audio\":[\"NVIDIA High Definition Audio\"],\"bot_group_cfg_version\":\"hash:d50e0a198b5ee4\",\"cost_usd_hour\":0.7575191297743056,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"C:\\\\b\\\\s\",\"cygwin\":[false],\"disks\":{\"c:\\\\\":{\"free_mb\":690166.5,\"size_mb\":763095.0}},\"env\":{\"PATH\":\"C:\\\\Windows\\\\system32;C:\\\\Windows;C:\\\\Windows\\\\System32\\\\Wbem;C:\\\\Windows\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\;c:\\\\Tools;C:\\\\CMake\\\\bin;C:\\\\Program Files\\\\Puppet Labs\\\\Puppet\\\\bin;C:\\\\Users\\\\chrome-bot\\\\AppData\\\\Local\\\\Microsoft\\\\WindowsApps\"},\"files\":{\"c:\\\\Users\\\\chrome-bot\\\\ntuser.dat\":1310720},\"gpu\":[\"Nvidia Quadro P400 25.21.14.1678\"],\"hostname\":\"build16-a9.labs.chromium.org\",\"ip\":\"192.168.216.26\",\"named_caches\":{\"vpython\":[[\"qp\",50935560],1549982906.0]},\"nb_files_in_temp\":2,\"pid\":7940,\"python\":{\"executable\":\"c:\\\\infra-system\\\\bin\\\\python.exe\",\"packages\":null,\"version\":\"2.7.13 (v2.7.13:a06454b1afa1, Dec 17 2016, 20:53:40) [MSC v.1500 64 bit (AMD64)]\"},\"ram\":32726,\"running_time\":21321,\"sleep_streak\":8,\"ssd\":[],\"started_ts\":1549961665,\"top_windows\":[],\"uptime\":21340,\"user\":\"chrome-bot\"}",
    "version": "8ea94136c96de7396fda8587d8e40cbc2d0c20ec01ce6b45c68d42a526d02316",
    "first_seen_ts": "2017-08-02T23:12:16.365500",
    "task_name": "Test-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-ANGLE",
    "last_seen_ts": "2019-02-12T14:54:12.335408",
    "bot_id": "running"
  },
  'quarantined': {
    "authenticated_as": "bot-with-really-long-service-account-name:running.chromium.org",
    "dimensions": [
      {
        "value": [
          "1"
        ],
        "key": "android_devices"
      },
      {
        "value": [
          "vpython"
        ],
        "key": "caches"
      },
      {
        "value": [
          "ondemand"
        ],
        "key": "cpu_governor"
      },
      {
        "value": [
          "12.5.21"
        ],
        "key": "device_gms_core_version"
      },
      {
        "value": [
          "O",
          "OPR2.170623.027"
        ],
        "key": "device_os"
      },
      {
        "value": [
          "google"
        ],
        "key": "device_os_flavor"
      },
      {
        "value": [
          "9.2.32-xhdpi"
        ],
        "key": "device_playstore_version"
      },
      {
        "value": [
          "brcm"
        ],
        "key": "device_tree_compatible"
      },
      {
        "value": [
          "fugu"
        ],
        "key": "device_type"
      },
      {
        "value": [
          "0"
        ],
        "key": "gce"
      },
      {
        "value": [
          "quarantined"
        ],
        "key": "id"
      },
      {
        "value": [
          "0"
        ],
        "key": "inside_docker"
      },
      {
        "value": [
          "Android"
        ],
        "key": "os"
      },
      {
        "value": [
          "Skia"
        ],
        "key": "pool"
      },
      {
        "value": [
          "2.7.9"
        ],
        "key": "python"
      },
      {
        "value": [
          "4098-34330fc"
        ],
        "key": "server_version"
      },
      {
        "value": [
          "us",
          "us-skolo",
          "us-skolo-1"
        ],
        "key": "zone"
      }
    ],
    "task_id": "",
    "external_ip": "100.115.95.143",
    "is_dead": false,
    "quarantined": true,
    "deleted": false,
    "state": "{\"audio\":null,\"bot_group_cfg_version\":\"hash:0d12ff88393b4d\",\"cost_usd_hour\":0.15235460069444445,\"cpu_name\":\"BCM2709\",\"cwd\":\"/b/s\",\"devices\":{\"3BE9F057\":{\"battery\":{\"current\":null,\"health\":2,\"level\":100,\"power\":[\"AC\"],\"status\":2,\"temperature\":424,\"voltage\":0},\"build\":{\"board.platform\":\"<missing>\",\"build.fingerprint\":\"google/fugu/fugu:8.0.0/OPR2.170623.027/4397545:userdebug/dev-keys\",\"build.id\":\"OPR2.170623.027\",\"build.product\":\"fugu\",\"build.version.sdk\":\"26\",\"product.board\":\"fugu\",\"product.cpu.abi\":\"x86\",\"product.device\":\"fugu\"},\"cpu\":{\"cur\":\"1833000\",\"governor\":\"interactive\"},\"disk\":{},\"imei\":null,\"ip\":[],\"max_uid\":null,\"mem\":{},\"other_packages\":[\"com.intel.thermal\",\"android.autoinstalls.config.google.fugu\"],\"port_path\":\"1/4\",\"processes\":2,\"state\":\"still booting (sys.boot_completed)\",\"temp\":{},\"uptime\":129027.39}},\"disks\":{\"/b\":{\"free_mb\":4314.0,\"size_mb\":26746.5},\"/boot\":{\"free_mb\":40.4,\"size_mb\":59.9},\"/home/chrome-bot\":{\"free_mb\":986.2,\"size_mb\":988.9},\"/tmp\":{\"free_mb\":974.6,\"size_mb\":975.9},\"/var\":{\"free_mb\":223.6,\"size_mb\":975.9}},\"env\":{\"PATH\":\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/games:/usr/games\"},\"gpu\":null,\"host_dimensions\":{\"caches\":[\"vpython\"],\"cores\":[\"4\"],\"cpu\":[\"arm\",\"arm-32\",\"armv7l\",\"armv7l-32\",\"armv7l-32-BCM2709\"],\"cpu_governor\":[\"ondemand\"],\"device_tree_compatible\":[\"brcm\"],\"gce\":[\"0\"],\"gpu\":[\"none\"],\"id\":[\"quarantined\"],\"inside_docker\":[\"0\"],\"kvm\":[\"0\"],\"machine_type\":[\"n1-highcpu-4\"],\"os\":[\"Linux\",\"Raspbian\",\"Raspbian-8.0\"],\"python\":[\"2.7.9\"],\"ssd\":[\"1\"]},\"hostname\":\"quarantined\",\"ip\":\"192.168.1.152\",\"named_caches\":{\"vpython\":[[\"sQ\",92420605],1550019574.0]},\"nb_files_in_temp\":6,\"pid\":499,\"python\":{\"executable\":\"/usr/bin/python\",\"packages\":[\"M2Crypto==0.21.1\",\"RPi.GPIO==0.6.3\",\"argparse==1.2.1\",\"chardet==2.3.0\",\"colorama==0.3.2\",\"html5lib==0.999\",\"libusb1==1.5.0\",\"ndg-httpsclient==0.3.2\",\"pyOpenSSL==0.13.1\",\"pyasn1==0.1.7\",\"requests==2.4.3\",\"rsa==3.4.2\",\"six==1.8.0\",\"urllib3==1.9.1\",\"wheel==0.24.0\",\"wsgiref==0.1.2\"],\"version\":\"2.7.9 (default, Sep 17 2016, 20:26:04) \\n[GCC 4.9.2]\"},\"quarantined\":\"No available devices.\",\"ram\":926,\"running_time\":23283,\"sleep_streak\":63,\"ssd\":[\"mmcblk0\"],\"started_ts\":1550125333,\"temp\":{\"thermal_zone0\":47.774},\"uptime\":23954,\"user\":\"chrome-bot\"}",
    "version": "f775dd9893167e6fee31b96ef20f7218f07fa437ea9d6fc44496208784108545",
    "first_seen_ts": "2016-09-09T21:05:34.439930",
    "last_seen_ts": "2019-02-12T12:50:20.961462",
    "bot_id": "quarantined"
  },
  'dead': {
    "lease_id": "f69394d5f68b1f1e6c5f13e82ba4ccf72de7e6a0",
    "authenticated_as": "bot:running.chromium.org",
    "dimensions": [
      {
        "value": [
          "swarming_module_cache_vpython"
        ],
        "key": "caches"
      },
      {
        "value": [
          "8"
        ],
        "key": "cores"
      },
      {
        "value": [
          "x86",
          "x86-64",
          "x86-64-Broadwell_GCE",
          "x86-64-avx2"
        ],
        "key": "cpu"
      },
      {
        "value": [
          "1"
        ],
        "key": "gce"
      },
      {
        "value": [
          "none"
        ],
        "key": "gpu"
      },
      {
        "value": [
          "dead"
        ],
        "key": "id"
      },
      {
        "value": [
          "chrome-trusty-18091700-38cc06ee3ee"
        ],
        "key": "image"
      },
      {
        "value": [
          "0"
        ],
        "key": "inside_docker"
      },
      {
        "value": [
          "1"
        ],
        "key": "kvm"
      },
      {
        "value": [
          "n1-standard-8"
        ],
        "key": "machine_type"
      },
      {
        "value": [
          "Linux",
          "Ubuntu",
          "Ubuntu-14.04"
        ],
        "key": "os"
      },
      {
        "value": [
          "Chrome"
        ],
        "key": "pool"
      },
      {
        "value": [
          "2.7.6"
        ],
        "key": "python"
      },
      {
        "value": [
          "3986-3c043d8"
        ],
        "key": "server_version"
      },
      {
        "value": [
          "us",
          "us-east",
          "us-east1",
          "us-east1-b"
        ],
        "key": "zone"
      }
    ],
    "task_id": "",
    "external_ip": "35.229.11.33",
    "machine_lease": "gce-trusty-176",
    "is_dead": true,
    "deleted": false,
    "quarantined": false,
    "lease_expiration_ts": "2019-01-15T00:39:04",
    "state": "{\"audio\":[],\"bot_group_cfg_version\":\"hash:5bbd7d8f05c65e\",\"cost_usd_hour\":0.41316150716145833,\"cpu_name\":\"Intel(R) Xeon(R) CPU Broadwell GCE\",\"cwd\":\"/b/s\",\"disks\":{\"/\":{\"free_mb\":246117.8,\"size_mb\":302347.0}},\"env\":{\"PATH\":\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin\"},\"files\":{\"/usr/share/fonts/truetype/\":[\"Gubbi\",\"Navilu\",\"dejavu\",\"fonts-japanese-gothic.ttf\",\"fonts-japanese-mincho.ttf\",\"kochi\",\"liberation\",\"msttcorefonts\",\"pagul\",\"tlwg\",\"ttf-bengali-fonts\",\"ttf-dejavu\",\"ttf-devanagari-fonts\",\"ttf-gujarati-fonts\",\"ttf-indic-fonts-core\",\"ttf-kannada-fonts\",\"ttf-malayalam-fonts\",\"ttf-oriya-fonts\",\"ttf-punjabi-fonts\",\"ttf-tamil-fonts\",\"ttf-telugu-fonts\"]},\"gpu\":[],\"hostname\":\"dead.us-east1-b.c.chromecompute.google.com.internal\",\"ip\":\"10.0.8.219\",\"named_caches\":{\"swarming_module_cache_vpython\":[[\"kL\",887763447],1547511540.0]},\"nb_files_in_temp\":8,\"pid\":1117,\"python\":{\"executable\":\"/usr/bin/python\",\"packages\":[\"Cheetah==2.4.4\",\"CherryPy==3.2.2\",\"Landscape-Client==14.12\",\"PAM==0.4.2\",\"PyYAML==3.10\",\"Routes==2.0\",\"Twisted-Core==13.2.0\",\"Twisted-Names==13.2.0\",\"Twisted-Web==13.2.0\",\"WebOb==1.3.1\",\"apt-xapian-index==0.45\",\"argparse==1.2.1\",\"boto==2.20.1\",\"chardet==2.0.1\",\"cloud-init==0.7.5\",\"colorama==0.2.5\",\"configobj==4.7.2\",\"coverage==3.7.1\",\"crcmod==1.7\",\"google-compute-engine==2.2.4\",\"html5lib==0.999\",\"iotop==0.6\",\"jsonpatch==1.3\",\"jsonpointer==1.0\",\"numpy==1.8.2\",\"oauth==1.0.1\",\"pexpect==3.1\",\"prettytable==0.7.2\",\"psutil==1.2.1\",\"pyOpenSSL==0.13\",\"pycrypto==2.6.1\",\"pycurl==7.19.3\",\"pyserial==2.6\",\"python-apt==0.9.3.5ubuntu2\",\"python-debian==0.1.21-nmu2ubuntu2\",\"pyxdg==0.25\",\"repoze.lru==0.6\",\"requests==2.2.1\",\"six==1.5.2\",\"ssh-import-id==3.21\",\"urllib3==1.7.1\",\"wheel==0.24.0\",\"wsgiref==0.1.2\",\"zope.interface==4.0.5\"],\"version\":\"2.7.6 (default, Nov 13 2018, 12:45:42) \\n[GCC 4.8.4]\"},\"ram\":30159,\"running_time\":7309,\"sleep_streak\":22,\"ssd\":[],\"started_ts\":1547505598,\"uptime\":7321,\"user\":\"chrome-bot\"}",
    "version": "9644ba2fcbeafe7628828602251e5405db3d79b9cd230523bdf7927e204d664e",
    "first_seen_ts": "2019-01-14T00:40:11.400947",
    "last_seen_ts": "2019-01-15T00:42:19.613017",
    "machine_type": "gce-trusty",
    "bot_id": "dead"
  },
}

// These lists are really long, and likely will not have the data modified,
// so it doesn't make much to pretty-print them.
export const tasksMap = {
  // Came from a Skia GPU bot (build16-a9)
  "SkiaGPU": [{"created_ts": "2019-02-12T15:30:35.877627", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "task_id": "43004cb4fca98111", "internal_failure": false, "server_versions": ["4085-c81638b"], "state": "RUNNING", "failure": false, "modified_ts": "2019-02-12T15:38:08.596496", "started_ts": "2019-02-12T15:30:48.002729", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-ANGLE"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9617", "total_bytes_items_hot": "33290179", "initial_size": "53680871378", "total_bytes_items_cold": "35770139", "num_items_hot": "802", "num_items_cold": "7", "duration": 1.8910000324249268}, "isolated_upload": {"total_bytes_items_hot": "0", "total_bytes_items_cold": "376860", "num_items_hot": "1", "num_items_cold": "3", "duration": 0.7660000324249268}, "bot_overhead": 11.122000217437744}, "duration": 220.84799981117249, "completed_ts": "2019-02-12T15:29:55.164571", "started_ts": "2019-02-12T15:26:00.163259", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T15:29:55.164571", "created_ts": "2019-02-12T15:25:50.970002", "name": "Test-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan", "task_id": "4300485c081fdf11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9623", "total_bytes_items_hot": "27001283", "initial_size": "53685046412", "total_bytes_items_cold": "61701458", "num_items_hot": "801", "num_items_cold": "10", "duration": 2.062999963760376}, "isolated_upload": {"num_items_cold": "2", "duration": 0.9559998512268066, "total_bytes_items_cold": "8989"}, "bot_overhead": 11.557999849319458}, "duration": 997.4630000591278, "completed_ts": "2019-02-12T15:25:58.129464", "started_ts": "2019-02-12T15:09:06.290899", "internal_failure": false, "exit_code": "117", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": true, "modified_ts": "2019-02-12T15:25:58.129464", "created_ts": "2019-02-12T15:08:56.355535", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-ANGLE", "task_id": "430038e0c10e3511"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9638", "total_bytes_items_hot": "27000735", "initial_size": "53686307648", "total_bytes_items_cold": "110246182", "num_items_hot": "800", "num_items_cold": "8", "duration": 2.3339998722076416}, "isolated_upload": {"num_items_cold": "2", "duration": 0.8899998664855957, "total_bytes_items_cold": "8969"}, "bot_overhead": 11.775000095367432}, "duration": 707.4429998397827, "completed_ts": "2019-02-12T15:08:13.877435", "started_ts": "2019-02-12T14:56:11.925137", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T15:08:13.877435", "created_ts": "2019-02-12T14:55:38.788960", "name": "Test-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Debug-All-MSRTC_Vulkan", "task_id": "43002cb535ff9c11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9634", "total_bytes_items_hot": "27001283", "initial_size": "53680338837", "total_bytes_items_cold": "41368752", "num_items_hot": "801", "num_items_cold": "8", "duration": 1.937999963760376}, "isolated_upload": {"total_bytes_items_hot": "0", "total_bytes_items_cold": "1228920", "num_items_hot": "1", "num_items_cold": "3", "duration": 0.8289999961853027}, "bot_overhead": 11.430000066757202}, "duration": 200.5699999332428, "completed_ts": "2019-02-12T14:54:25.461265", "started_ts": "2019-02-12T14:50:50.645785", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T14:54:25.461265", "created_ts": "2019-02-12T14:50:31.584453", "name": "Test-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-ReleaseAndAbandonGpuContextReallyLongTaskNameLikeWowHowLongCanItBe", "task_id": "430028052d2d7511"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9629", "total_bytes_items_hot": "27001283", "initial_size": "53685657133", "total_bytes_items_cold": "46848848", "num_items_hot": "801", "num_items_cold": "10", "duration": 2.1569998264312744}, "isolated_upload": {"num_items_cold": "3", "duration": 1.3429999351501465, "total_bytes_items_cold": "6326501"}, "bot_overhead": 12.039999723434448}, "duration": 1587.574000120163, "completed_ts": "2019-02-12T13:55:35.235389", "started_ts": "2019-02-12T13:28:52.601959", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T13:55:35.235389", "created_ts": "2019-02-12T13:28:45.497111", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-ANGLE", "task_id": "42ffdd28c2e72711"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9630", "total_bytes_items_hot": "23138226", "initial_size": "53672210121", "total_bytes_items_cold": "57160886", "num_items_hot": "800", "num_items_cold": "9", "duration": 2.2190001010894775}, "isolated_upload": {"num_items_cold": "2", "duration": 0.8900001049041748, "total_bytes_items_cold": "8992"}, "bot_overhead": 11.515000104904175}, "duration": 498.52999997138977, "completed_ts": "2019-02-12T13:28:28.981371", "started_ts": "2019-02-12T13:19:56.231215", "internal_failure": false, "exit_code": "0", "state": "TIMED_OUT", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T13:28:28.981371", "created_ts": "2019-02-12T13:19:47.942258", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All", "task_id": "42ffd4f4f19a3f11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9624", "total_bytes_items_hot": "27001283", "initial_size": "53670792891", "total_bytes_items_cold": "61697362", "num_items_hot": "801", "num_items_cold": "10", "duration": 2.4100000858306885}, "isolated_upload": {"num_items_cold": "2", "duration": 1.0160000324249268, "total_bytes_items_cold": "8986"}, "bot_overhead": 12.148000240325928}, "duration": 1013.4649999141693, "completed_ts": "2019-02-12T13:17:00.993434", "started_ts": "2019-02-12T12:59:52.489832", "internal_failure": true, "exit_code": "0", "state": "BOT_DIED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T13:17:00.993434", "created_ts": "2019-02-12T12:57:44.010485", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-ANGLE", "task_id": "42ffc0c1539fad11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9623", "total_bytes_items_hot": "165156628", "initial_size": "53683738339", "total_bytes_items_cold": "40000650", "num_items_hot": "880", "num_items_cold": "8", "duration": 1.9230000972747803}, "isolated_upload": {"num_items_cold": "4", "duration": 0.9529998302459717, "total_bytes_items_cold": "36921"}, "bot_overhead": 8.297999858856201}, "duration": 557.0270001888275, "completed_ts": "2019-02-12T12:59:50.412738", "started_ts": "2019-02-12T12:50:22.324874", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T12:59:50.412738", "created_ts": "2019-02-12T12:50:12.157759", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan_Skpbench", "task_id": "42ffb9dc4764d111"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9619", "total_bytes_items_hot": "27001283", "initial_size": "53673870879", "total_bytes_items_cold": "41364138", "num_items_hot": "801", "num_items_cold": "8", "duration": 2.171999931335449}, "isolated_upload": {"total_bytes_items_hot": "0", "total_bytes_items_cold": "1228923", "num_items_hot": "1", "num_items_cold": "3", "duration": 1.0160000324249268}, "bot_overhead": 11.491000175476074}, "duration": 239.79499983787537, "completed_ts": "2019-02-12T12:49:49.267475", "started_ts": "2019-02-12T12:45:35.148432", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T12:49:49.267475", "created_ts": "2019-02-12T12:45:29.680463", "name": "Test-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-ReleaseAndAbandonGpuContext", "task_id": "42ffb58cd869f711"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9617", "total_bytes_items_hot": "23138226", "initial_size": "53686677193", "total_bytes_items_cold": "45919036", "num_items_hot": "800", "num_items_cold": "9", "duration": 2.11299991607666}, "isolated_upload": {"num_items_cold": "3", "duration": 1.2660000324249268, "total_bytes_items_cold": "1243948"}, "bot_overhead": 11.824999809265137}, "duration": 287.85000014305115, "completed_ts": "2019-02-12T12:45:00.200618", "started_ts": "2019-02-12T12:39:57.726944", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T12:45:00.200618", "created_ts": "2019-02-12T12:39:38.295857", "name": "Perf-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan", "task_id": "42ffb0304975ee11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9616", "total_bytes_items_hot": "68062081", "initial_size": "53686675205", "total_bytes_items_cold": "1988", "num_items_hot": "803", "num_items_cold": "1", "duration": 2.126000165939331}, "isolated_upload": {"num_items_cold": "3", "duration": 1.0780000686645508, "total_bytes_items_cold": "5806171"}, "bot_overhead": 11.628999710083008}, "duration": 965.1070001125336, "completed_ts": "2019-02-12T10:43:30.772993", "started_ts": "2019-02-12T10:27:11.188149", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T10:43:30.772993", "created_ts": "2019-02-12T10:27:08.285232", "name": "Perf-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Release-All", "task_id": "42ff36e1b849b611"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9616", "total_bytes_items_hot": "33162783", "initial_size": "53682983155", "total_bytes_items_cold": "35761454", "num_items_hot": "797", "num_items_cold": "7", "duration": 2.3439998626708984}, "isolated_upload": {"num_items_cold": "3", "duration": 1.0169999599456787, "total_bytes_items_cold": "1243990"}, "bot_overhead": 11.843999862670898}, "duration": 282.4739999771118, "completed_ts": "2019-02-12T10:27:03.724610", "started_ts": "2019-02-12T10:22:06.698642", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T10:27:03.724610", "created_ts": "2019-02-12T10:22:04.365694", "name": "Perf-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan", "task_id": "42ff323e65b21211"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9608", "total_bytes_items_hot": "33162783", "initial_size": "53671603618", "total_bytes_items_cold": "40559441", "num_items_hot": "797", "num_items_cold": "9", "duration": 2.2200000286102295}, "isolated_upload": {"num_items_cold": "3", "duration": 1.4540002346038818, "total_bytes_items_cold": "6326365"}, "bot_overhead": 12.007999897003174}, "duration": 1579.9739999771118, "completed_ts": "2019-02-12T10:21:42.679129", "started_ts": "2019-02-12T09:55:07.764185", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T10:21:42.679129", "created_ts": "2019-02-12T09:54:59.494006", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-ANGLE", "task_id": "42ff19732eee9b11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9603", "total_bytes_items_hot": "33162783", "initial_size": "53674047110", "total_bytes_items_cold": "35761436", "num_items_hot": "797", "num_items_cold": "7", "duration": 2.1089999675750732}, "isolated_upload": {"num_items_cold": "3", "duration": 1.2029998302459717, "total_bytes_items_cold": "1243965"}, "bot_overhead": 12.569999933242798}, "duration": 281.83899998664856, "completed_ts": "2019-02-12T09:54:32.547245", "started_ts": "2019-02-12T09:49:35.161567", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T09:54:32.547245", "created_ts": "2019-02-12T09:49:30.439016", "name": "Perf-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan", "task_id": "42ff146dea117011"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9605", "total_bytes_items_hot": "201074773", "initial_size": "53683978775", "total_bytes_items_cold": "4012442", "num_items_hot": "881", "num_items_cold": "2", "duration": 1.9240000247955322}, "isolated_upload": {"num_items_cold": "4", "duration": 1.194000005722046, "total_bytes_items_cold": "36946"}, "bot_overhead": 8.634000062942505}, "duration": 573.4800000190735, "completed_ts": "2019-02-12T09:49:10.858093", "started_ts": "2019-02-12T09:39:26.080905", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T09:49:10.858093", "created_ts": "2019-02-12T09:39:16.186182", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan_Skpbench", "task_id": "42ff0b0e79c7e011"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9605", "total_bytes_items_hot": "169111920", "initial_size": "53682313025", "total_bytes_items_cold": "35975295", "num_items_hot": "876", "num_items_cold": "7", "duration": 2.375}, "isolated_upload": {"num_items_cold": "4", "duration": 1.1570000648498535, "total_bytes_items_cold": "36951"}, "bot_overhead": 9.185999870300293}, "duration": 639.8840000629425, "completed_ts": "2019-02-12T09:38:54.897999", "started_ts": "2019-02-12T09:28:03.482624", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T09:38:54.897999", "created_ts": "2019-02-12T09:27:29.620566", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan_Skpbench_DDLRecord_9x9", "task_id": "42ff00468bda1111"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9606", "total_bytes_items_hot": "33162783", "initial_size": "53683301666", "total_bytes_items_cold": "35965608", "num_items_hot": "797", "num_items_cold": "7", "duration": 2.2350001335144043}, "isolated_upload": {"num_items_cold": "3", "duration": 1.1400001049041748, "total_bytes_items_cold": "1244282"}, "bot_overhead": 11.920000076293945}, "duration": 282.98999977111816, "completed_ts": "2019-02-12T09:25:42.269087", "started_ts": "2019-02-12T09:20:44.597603", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T09:25:42.269087", "created_ts": "2019-02-12T09:19:39.280890", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan", "task_id": "42fef9191a7f8a11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9602", "total_bytes_items_hot": "33162783", "initial_size": "53683742198", "total_bytes_items_cold": "34892588", "num_items_hot": "797", "num_items_cold": "7", "duration": 2.625}, "isolated_upload": {"num_items_cold": "3", "duration": 1.9070000648498535, "total_bytes_items_cold": "5805528"}, "bot_overhead": 12.948999881744385}, "duration": 978.2510001659393, "completed_ts": "2019-02-12T09:20:42.579770", "started_ts": "2019-02-12T09:04:08.358740", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T09:20:42.579770", "created_ts": "2019-02-12T09:03:59.664762", "name": "Perf-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Release-All", "task_id": "42feeac2c987ea11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9605", "total_bytes_items_hot": "33162783", "initial_size": "53683938954", "total_bytes_items_cold": "47939246", "num_items_hot": "797", "num_items_cold": "7", "duration": 2.562999963760376}, "isolated_upload": {"num_items_cold": "2", "duration": 0.6410000324249268, "total_bytes_items_cold": "8983"}, "bot_overhead": 11.83299994468689}, "duration": 177.01900005340576, "completed_ts": "2019-02-12T09:03:33.049419", "started_ts": "2019-02-12T09:00:21.171165", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T09:03:33.049419", "created_ts": "2019-02-12T09:00:00.462518", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-Vulkan", "task_id": "42fee71c62c98711"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9615", "total_bytes_items_hot": "33162235", "initial_size": "53685362684", "total_bytes_items_cold": "85782826", "num_items_hot": "796", "num_items_cold": "7", "duration": 2.2660000324249268}, "isolated_upload": {"num_items_cold": "2", "duration": 0.9529998302459717, "total_bytes_items_cold": "8989"}, "bot_overhead": 11.559000015258789}, "duration": 343.0149998664856, "completed_ts": "2019-02-12T08:59:02.846407", "started_ts": "2019-02-12T08:53:04.476996", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T08:59:02.846407", "created_ts": "2019-02-12T08:52:59.961787", "name": "Perf-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Debug-All-Vulkan", "task_id": "42fee0b1c5fb6511"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9624", "total_bytes_items_hot": "33162235", "initial_size": "53685475533", "total_bytes_items_cold": "85800743", "num_items_hot": "796", "num_items_cold": "7", "duration": 2.2660000324249268}, "isolated_upload": {"total_bytes_items_hot": "0", "total_bytes_items_cold": "337603", "num_items_hot": "1", "num_items_cold": "3", "duration": 0.9530000686645508}, "bot_overhead": 11.81499981880188}, "duration": 499.9810001850128, "completed_ts": "2019-02-12T08:52:27.789685", "started_ts": "2019-02-12T08:43:53.091785", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T08:52:27.789685", "created_ts": "2019-02-12T08:43:47.890509", "name": "Test-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Debug-All-Vulkan", "task_id": "42fed84542ceed11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9624", "total_bytes_items_hot": "165099888", "initial_size": "53668282075", "total_bytes_items_cold": "39976062", "num_items_hot": "875", "num_items_cold": "8", "duration": 1.9060001373291016}, "isolated_upload": {"num_items_cold": "4", "duration": 0.9530000686645508, "total_bytes_items_cold": "36930"}, "bot_overhead": 8.210000038146973}, "duration": 551.2039999961853, "completed_ts": "2019-02-12T08:43:51.154274", "started_ts": "2019-02-12T08:34:28.867532", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T08:43:51.154274", "created_ts": "2019-02-12T08:34:18.822554", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan_Skpbench", "task_id": "42fecf965ddcb911"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9619", "total_bytes_items_hot": "33162783", "initial_size": "53670602818", "total_bytes_items_cold": "47939225", "num_items_hot": "797", "num_items_cold": "7", "duration": 1.9689998626708984}, "isolated_upload": {"num_items_cold": "2", "duration": 0.7030000686645508, "total_bytes_items_cold": "8992"}, "bot_overhead": 11.733000040054321}, "duration": 177.55999994277954, "completed_ts": "2019-02-12T08:33:38.298168", "started_ts": "2019-02-12T08:30:26.184015", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T08:33:38.298168", "created_ts": "2019-02-12T08:30:23.469833", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-Vulkan", "task_id": "42fecbff069f5711"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9616", "total_bytes_items_hot": "33162783", "initial_size": "53654665601", "total_bytes_items_cold": "46996649", "num_items_hot": "797", "num_items_cold": "7", "duration": 2.882000207901001}, "isolated_upload": {"num_items_cold": "2", "duration": 0.9530000686645508, "total_bytes_items_cold": "8980"}, "bot_overhead": 12.17799997329712}, "duration": 483.00099992752075, "completed_ts": "2019-02-12T08:30:02.405816", "started_ts": "2019-02-12T08:21:44.503602", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T08:30:02.405816", "created_ts": "2019-02-12T08:21:36.455063", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All", "task_id": "42fec3f4543a6d11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9609", "total_bytes_items_hot": "169111920", "initial_size": "53618701567", "total_bytes_items_cold": "35964034", "num_items_hot": "876", "num_items_cold": "7", "duration": 2.250999927520752}, "isolated_upload": {"num_items_cold": "4", "duration": 1.1399998664855957, "total_bytes_items_cold": "36954"}, "bot_overhead": 9.472999811172485}, "duration": 593.7170000076294, "completed_ts": "2019-02-12T08:21:21.327186", "started_ts": "2019-02-12T08:11:15.380896", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T08:21:21.327186", "created_ts": "2019-02-12T08:11:06.567674", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan_Skpbench_DDLRecord_9x9", "task_id": "42feba57df5a4a11"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9602", "total_bytes_items_hot": "33162235", "initial_size": "53532918748", "total_bytes_items_cold": "85782819", "num_items_hot": "796", "num_items_cold": "7", "duration": 2.1570000648498535}, "isolated_upload": {"num_items_cold": "2", "duration": 1.0780000686645508, "total_bytes_items_cold": "8980"}, "bot_overhead": 11.759000062942505}, "duration": 346.72000002861023, "completed_ts": "2019-02-12T08:10:22.463052", "started_ts": "2019-02-12T08:04:21.233981", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T08:10:22.463052", "created_ts": "2019-02-12T08:04:14.780973", "name": "Perf-Win10-MSVC-Golo-GPU-QuadroP400-x86_64-Debug-All-Vulkan", "task_id": "42feb40f62b87311"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9595", "total_bytes_items_hot": "33162783", "initial_size": "53484976433", "total_bytes_items_cold": "47942315", "num_items_hot": "797", "num_items_cold": "7", "duration": 2.4689998626708984}, "isolated_upload": {"num_items_cold": "2", "duration": 0.5939998626708984, "total_bytes_items_cold": "8983"}, "bot_overhead": 11.586000204086304}, "duration": 182.74399995803833, "completed_ts": "2019-02-12T08:03:34.435319", "started_ts": "2019-02-12T08:00:17.223617", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T08:03:34.435319", "created_ts": "2019-02-12T08:00:07.527870", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-Vulkan", "task_id": "42feb04983e8a011"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9588", "total_bytes_items_hot": "33162783", "initial_size": "53437976716", "total_bytes_items_cold": "46999717", "num_items_hot": "797", "num_items_cold": "7", "duration": 3.375}, "isolated_upload": {"num_items_cold": "2", "duration": 1.3910000324249268, "total_bytes_items_cold": "8977"}, "bot_overhead": 13.366999864578247}, "duration": 484.3910000324249, "completed_ts": "2019-02-12T07:59:42.483992", "started_ts": "2019-02-12T07:51:21.912338", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T07:59:42.483992", "created_ts": "2019-02-12T07:51:17.482688", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All", "task_id": "42fea832f46d6211"}, {"server_versions": ["4085-c81638b"], "performance_stats": {"isolated_download": {"initial_number_items": "9580", "total_bytes_items_hot": "165125179", "initial_size": "53397980683", "total_bytes_items_cold": "39996033", "num_items_hot": "875", "num_items_cold": "8", "duration": 1.9530000686645508}, "isolated_upload": {"num_items_cold": "4", "duration": 1.0149998664855957, "total_bytes_items_cold": "36811"}, "bot_overhead": 8.788000106811523}, "duration": 3228.0910000801086, "completed_ts": "2019-02-12T07:50:51.495160", "started_ts": "2019-02-12T06:56:51.851579", "internal_failure": false, "exit_code": "0", "state": "COMPLETED", "bot_version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "failure": false, "modified_ts": "2019-02-12T07:50:51.495160", "created_ts": "2019-02-12T06:56:50.507385", "name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan_Skpbench_DDLTotal_9x9", "task_id": "42fe76597cf16f11"}],
};

export const eventsMap = {
  // Came from a Skia GPU bot (build16-a9)
  "SkiaGPU": [{"task_id": "4300ceb85b93e011", "ts": "2019-02-12T17:52:40.070982", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "request_task"}, {"task_id": "4300c3b09d7d1911", "ts": "2019-02-12T17:50:43.086788", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "task_completed"}, {"task_id": "4300c3b09d7d1911", "ts": "2019-02-12T17:40:54.772160", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "request_task"}, {"task_id": "4300adb2ad3af711", "ts": "2019-02-12T17:39:36.083344", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "task_completed"}, {"task_id": "4300adb2ad3af711", "ts": "2019-02-12T17:16:37.620175", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "request_task"}, {"task_id": "4300a27d716aee11", "ts": "2019-02-12T17:15:44.896506", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "task_completed"}, {"task_id": "4300a27d716aee11", "ts": "2019-02-12T17:04:45.428521", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "request_task"}, {"task_id": "430099cf48da0d11", "ts": "2019-02-12T16:59:47.073509", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "task_completed"}, {"task_id": "430099cf48da0d11", "ts": "2019-02-12T16:54:51.032905", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "request_task"}, {"task_id": "4300812ca4099a11", "ts": "2019-02-12T16:54:44.284514", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "task_completed"}, {"task_id": "4300812ca4099a11", "ts": "2019-02-12T16:28:00.471552", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "request_task"}, {"task_id": "43007cde1979ed11", "ts": "2019-02-12T16:27:07.693668", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "task_completed"}, {"task_id": "43007cde1979ed11", "ts": "2019-02-12T16:23:34.564148", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "request_task"}, {"ts": "2019-02-12T16:23:32.572685", "quarantined": false, "version": "abcdoeraymeyouandme", "event_type": "bot_connected"}, {"ts": "2019-02-12T16:23:23.213689", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "message": "About to restart: Updating to abcdoeraymeyouandme", "event_type": "bot_shutdown"}, {"ts": "2019-02-12T16:23:21.463539", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_update"}, {"task_id": "43006da394d9cf11", "ts": "2019-02-12T16:23:19.251628", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "43006da394d9cf11", "ts": "2019-02-12T16:06:56.101600", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "4300619aa8a7ac11", "ts": "2019-02-12T16:05:37.140792", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "4300619aa8a7ac11", "ts": "2019-02-12T15:53:32.837659", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "43005d3ac6777311", "ts": "2019-02-12T15:52:40.259648", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "43005d3ac6777311", "ts": "2019-02-12T15:48:51.611798", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "43004cb4fca98111", "ts": "2019-02-12T15:47:58.755976", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "43004cb4fca98111", "ts": "2019-02-12T15:30:48.387169", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "4300485c081fdf11", "ts": "2019-02-12T15:29:55.436456", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "4300485c081fdf11", "ts": "2019-02-12T15:26:00.676840", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "430038e0c10e3511", "ts": "2019-02-12T15:25:58.443164", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "430038e0c10e3511", "ts": "2019-02-12T15:09:06.646355", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "43002cb535ff9c11", "ts": "2019-02-12T15:08:14.291897", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "43002cb535ff9c11", "ts": "2019-02-12T14:56:12.164330", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "430028052d2d7511", "ts": "2019-02-12T14:54:25.781920", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "430028052d2d7511", "ts": "2019-02-12T14:50:50.858780", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ffdd28c2e72711", "ts": "2019-02-12T13:55:36.014396", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ffdd28c2e72711", "ts": "2019-02-12T13:28:52.998476", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ffd4f4f19a3f11", "ts": "2019-02-12T13:28:29.451825", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ffd4f4f19a3f11", "ts": "2019-02-12T13:19:56.420584", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ffc0c1539fad11", "ts": "2019-02-12T13:17:01.570025", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ffc0c1539fad11", "ts": "2019-02-12T12:59:52.823350", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ffb9dc4764d111", "ts": "2019-02-12T12:59:50.681929", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ffb9dc4764d111", "ts": "2019-02-12T12:50:22.571345", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ffb58cd869f711", "ts": "2019-02-12T12:49:49.591261", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ffb58cd869f711", "ts": "2019-02-12T12:45:35.487901", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ffb0304975ee11", "ts": "2019-02-12T12:45:00.637659", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ffb0304975ee11", "ts": "2019-02-12T12:39:57.993849", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ff36e1b849b611", "ts": "2019-02-12T10:43:31.172260", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ff36e1b849b611", "ts": "2019-02-12T10:27:11.531507", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ff323e65b21211", "ts": "2019-02-12T10:27:04.375023", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ff323e65b21211", "ts": "2019-02-12T10:22:06.905287", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}, {"task_id": "42ff19732eee9b11", "ts": "2019-02-12T10:21:43.105154", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "task_completed"}, {"task_id": "42ff19732eee9b11", "ts": "2019-02-12T09:55:08.127643", "quarantined": false, "version": "6fda8587d8e40cbc2d0c208ea94136c96de739ec01ce6b45c68d42a526d02316", "event_type": "request_task"}],
}
