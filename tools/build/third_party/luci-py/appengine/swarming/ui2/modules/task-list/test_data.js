// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This file has some data used for task-list tests.

// 2 tasks of each of the various states.
export const tasks_20 = {
  "items": [
    {
      "created_ts": "2018-12-19T16:31:28.290449",
      "name": "webview_instrumentation_test_apk on Android device Nexus 5X/Android/4e644cb4b4/android-marshmallow-arm64-rel/10996:3:7",
      "task_id": "41e020504d0a5110",
      "tags": [
        "build_is_experimental:false",
        "buildername:android-marshmallow-arm64-rel",
        "buildnumber:10996",
        "data:4e644cb4b4548ca73ba5aaad78fb020b631498cf",
        "device_os:MMB29Q",
        "device_type:bullhead",
        "master:chromium.android",
        "name:webview_instrumentation_test_apk",
        "os:Android",
        "pool:Chrome",
        "priority:25",
        "project:chromium",
        "purpose:CI",
        "purpose:luci",
        "purpose:post-commit",
        "service_account:none",
        "slavename:swarm1596-c4",
        "spec_name:chromium.ci:android-marshmallow-arm64-rel",
        "stepname:webview_instrumentation_test_apk on Android device Nexus 5X",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "internal_failure": false,
      "server_versions": [
        "3945-402d3be"
      ],
      "failure": false,
      "state": "PENDING",
      "modified_ts": "2018-12-19T16:31:28.333370",
      "user": "",
      "current_task_slice": "0"
    },
    {
      "created_ts": "2018-12-19T16:31:28.036976",
      "name": "ios_net_unittests (iPhone X iOS 12.1)/Mac/1f1610532e/ios12-beta-simulator/1792",
      "task_id": "41e0204f39d06210",
      "tags": [
        "build_is_experimental:false",
        "buildername:ios12-beta-simulator",
        "buildnumber:1792",
        "data:1f1610532e5a4e3e1dafdc320b727c0ce1f43938",
        "device_type:iPhone X",
        "ios_version:12.1",
        "master:chromium.fyi",
        "name:ios_net_unittests",
        "os:Mac-10.13.6",
        "platform:simulator",
        "pool:Chrome",
        "priority:200",
        "service_account:ios-isolated-tester@chops-service-accounts.iam.gserviceaccount.com",
        "slavename:build228-m9",
        "spec_name:chromium.ci:ios12-beta-simulator",
        "spec_name:chromium.fyi:ios12-beta-simulator:ios_net_unittests:simulator:iPhone X:12.1:10o45e",
        "stepname:ios_net_unittests (iPhone X iOS 12.1)",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "test:ios_net_unittests",
        "user:None"
      ],
      "internal_failure": false,
      "server_versions": [
        "3945-402d3be"
      ],
      "failure": false,
      "state": "PENDING",
      "modified_ts": "2018-12-19T16:31:28.056791",
      "user": "",
      "current_task_slice": "0"
    },
    {
      "created_ts": "2018-12-19T16:33:27.621328",
      "bot_dimensions": [
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
            "x86-64-i7-7700"
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
            "8086",
            "8086:5912",
            "8086:5912-24.20.100.6286"
          ],
          "key": "gpu"
        },
        {
          "value": [
            "build128-a9"
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
            "n1-highcpu-8"
          ],
          "key": "machine_type"
        },
        {
          "value": [
            "Windows",
            "Windows-10",
            "Windows-10-17134.345"
          ],
          "key": "os"
        },
        {
          "value": [
            "Chrome-GPU"
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
            "3945-402d3be"
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
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "task_id": "41e0222290be8110",
      "run_id": "41e0222290be8111",
      "internal_failure": false,
      "tags": [
        "build_is_experimental:false",
        "buildername:win_optional_gpu_tests_rel",
        "buildnumber:12508",
        "cpu:x86-64",
        "data:44c4b111a78203e34fbbb84f1fe748f8f9d56fc3",
        "gerrit:https://chromium-review.googlesource.com/c/1384644/1",
        "gpu:8086:5912-24.20.100.6286",
        "master:tryserver.chromium.win",
        "name:info_collection_tests",
        "os:Windows-10",
        "patch_project:chromium/src",
        "pool:Chrome-GPU",
        "priority:30",
        "project:chromium",
        "purpose:ManualTS",
        "purpose:luci",
        "purpose:pre-commit",
        "service_account:none",
        "slavename:swarm522-c4",
        "spec_name:chromium.try:win_optional_gpu_tests_rel",
        "stepname:info_collection_tests on Intel GPU on Windows (with patch) on Windows-10",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "server_versions": [
        "3945-402d3be"
      ],
      "costs_usd": [
        0
      ],
      "name": "info_collection_tests on Intel GPU on Windows (with patch)/Windows-10/44c4b111a7/win_optional_gpu_tests_rel/12508",
      "failure": false,
      "state": "RUNNING",
      "modified_ts": "2018-12-19T16:33:28.255378",
      "user": "",
      "bot_id": "build128-a9",
      "current_task_slice": "0",
      "try_number": "1",
      "started_ts": "2018-12-19T16:33:28.255378"
    },
    {
      "created_ts": "2018-12-19T16:33:26.901933",
      "bot_dimensions": [
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
            "gce-trusty-e833d7b0-us-east1-b-fb3g"
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
            "3945-402d3be"
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
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "task_id": "41e0222076a33010",
      "run_id": "41e0222076a33011",
      "internal_failure": false,
      "tags": [
        "build_is_experimental:false",
        "buildername:linux_chromium_tsan_rel_ng",
        "buildnumber:169264",
        "cpu:x86-64",
        "data:6bd49d4dc3c5b5d12c3413856ce06250da9799eb",
        "gerrit:https://chromium-review.googlesource.com/c/1377132/2",
        "gpu:none",
        "master:tryserver.chromium.linux",
        "name:extensions_browsertests",
        "os:Ubuntu-14.04",
        "patch_project:chromium/src",
        "pool:Chrome",
        "priority:30",
        "project:chromium",
        "purpose:ManualTS",
        "purpose:luci",
        "purpose:pre-commit",
        "service_account:none",
        "slavename:swarm146-c4",
        "spec_name:chromium.try:linux_chromium_tsan_rel_ng",
        "stepname:extensions_browsertests (with patch)",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "server_versions": [
        "3945-402d3be"
      ],
      "costs_usd": [
        0
      ],
      "name": "extensions_browsertests (with patch)/Ubuntu-14.04/6bd49d4dc3/linux_chromium_tsan_rel_ng/169264",
      "failure": false,
      "state": "RUNNING",
      "modified_ts": "2018-12-19T16:33:27.622954",
      "user": "",
      "bot_id": "gce-trusty-e833d7b0-us-east1-b-fb3g",
      "current_task_slice": "0",
      "try_number": "1",
      "started_ts": "2018-12-19T16:33:27.622954"
    },
    {
      "created_ts": "2018-12-19T16:24:24.461707",
      "name": "Khadas Vim2 Max",
      "task_id": "41e019d8b7aa2f10",
      "tags": [
        "device_type:Khadas Vim2 Max",
        "pool:fuchsia.tests",
        "priority:200",
        "service_account:none",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "internal_failure": false,
      "server_versions": [
        "3945-402d3be"
      ],
      "abandoned_ts": "2018-12-19T16:30:01.437639",
      "failure": false,
      "state": "EXPIRED",
      "modified_ts": "2018-12-19T16:30:01.437639",
      "user": "",
      "current_task_slice": "0"
    },
    {
      "created_ts": "2018-12-19T16:20:01.114882",
      "name": "all tests",
      "task_id": "41e015d550464910",
      "tags": [
        "device_type:Khadas Vim2 Max",
        "pool:fuchsia.tests",
        "priority:200",
        "service_account:none",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "internal_failure": false,
      "server_versions": [
        "3945-402d3be"
      ],
      "abandoned_ts": "2018-12-19T16:25:07.012662",
      "failure": false,
      "state": "EXPIRED",
      "modified_ts": "2018-12-19T16:25:07.012662",
      "user": "",
      "current_task_slice": "0"
    },
    {
      "cipd_pins": {
        "packages": [
          {
            "path": ".swarming_module",
            "version": "46c0c897ca0f053799ee41fd148bb7a47232df47",
            "package_name": "infra/python/cpython/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "2737ea8ed9b958f4d5aa9ffe106115a649ada241",
            "package_name": "infra/tools/luci/logdog/butler/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "OAXVAmcUSrvDygYUrCDzv20LRono9938YOHPu0zKowgC",
            "package_name": "infra/tools/luci/vpython-native/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "ucaOciwAE9aweCUDOrmSvyiwrjmbywuB0NzAGUXIHjAC",
            "package_name": "infra/tools/luci/vpython/linux-amd64"
          },
          {
            "path": "bin",
            "version": "a57ad614c01fec9fa4259473c8ea3fd992d7c349",
            "package_name": "infra/tools/luci/logdog/butler/linux-amd64"
          }
        ],
        "client_package": {
          "version": "a2dqpK39PjGpFdcdw62OAE0JOJJ9n8J_AXpJHmH0QCIC",
          "package_name": "infra/tools/cipd/linux-amd64"
        }
      },
      "run_id": "41e023035ecced11",
      "outputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "6a872951f78a349bd9b96b01fc547628305af99b"
      },
      "server_versions": [
        "3945-402d3be"
      ],
      "duration": 121.06866407394409,
      "completed_ts": "2018-12-19T16:36:49.812379",
      "started_ts": "2018-12-19T16:34:33.445109",
      "internal_failure": false,
      "exit_code": "1",
      "state": "TIMED_OUT",
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "tags": [
        "build_is_experimental:false",
        "buildername:Lollipop Tablet Tester",
        "buildnumber:2861",
        "data:c56d2fa421fcf793003055f44cc341200f3c9e46",
        "device_os:LMY49B",
        "device_type:flo",
        "master:chromium.android",
        "name:blink_heap_unittests",
        "os:Android",
        "pool:Chrome",
        "priority:25",
        "project:chromium",
        "purpose:CI",
        "purpose:luci",
        "purpose:post-commit",
        "service_account:none",
        "slavename:swarm99-c4",
        "spec_name:chromium.ci:Lollipop Tablet Tester",
        "stepname:blink_heap_unittests on Android device Nexus 7 [2013]",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "failure": true,
      "modified_ts": "2018-12-19T16:36:49.812379",
      "user": "",
      "created_ts": "2018-12-19T16:34:25.232128",
      "name": "blink_heap_unittests on Android device Nexus 7 [2013]/Android/c56d2fa421/Lollipop Tablet Tester/2861",
      "task_id": "41e023035ecced10",
      "bot_dimensions": [
        {
          "value": [
            "1"
          ],
          "key": "android_devices"
        },
        {
          "value": [
            "swarming_module_cache_vpython"
          ],
          "key": "caches"
        },
        {
          "value": [
            "6.7.79"
          ],
          "key": "device_gms_core_version"
        },
        {
          "value": [
            "L",
            "LMY49B"
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
            "flo"
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
            "build40-b1--device3"
          ],
          "key": "id"
        },
        {
          "value": [
            "1",
            "stock"
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
            "Chrome"
          ],
          "key": "pool"
        },
        {
          "value": [
            "2.7.12"
          ],
          "key": "python"
        },
        {
          "value": [
            "3945-402d3be"
          ],
          "key": "server_version"
        },
        {
          "value": [
            "<30"
          ],
          "key": "temp_band"
        },
        {
          "value": [
            "us",
            "us-mtv",
            "us-mtv-chops",
            "us-mtv-chops-b",
            "us-mtv-chops-b-1"
          ],
          "key": "zone"
        }
      ],
      "try_number": "1",
      "current_task_slice": "0",
      "costs_usd": [
        0.015322936171758469
      ],
      "bot_id": "build40-b1--device3"
    },
    {
      "cipd_pins": {
        "packages": [
          {
            "path": ".swarming_module",
            "version": "46c0c897ca0f053799ee41fd148bb7a47232df47",
            "package_name": "infra/python/cpython/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "2737ea8ed9b958f4d5aa9ffe106115a649ada241",
            "package_name": "infra/tools/luci/logdog/butler/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "OAXVAmcUSrvDygYUrCDzv20LRono9938YOHPu0zKowgC",
            "package_name": "infra/tools/luci/vpython-native/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "ucaOciwAE9aweCUDOrmSvyiwrjmbywuB0NzAGUXIHjAC",
            "package_name": "infra/tools/luci/vpython/linux-amd64"
          },
          {
            "path": "bin",
            "version": "a57ad614c01fec9fa4259473c8ea3fd992d7c349",
            "package_name": "infra/tools/luci/logdog/butler/linux-amd64"
          }
        ],
        "client_package": {
          "version": "a2dqpK39PjGpFdcdw62OAE0JOJJ9n8J_AXpJHmH0QCIC",
          "package_name": "infra/tools/cipd/linux-amd64"
        }
      },
      "run_id": "41e01fe02b981411",
      "outputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "7b97a56916509478552ef30c6375d3564b126b96"
      },
      "server_versions": [
        "3945-402d3be"
      ],
      "duration": 121.17190790176392,
      "completed_ts": "2018-12-19T16:33:12.630766",
      "started_ts": "2018-12-19T16:31:01.956320",
      "internal_failure": false,
      "exit_code": "1",
      "state": "TIMED_OUT",
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "tags": [
        "build_is_experimental:false",
        "buildername:Lollipop Tablet Tester",
        "buildnumber:2861",
        "data:baee414ff051cfa3d6955532c4aab393dfd2edd0",
        "device_os:LMY49B",
        "device_type:flo",
        "master:chromium.android",
        "name:base_unittests",
        "os:Android",
        "pool:Chrome",
        "priority:25",
        "project:chromium",
        "purpose:CI",
        "purpose:luci",
        "purpose:post-commit",
        "service_account:none",
        "slavename:swarm99-c4",
        "spec_name:chromium.ci:Lollipop Tablet Tester",
        "stepname:base_unittests on Android device Nexus 7 [2013]",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "failure": true,
      "modified_ts": "2018-12-19T16:33:12.630766",
      "user": "",
      "created_ts": "2018-12-19T16:30:59.608430",
      "name": "base_unittests on Android device Nexus 7 [2013]/Android/baee414ff0/Lollipop Tablet Tester/2861",
      "task_id": "41e01fe02b981410",
      "bot_dimensions": [
        {
          "value": [
            "1"
          ],
          "key": "android_devices"
        },
        {
          "value": [
            "swarming_module_cache_vpython"
          ],
          "key": "caches"
        },
        {
          "value": [
            "6.7.79"
          ],
          "key": "device_gms_core_version"
        },
        {
          "value": [
            "L",
            "LMY49B"
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
            "flo"
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
            "build40-b1--device3"
          ],
          "key": "id"
        },
        {
          "value": [
            "1",
            "stock"
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
            "Chrome"
          ],
          "key": "pool"
        },
        {
          "value": [
            "2.7.12"
          ],
          "key": "python"
        },
        {
          "value": [
            "3945-402d3be"
          ],
          "key": "server_version"
        },
        {
          "value": [
            "<30"
          ],
          "key": "temp_band"
        },
        {
          "value": [
            "us",
            "us-mtv",
            "us-mtv-chops",
            "us-mtv-chops-b",
            "us-mtv-chops-b-1"
          ],
          "key": "zone"
        }
      ],
      "try_number": "1",
      "current_task_slice": "0",
      "costs_usd": [
        0.014702407470976892
      ],
      "bot_id": "build40-b1--device3"
    },
    {
      "created_ts": "2018-12-19T15:55:52.105930",
      "bot_dimensions": [
        {
          "value": [
            "vpython"
          ],
          "key": "caches"
        },
        {
          "value": [
            "4"
          ],
          "key": "cores"
        },
        {
          "value": [
            "x86",
            "x86-64",
            "x86-64-i5-6260U"
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
            "8086",
            "8086:1926",
            "8086:1926-25.20.100.6444"
          ],
          "key": "gpu"
        },
        {
          "value": [
            "skia-e-win-352"
          ],
          "key": "id"
        },
        {
          "value": [
            "medium"
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
            "n1-standard-4"
          ],
          "key": "machine_type"
        },
        {
          "value": [
            "Windows",
            "Windows-10",
            "Windows-10-17134.441"
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
            "2.7.14"
          ],
          "key": "python"
        },
        {
          "value": [
            "3945-402d3be"
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
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "task_id": "41dfffb8b1414b10",
      "run_id": "41dfffb8b1414b11",
      "internal_failure": true,
      "tags": [
        "id:skia-e-win-352",
        "pool:Skia",
        "priority:50",
        "service_account:none",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:skiabot@example.com"
      ],
      "server_versions": [
        "3945-402d3be"
      ],
      "costs_usd": [
        0.05177581497216855
      ],
      "name": "Leased by user@example.com using leasing.skia.org",
      "abandoned_ts": "2018-12-19T16:22:04.935851",
      "failure": false,
      "state": "BOT_DIED",
      "modified_ts": "2018-12-19T16:22:04.935851",
      "user": "skiabot@example.com",
      "bot_id": "skia-e-win-352",
      "current_task_slice": "0",
      "try_number": "1",
      "started_ts": "2018-12-19T16:07:20.776635"
    },
    {
      "created_ts": "2018-12-19T15:55:51.302827",
      "bot_dimensions": [
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
            "gce-trusty-e833d7b0-us-west1-c-l5rh"
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
            "3945-402d3be"
          ],
          "key": "server_version"
        },
        {
          "value": [
            "us",
            "us-west",
            "us-west1",
            "us-west1-c"
          ],
          "key": "zone"
        }
      ],
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "task_id": "41dfffb4970ae410",
      "run_id": "41dfffb4970ae411",
      "internal_failure": true,
      "tags": [
        "build_is_experimental:false",
        "buildername:linux_chromium_asan_rel_ng",
        "buildnumber:167471",
        "cpu:x86-64",
        "data:5f132ace226a59fe94ca33d3f35f1de83d027a67",
        "gerrit:https://chromium-review.googlesource.com/c/1384251/1",
        "gpu:none",
        "master:tryserver.chromium.linux",
        "name:extensions_browsertests",
        "os:Ubuntu-14.04",
        "patch_project:chromium/src",
        "pool:Chrome",
        "priority:30",
        "project:chromium",
        "purpose:ManualTS",
        "purpose:luci",
        "purpose:pre-commit",
        "service_account:none",
        "slavename:swarm2787-c4",
        "spec_name:chromium.try:linux_chromium_asan_rel_ng",
        "stepname:extensions_browsertests (with patch)",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "server_versions": [
        "3945-402d3be"
      ],
      "costs_usd": [
        0.015967942569577766
      ],
      "name": "extensions_browsertests (with patch)/Ubuntu-14.04/5f132ace22/linux_chromium_asan_rel_ng/167471",
      "abandoned_ts": "2018-12-19T15:58:11.142556",
      "failure": false,
      "state": "BOT_DIED",
      "modified_ts": "2018-12-19T15:58:11.142556",
      "user": "",
      "bot_id": "gce-trusty-e833d7b0-us-west1-c-l5rh",
      "current_task_slice": "0",
      "try_number": "1",
      "started_ts": "2018-12-19T15:55:51.961390"
    },
    {
      "cipd_pins": {
        "packages": [
          {
            "path": ".swarming_module",
            "version": "46c0c897ca0f053799ee41fd148bb7a47232df47",
            "package_name": "infra/python/cpython/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "2737ea8ed9b958f4d5aa9ffe106115a649ada241",
            "package_name": "infra/tools/luci/logdog/butler/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "OAXVAmcUSrvDygYUrCDzv20LRono9938YOHPu0zKowgC",
            "package_name": "infra/tools/luci/vpython-native/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "ucaOciwAE9aweCUDOrmSvyiwrjmbywuB0NzAGUXIHjAC",
            "package_name": "infra/tools/luci/vpython/linux-amd64"
          }
        ],
        "client_package": {
          "version": "a2dqpK39PjGpFdcdw62OAE0JOJJ9n8J_AXpJHmH0QCIC",
          "package_name": "infra/tools/cipd/linux-amd64"
        }
      },
      "run_id": "41def2cd67262111",
      "outputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "5dc1c4de48f0c991b793f322856824dbfe6c7280"
      },
      "server_versions": [
        "3945-402d3be"
      ],
      "duration": 98.734139919281,
      "completed_ts": "2018-12-19T11:03:57.376648",
      "started_ts": "2018-12-19T11:02:11.302712",
      "cost_saved_usd": 0.012174377087208232,
      "internal_failure": false,
      "exit_code": "0",
      "state": "COMPLETED",
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "tags": [
        "build_is_experimental:false",
        "buildername:chromeos-amd64-generic-rel",
        "buildnumber:155679",
        "data:2400d49f9f92b326885593db480f2a3df5862a09",
        "gerrit:https://chromium-review.googlesource.com/c/1381231/2",
        "kvm:1",
        "master:tryserver.chromium.chromiumos",
        "name:aura_unittests",
        "os:Ubuntu-14.04",
        "patch_project:chromium/src",
        "pool:Chrome-CrOS-VM",
        "priority:30",
        "project:chromium",
        "purpose:ManualTS",
        "purpose:luci",
        "purpose:pre-commit",
        "service_account:none",
        "slavename:swarm1362-c4",
        "spec_name:chromium.try:chromeos-amd64-generic-rel",
        "stepname:aura_unittests (with patch)",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "deduped_from": "41def2cd67262111",
      "failure": false,
      "modified_ts": "2018-12-19T16:40:11.504026",
      "user": "",
      "created_ts": "2018-12-19T16:40:11.462125",
      "name": "aura_unittests (with patch)/Ubuntu-14.04/2400d49f9f/chromeos-amd64-generic-rel/155679",
      "task_id": "41e0284bf01aef10",
      "bot_dimensions": [
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
            "gce-trusty-e833d7b0-us-east1-b-6z96"
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
            "Chrome-CrOS-VM"
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
            "3945-402d3be"
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
      "try_number": "0",
      "current_task_slice": "0",
      "bot_id": "gce-trusty-e833d7b0-us-east1-b-6z96"
    },
    {
      "cipd_pins": {
        "packages": [
          {
            "path": ".swarming_module",
            "version": "46c0c897ca0f053799ee41fd148bb7a47232df47",
            "package_name": "infra/python/cpython/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "2737ea8ed9b958f4d5aa9ffe106115a649ada241",
            "package_name": "infra/tools/luci/logdog/butler/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "OAXVAmcUSrvDygYUrCDzv20LRono9938YOHPu0zKowgC",
            "package_name": "infra/tools/luci/vpython-native/linux-amd64"
          },
          {
            "path": ".swarming_module",
            "version": "ucaOciwAE9aweCUDOrmSvyiwrjmbywuB0NzAGUXIHjAC",
            "package_name": "infra/tools/luci/vpython/linux-amd64"
          }
        ],
        "client_package": {
          "version": "a2dqpK39PjGpFdcdw62OAE0JOJJ9n8J_AXpJHmH0QCIC",
          "package_name": "infra/tools/cipd/linux-amd64"
        }
      },
      "run_id": "41db9ded68fb5c11",
      "outputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "de32bbbe91a211fb52550fe465ac63bc6ef314a9"
      },
      "server_versions": [
        "3945-402d3be"
      ],
      "duration": 0.6186468601226807,
      "completed_ts": "2018-12-18T19:41:02.780795",
      "started_ts": "2018-12-18T19:40:56.752498",
      "cost_saved_usd": 0.0008087886289414981,
      "internal_failure": false,
      "exit_code": "0",
      "state": "COMPLETED",
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "tags": [
        "build_is_experimental:false",
        "buildername:linux-chromeos-rel",
        "buildnumber:165248",
        "cpu:x86-64",
        "data:ff796fc6ae486982281a9f15a8ffb8c40e51cd03",
        "gerrit:https://chromium-review.googlesource.com/c/1383127/4",
        "gpu:none",
        "master:tryserver.chromium.chromiumos",
        "name:mojo_core_unittests",
        "os:Ubuntu-14.04",
        "patch_project:chromium/src",
        "pool:Chrome",
        "priority:30",
        "project:chromium",
        "purpose:ManualTS",
        "purpose:luci",
        "purpose:pre-commit",
        "service_account:none",
        "slavename:swarm1391-c4",
        "spec_name:chromium.try:linux-chromeos-rel",
        "stepname:mojo_core_unittests (with patch)",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None"
      ],
      "deduped_from": "41db9ded68fb5c11",
      "failure": false,
      "modified_ts": "2018-12-19T16:40:11.459171",
      "user": "",
      "created_ts": "2018-12-19T16:40:11.437253",
      "name": "mojo_core_unittests (with patch)/Ubuntu-14.04/ff796fc6ae/linux-chromeos-rel/165248",
      "task_id": "41e0284bc3ef4f10",
      "bot_dimensions": [
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
            "gce-trusty-e833d7b0-us-west1-b-g3nv"
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
            "3945-402d3be"
          ],
          "key": "server_version"
        },
        {
          "value": [
            "us",
            "us-west",
            "us-west1",
            "us-west1-b"
          ],
          "key": "zone"
        }
      ],
      "try_number": "0",
      "current_task_slice": "0",
      "bot_id": "gce-trusty-e833d7b0-us-west1-b-g3nv"
    },
    {
      "created_ts": "2018-12-19T16:22:34.102301",
      "name": "bb-8926695751423164512-chromium-Chromium Mac Goma RBE Staging (dbg)",
      "task_id": "41e0182a00fcc110",
      "tags": [
        "build_address:luci.chromium.ci/Chromium Mac Goma RBE Staging (dbg)/2849",
        "buildbucket_bucket:chromium/ci",
        "buildbucket_build_id:8926695751423164512",
        "buildbucket_hostname:cr-buildbucket.appspot.com",
        "buildbucket_template_canary:0",
        "buildbucket_template_revision:daaff082e95b94bec84fa6e440a8b97677d6f76d",
        "builder:Chromium Mac Goma RBE Staging (dbg)",
        "buildset:commit/git/f5d8d0b4d3beacbccec4638b825df7cda05bf9f3",
        "buildset:commit/gitiles/chromium.googlesource.com/chromium/src/+/f5d8d0b4d3beacbccec4638b825df7cda05bf9f3",
        "caches:builder_5f9d00a5e167f56922678a8467f197b7e00d6e2ea47ac3be386a225df553b989_v2",
        "cores:4",
        "cpu:x86-64",
        "gitiles_ref:refs/heads/master",
        "log_location:logdog://logs.chromium.org/chromium/buildbucket/cr-buildbucket.appspot.com/8926695751423164512/+/annotations",
        "luci_project:chromium",
        "os:Mac-10.13",
        "pool:luci.chromium.ci",
        "priority:30",
        "recipe_name:chromium",
        "recipe_package:infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build",
        "scheduler_invocation_id:9092125813699649312",
        "scheduler_job_id:chromium/Chromium Mac Goma RBE Staging (dbg)",
        "service_account:chromium-ci-builder@chops-service-accounts.iam.gserviceaccount.com",
        "swarming.pool.template:skip",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None",
        "user_agent:luci-scheduler",
        "vpython:native-python-wrapper"
      ],
      "internal_failure": false,
      "server_versions": [
        "3945-402d3be"
      ],
      "abandoned_ts": "2018-12-19T16:22:34.102301",
      "failure": false,
      "state": "NO_RESOURCE",
      "modified_ts": "2018-12-19T16:22:34.240369",
      "user": "",
      "current_task_slice": "0"
    },
    {
      "created_ts": "2018-12-19T16:21:08.652990",
      "name": "bb-8926695841132150096-chromium-Chromium Mac Goma RBE Staging",
      "task_id": "41e016dc85735b10",
      "tags": [
        "build_address:luci.chromium.ci/Chromium Mac Goma RBE Staging/2846",
        "buildbucket_bucket:chromium/ci",
        "buildbucket_build_id:8926695841132150096",
        "buildbucket_hostname:cr-buildbucket.appspot.com",
        "buildbucket_template_canary:0",
        "buildbucket_template_revision:daaff082e95b94bec84fa6e440a8b97677d6f76d",
        "builder:Chromium Mac Goma RBE Staging",
        "buildset:commit/git/f5d8d0b4d3beacbccec4638b825df7cda05bf9f3",
        "buildset:commit/gitiles/chromium.googlesource.com/chromium/src/+/f5d8d0b4d3beacbccec4638b825df7cda05bf9f3",
        "caches:builder_bdb031eead4c88d3bd716a83568659317e7e53ca50cecefb6e56a7f61dd3cec3_v2",
        "cores:4",
        "cpu:x86-64",
        "gitiles_ref:refs/heads/master",
        "log_location:logdog://logs.chromium.org/chromium/buildbucket/cr-buildbucket.appspot.com/8926695841132150096/+/annotations",
        "luci_project:chromium",
        "os:Mac-10.13",
        "pool:luci.chromium.ci",
        "priority:30",
        "recipe_name:chromium",
        "recipe_package:infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build",
        "scheduler_invocation_id:9092125903391076096",
        "scheduler_job_id:chromium/Chromium Mac Goma RBE Staging",
        "service_account:chromium-ci-builder@chops-service-accounts.iam.gserviceaccount.com",
        "swarming.pool.template:skip",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None",
        "user_agent:luci-scheduler",
        "vpython:native-python-wrapper"
      ],
      "internal_failure": false,
      "server_versions": [
        "3945-402d3be"
      ],
      "abandoned_ts": "2018-12-19T16:21:08.652990",
      "failure": false,
      "state": "NO_RESOURCE",
      "modified_ts": "2018-12-19T16:21:08.868984",
      "user": "",
      "current_task_slice": "0"
    },
    {
      "created_ts": "2018-12-19T03:04:34.639358",
      "name": "bb-8926745958003396880-fuchsia-zircon-arm64-clang-release-build_only",
      "task_id": "41dd3d9564402e10",
      "tags": [
        "allow_milo:1",
        "buildbucket_bucket:fuchsia/try",
        "buildbucket_build_id:8926745958003396880",
        "buildbucket_hostname:cr-buildbucket.appspot.com",
        "buildbucket_template_canary:0",
        "buildbucket_template_revision:daaff082e95b94bec84fa6e440a8b97677d6f76d",
        "builder:zircon-arm64-clang-release-build_only",
        "buildset:patch/gerrit/fuchsia-review.googlesource.com/234377/6",
        "caches:builder_b92d0dff9f3233690bf0a887423678df79d9bb0df706bf8aed0872ba726a3f0a_v2",
        "cq_experimental:false",
        "log_location:logdog://logs.chromium.org/fuchsia/buildbucket/cr-buildbucket.appspot.com/8926745958003396880/+/annotations",
        "luci_project:fuchsia",
        "os:Linux",
        "pool:luci.fuchsia.try",
        "priority:30",
        "recipe_name:zircon",
        "recipe_package:fuchsia/infra/recipe_bundles/fuchsia.googlesource.com/infra/recipes",
        "service_account:zircon-try-builder@fuchsia-infra.iam.gserviceaccount.com",
        "swarming.pool.template:skip",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None",
        "user_agent:cq",
        "vpython:native-python-wrapper"
      ],
      "internal_failure": false,
      "server_versions": [
        "3945-402d3be"
      ],
      "abandoned_ts": "2018-12-19T03:05:45.963085",
      "failure": false,
      "state": "CANCELED",
      "modified_ts": "2018-12-19T03:05:45.963085",
      "user": "",
      "current_task_slice": "1"
    },
    {
      "created_ts": "2018-12-19T03:04:34.640316",
      "name": "bb-8926745958003396656-fuchsia-zircon-arm64-gcc-qemu",
      "task_id": "41dd3d950bb52710",
      "tags": [
        "allow_milo:1",
        "buildbucket_bucket:fuchsia/try",
        "buildbucket_build_id:8926745958003396656",
        "buildbucket_hostname:cr-buildbucket.appspot.com",
        "buildbucket_template_canary:0",
        "buildbucket_template_revision:daaff082e95b94bec84fa6e440a8b97677d6f76d",
        "builder:zircon-arm64-gcc-qemu",
        "buildset:patch/gerrit/fuchsia-review.googlesource.com/234377/6",
        "caches:builder_0a058a51460c361272141cfa3c20132bd59ab5703438c02431a2959cc0a0310e_v2",
        "cq_experimental:false",
        "log_location:logdog://logs.chromium.org/fuchsia/buildbucket/cr-buildbucket.appspot.com/8926745958003396656/+/annotations",
        "luci_project:fuchsia",
        "os:Linux",
        "pool:luci.fuchsia.try",
        "priority:30",
        "recipe_name:zircon",
        "recipe_package:fuchsia/infra/recipe_bundles/fuchsia.googlesource.com/infra/recipes",
        "service_account:zircon-try-builder@fuchsia-infra.iam.gserviceaccount.com",
        "swarming.pool.template:skip",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None",
        "user_agent:cq",
        "vpython:native-python-wrapper"
      ],
      "internal_failure": false,
      "server_versions": [
        "3945-402d3be"
      ],
      "abandoned_ts": "2018-12-19T03:05:51.709342",
      "failure": false,
      "state": "CANCELED",
      "modified_ts": "2018-12-19T03:05:51.709342",
      "user": "",
      "current_task_slice": "1"
    },
    {
      "cipd_pins": {
        "packages": [
          {
            "path": ".",
            "version": "63ac2228b1fc1132fe107f7396abfacf3dd9396b",
            "package_name": "infra/tools/luci/kitchen/windows-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "748c524bffe9c29558e998e6a1df9ab3e8821b83",
            "package_name": "infra/git/windows-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "1ba7d485930b05eb07f6bc7724447d6a7c22a6b6",
            "package_name": "infra/python/cpython/windows-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "38514ce7ccccd8463e0a5d8dc9deb12d4bbfb626",
            "package_name": "infra/tools/git/windows-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "c65821d10ef0a90b9acc83c49b06a306cb93f11c",
            "package_name": "infra/tools/luci-auth/windows-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "64841ce7fe1d2be5e1bcd524df6d75cebd800151",
            "package_name": "infra/tools/luci/git-credential-luci/windows-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "1d093caa3e3164f1521a0c51218ebadc042b4312",
            "package_name": "infra/tools/luci/vpython/windows-amd64"
          },
          {
            "path": "clang_win",
            "version": "HMkLX_M8w9awf517UrZm1PPmSpekcYUWC-GsWk9xU34C",
            "package_name": "skia/bots/clang_win"
          }
        ],
        "client_package": {
          "version": "zdnhfpa9SEHKowDgpeM5nc673_9w-3_EmegrKl-VwPcC",
          "package_name": "infra/tools/cipd/windows-amd64"
        }
      },
      "run_id": "41e031b2c8b46711",
      "outputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "690fc083605d7f9a9401d79a18f8c3ced8789c8c"
      },
      "server_versions": [
        "3945-402d3be"
      ],
      "duration": 17.8439998626709,
      "completed_ts": "2018-12-19T16:52:11.201276",
      "started_ts": "2018-12-19T16:50:29.915751",
      "internal_failure": false,
      "exit_code": "1",
      "state": "COMPLETED",
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "tags": [
        "cpu:x86-64-Haswell_GCE",
        "gpu:none",
        "image:windows-server-2016-dc-v20180710",
        "log_location:logdog://logs.chromium.org/skia/20181219T165027.432488369Z_0000000000aa7f5d/+/annotations",
        "luci_project:skia",
        "machine_type:n1-highcpu-64",
        "milo_host:https://ci.chromium.org/raw/build/%s",
        "os:Windows-2016Server-14393",
        "pool:Skia",
        "priority:90",
        "service_account:skia-external-compile-tasks@skia-swarming-bots.iam.gserviceaccount.com",
        "sk_attempt:1",
        "sk_dim_cpu:x86-64-Haswell_GCE",
        "sk_dim_gpu:none",
        "sk_dim_image:windows-server-2016-dc-v20180710",
        "sk_dim_machine_type:n1-highcpu-64",
        "sk_dim_os:Windows-2016Server-14393",
        "sk_dim_pool:Skia",
        "sk_forced_job_id:",
        "sk_id:20181219T165027.432488369Z_0000000000aa7f5d",
        "sk_issue:179027",
        "sk_issue_server:https://skia-review.googlesource.com",
        "sk_name:Build-Win-Clang-x86_64-Debug-ANGLE",
        "sk_parent_task_id:20181219T164559.577581835Z_0000000000aa7f18",
        "sk_parent_task_id:20181219T164559.598749491Z_0000000000aa7f19",
        "sk_patchset:2",
        "sk_repo:https://skia.googlesource.com/skia.git",
        "sk_retry_of:20181219T164812.382095852Z_0000000000aa7f49",
        "sk_revision:77e1ccf3cd19ed079a3c590a67f28f0fa1d73511",
        "source_repo:https://skia.googlesource.com/skia.git/+/%s",
        "source_revision:77e1ccf3cd19ed079a3c590a67f28f0fa1d73511",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:skiabot@example.com"
      ],
      "failure": true,
      "modified_ts": "2018-12-19T16:52:11.201276",
      "user": "skiabot@example.com",
      "created_ts": "2018-12-19T16:50:27.556151",
      "name": "Build-Win-Clang-x86_64-Debug-ANGLE",
      "task_id": "41e031b2c8b46710",
      "bot_dimensions": [
        {
          "value": [
            "git",
            "vpython",
            "work"
          ],
          "key": "caches"
        },
        {
          "value": [
            "64"
          ],
          "key": "cores"
        },
        {
          "value": [
            "x86",
            "x86-64",
            "x86-64-Haswell_GCE"
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
            "skia-gce-610"
          ],
          "key": "id"
        },
        {
          "value": [
            "windows-server-2016-dc-v20180710"
          ],
          "key": "image"
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
            "n1-highcpu-64"
          ],
          "key": "machine_type"
        },
        {
          "value": [
            "Windows",
            "Windows-2016Server",
            "Windows-2016Server-14393"
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
            "2.7.6"
          ],
          "key": "python"
        },
        {
          "value": [
            "3945-402d3be"
          ],
          "key": "server_version"
        },
        {
          "value": [
            "us",
            "us-central",
            "us-central1",
            "us-central1-c"
          ],
          "key": "zone"
        }
      ],
      "try_number": "1",
      "current_task_slice": "0",
      "costs_usd": [
        0.07229456286102291
      ],
      "bot_id": "skia-gce-610"
    },
    {
      "cipd_pins": {
        "packages": [
          {
            "path": ".",
            "version": "cB0p2RtHs2EHLqznE3Ju8Q0sFT8XsBGN5AsB3Kl1g-kC",
            "package_name": "infra/tools/luci/kitchen/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "c07bfdfd244cef1fb87fb2f8fbd0e4296d4b6b42",
            "package_name": "infra/git/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "6dd10e31dc5d4cbb3c8f42a6fbd9485aeeb9ef0c",
            "package_name": "infra/python/cpython/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "WraY8g66hchZ4nqLCdCSdnNyCNa-QDbWOBUGePPHuxAC",
            "package_name": "infra/tools/buildbucket/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "rMx-TrOzggjbW23LQRi9RsvcJ-mq-qKcMZMQ65iwLkIC",
            "package_name": "infra/tools/cloudtail/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "t1n794ArUAFkbYvk0BFF-UZzhGCBc7gPhW0vNz2M7G4C",
            "package_name": "infra/tools/git/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "PloatKD-3d2iygSAaYo9jT6IO2C-wZvEqcto1vndsMoC",
            "package_name": "infra/tools/luci-auth/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "jxxKLB3pyr2EkKg5j9LLqbTnOPTiXSjq82Q4PpynUm8C",
            "package_name": "infra/tools/luci/docker-credential-luci/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "0yxPhZgLyQ6QWYtqO9PwCGEzAAN0exc0wpp4P24BtmMC",
            "package_name": "infra/tools/luci/git-credential-luci/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "LH2D9eSe-sJImw8pxSsKdnKz9LueOAM77FFBJMvP8BwC",
            "package_name": "infra/tools/luci/vpython-native/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "yhDQeJa8hj64DkPAcLKVNQmNTgYsirl5_wvmnHvOz9cC",
            "package_name": "infra/tools/luci/vpython/mac-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "0O-YV-VdtsIyIvEcBluv6fW_qa_ZZ5t6_iod8vYcuhAC",
            "package_name": "infra/tools/prpc/mac-amd64"
          },
          {
            "path": "kitchen-checkout",
            "version": "7hCtf7nHQunIgGyK8xp7k83nBh9jT0X44fLWHZrmhTMC",
            "package_name": "infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build"
          }
        ],
        "client_package": {
          "version": "JE3ZzQ8onwtk6D8nX9JcpqR8OJ3Q3MDh6N0zaSToqWsC",
          "package_name": "infra/tools/cipd/mac-amd64"
        }
      },
      "run_id": "41e0310fe0b7c411",
      "outputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "fa47a85e703554abc3b93101d143f840079331f7"
      },
      "server_versions": [
        "3945-402d3be"
      ],
      "duration": 2.9027669429779053,
      "completed_ts": "2018-12-19T16:50:27.026022",
      "started_ts": "2018-12-19T16:50:10.505241",
      "internal_failure": false,
      "exit_code": "1",
      "state": "COMPLETED",
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "tags": [
        "build_address:luci.chromium.ci/mac-code-coverage-generation/26962",
        "buildbucket_bucket:chromium/ci",
        "buildbucket_build_id:8926694040738803408",
        "buildbucket_hostname:cr-buildbucket.appspot.com",
        "buildbucket_template_canary:0",
        "buildbucket_template_revision:daaff082e95b94bec84fa6e440a8b97677d6f76d",
        "builder:mac-code-coverage-generation",
        "buildset:commit/git/9695ac9fec245f1f303c991a86c61bc73e3ab39d",
        "buildset:commit/gitiles/chromium.googlesource.com/chromium/src/+/9695ac9fec245f1f303c991a86c61bc73e3ab39d",
        "caches:builder_b1d34f992ba739d63ba9119bd7baaab5b46111984e3a29754b6b6b387041680c_v2",
        "cores:24",
        "cpu:x86-64",
        "gitiles_ref:refs/heads/master",
        "log_location:logdog://logs.chromium.org/chromium/buildbucket/cr-buildbucket.appspot.com/8926694040738803408/+/annotations",
        "luci_project:chromium",
        "pool:luci.chromium.ci",
        "priority:30",
        "recipe_name:chromium",
        "recipe_package:infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build",
        "scheduler_invocation_id:9092124102998517712",
        "scheduler_job_id:chromium/mac-code-coverage-generation",
        "service_account:chromium-code-coverage-builder@chops-service-accounts.iam.gserviceaccount.com",
        "swarming.pool.template:skip",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:None",
        "user_agent:luci-scheduler",
        "vpython:native-python-wrapper"
      ],
      "failure": true,
      "modified_ts": "2018-12-19T16:50:27.026022",
      "user": "",
      "created_ts": "2018-12-19T16:49:45.929564",
      "name": "bb-8926694040738803408-chromium-mac-code-coverage-generation",
      "task_id": "41e0310fe0b7c410",
      "bot_dimensions": [
        {
          "value": [
            "mac-code-coverage-generation"
          ],
          "key": "builder"
        },
        {
          "value": [
            "git",
            "goma_v2",
            "vpython"
          ],
          "key": "caches"
        },
        {
          "value": [
            "24"
          ],
          "key": "cores"
        },
        {
          "value": [
            "x86",
            "x86-64",
            "x86-64-E5-2697_v2"
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
            "1002",
            "1002:679e",
            "1002:679e-4.0.11-3.2.8"
          ],
          "key": "gpu"
        },
        {
          "value": [
            "0"
          ],
          "key": "hidpi"
        },
        {
          "value": [
            "build227-m9"
          ],
          "key": "id"
        },
        {
          "value": [
            "MacPro6,1"
          ],
          "key": "mac_model"
        },
        {
          "value": [
            "n1-standard-16"
          ],
          "key": "machine_type"
        },
        {
          "value": [
            "Mac",
            "Mac-10.13",
            "Mac-10.13.4"
          ],
          "key": "os"
        },
        {
          "value": [
            "luci.chromium.ci"
          ],
          "key": "pool"
        },
        {
          "value": [
            "2.7.10"
          ],
          "key": "python"
        },
        {
          "value": [
            "3945-402d3be"
          ],
          "key": "server_version"
        },
        {
          "value": [
            "1"
          ],
          "key": "ssd"
        },
        {
          "value": [
            "9.3"
          ],
          "key": "xcode_version"
        },
        {
          "value": [
            "us",
            "us-golo",
            "us-golo-9"
          ],
          "key": "zone"
        }
      ],
      "try_number": "1",
      "current_task_slice": "1",
      "costs_usd": [
        0.01062192540761886
      ],
      "bot_id": "build227-m9"
    },
    {
      "cipd_pins": {
        "packages": [
          {
            "path": ".",
            "version": "5105c5347912fb481c3422ece2ed5fee722ddb25",
            "package_name": "infra/tools/luci/kitchen/linux-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "df53a719b65668e3b16ecdb600f29f8c901cd67e",
            "package_name": "infra/tools/luci-auth/linux-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "c59ec2607c850d483308e916ebf3333c8b43a3cd",
            "package_name": "infra/tools/luci/vpython/linux-amd64"
          },
          {
            "path": "skimage",
            "version": "sdNO5Pnl5Gk13L6zSSQwpeFVsvSMsDsfkMkdkY7d2mAC",
            "package_name": "skia/bots/skimage"
          },
          {
            "path": "skp",
            "version": "StnfOGJydJop3grW-YzbAyt6mwDlxDrkN-8CqJULlj8C",
            "package_name": "skia/bots/skp"
          },
          {
            "path": "svg",
            "version": "c2784ea640f0c9089ab3ea53775e2d24e1c89f63",
            "package_name": "skia/bots/svg"
          },
          {
            "path": "valgrind",
            "version": "653085d14b77163949368e48ee81b3839721f238",
            "package_name": "skia/bots/valgrind"
          }
        ],
        "client_package": {
          "version": "a2dqpK39PjGpFdcdw62OAE0JOJJ9n8J_AXpJHmH0QCIC",
          "package_name": "infra/tools/cipd/linux-amd64"
        }
      },
      "run_id": "41dfa79d3bf29011",
      "outputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "8f82f443ba36f4c6d7fa4a803cdcfb0b5c22d992"
      },
      "server_versions": [
        "3945-402d3be"
      ],
      "duration": 8175.204389810562,
      "completed_ts": "2018-12-19T16:36:02.766665",
      "started_ts": "2018-12-19T14:19:40.726862",
      "internal_failure": false,
      "exit_code": "0",
      "state": "COMPLETED",
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "tags": [
        "gpu:10de:1cb3-384.59",
        "log_location:logdog://logs.chromium.org/skia/20181219T141938.105162139Z_0000000000aa680f/+/annotations",
        "luci_project:skia",
        "milo_host:https://ci.chromium.org/raw/build/%s",
        "os:Ubuntu-17.04",
        "pool:Skia",
        "priority:90",
        "rack:1",
        "service_account:none",
        "sk_attempt:0",
        "sk_dim_gpu:10de:1cb3-384.59",
        "sk_dim_os:Ubuntu-17.04",
        "sk_dim_pool:Skia",
        "sk_dim_rack:1",
        "sk_dim_valgrind:1",
        "sk_forced_job_id:",
        "sk_id:20181219T141938.105162139Z_0000000000aa680f",
        "sk_name:Perf-Ubuntu17-GCC-Golo-GPU-QuadroP400-x86_64-Release-All-Valgrind_SK_CPU_LIMIT_SSE41",
        "sk_parent_task_id:20181219T133436.071404958Z_0000000000aa638b",
        "sk_parent_task_id:20181219T133528.472439257Z_0000000000aa6399",
        "sk_repo:https://skia.googlesource.com/skia.git",
        "sk_retry_of:",
        "sk_revision:c59caa406007b0ead362dd7eb013a1b635e6742a",
        "source_repo:https://skia.googlesource.com/skia.git/+/%s",
        "source_revision:c59caa406007b0ead362dd7eb013a1b635e6742a",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:skiabot@example.com",
        "valgrind:1"
      ],
      "failure": false,
      "modified_ts": "2018-12-19T16:36:02.766665",
      "user": "skiabot@example.com",
      "created_ts": "2018-12-19T14:19:38.149850",
      "name": "Perf-Ubuntu17-GCC-Golo-GPU-QuadroP400-x86_64-Release-All-Valgrind_SK_CPU_LIMIT_SSE41",
      "task_id": "41dfa79d3bf29010",
      "bot_dimensions": [
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
            "x86-64-E3-1230_v5",
            "x86-64-avx2"
          ],
          "key": "cpu"
        },
        {
          "value": [
            "powersave"
          ],
          "key": "cpu_governor"
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
            "10de:1cb3-384.59"
          ],
          "key": "gpu"
        },
        {
          "value": [
            "build3-a9"
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
            "1"
          ],
          "key": "kvm"
        },
        {
          "value": [
            "en_US.ISO8859-1"
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
            "Linux",
            "Ubuntu",
            "Ubuntu-17.04"
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
            "3945-402d3be"
          ],
          "key": "server_version"
        },
        {
          "value": [
            "1"
          ],
          "key": "ssd"
        },
        {
          "value": [
            "1"
          ],
          "key": "valgrind"
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
      "try_number": "1",
      "current_task_slice": "0",
      "costs_usd": [
        0.9931866545447807
      ],
      "bot_id": "build3-a9"
    },
    {
      "cipd_pins": {
        "packages": [
          {
            "path": ".",
            "version": "5105c5347912fb481c3422ece2ed5fee722ddb25",
            "package_name": "infra/tools/luci/kitchen/linux-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "df53a719b65668e3b16ecdb600f29f8c901cd67e",
            "package_name": "infra/tools/luci-auth/linux-amd64"
          },
          {
            "path": "cipd_bin_packages",
            "version": "c59ec2607c850d483308e916ebf3333c8b43a3cd",
            "package_name": "infra/tools/luci/vpython/linux-amd64"
          },
          {
            "path": "skimage",
            "version": "sdNO5Pnl5Gk13L6zSSQwpeFVsvSMsDsfkMkdkY7d2mAC",
            "package_name": "skia/bots/skimage"
          },
          {
            "path": "skp",
            "version": "StnfOGJydJop3grW-YzbAyt6mwDlxDrkN-8CqJULlj8C",
            "package_name": "skia/bots/skp"
          },
          {
            "path": "svg",
            "version": "c2784ea640f0c9089ab3ea53775e2d24e1c89f63",
            "package_name": "skia/bots/svg"
          },
          {
            "path": "valgrind",
            "version": "653085d14b77163949368e48ee81b3839721f238",
            "package_name": "skia/bots/valgrind"
          }
        ],
        "client_package": {
          "version": "a2dqpK39PjGpFdcdw62OAE0JOJJ9n8J_AXpJHmH0QCIC",
          "package_name": "infra/tools/cipd/linux-amd64"
        }
      },
      "run_id": "41df677202f20311",
      "outputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "20751b9c699bc9af5b6951bb42631bbf2fbd39b6"
      },
      "server_versions": [
        "3945-402d3be"
      ],
      "duration": 4187.02413392067,
      "completed_ts": "2018-12-19T14:19:39.025593",
      "started_ts": "2018-12-19T13:09:38.480038",
      "internal_failure": false,
      "exit_code": "0",
      "state": "COMPLETED",
      "bot_version": "14fa84c24d76966aabd6df511c4f24bad7f85c257fbfc5598ebc215b5d8f9e97",
      "tags": [
        "gpu:10de:1cb3-384.59",
        "log_location:logdog://logs.chromium.org/skia/20181219T130932.632348262Z_0000000000aa6322/+/annotations",
        "luci_project:skia",
        "milo_host:https://ci.chromium.org/raw/build/%s",
        "os:Ubuntu-17.04",
        "pool:Skia",
        "priority:90",
        "service_account:none",
        "sk_attempt:0",
        "sk_dim_gpu:10de:1cb3-384.59",
        "sk_dim_os:Ubuntu-17.04",
        "sk_dim_pool:Skia",
        "sk_dim_valgrind:1",
        "sk_forced_job_id:",
        "sk_id:20181219T130932.632348262Z_0000000000aa6322",
        "sk_name:Test-Ubuntu17-GCC-Golo-GPU-QuadroP400-x86_64-Release-All-Valgrind_PreAbandonGpuContext_SK_CPU_LIMIT_SSE41",
        "sk_parent_task_id:20181219T123715.543824529Z_0000000000aa5fa7",
        "sk_parent_task_id:20181219T123811.487104410Z_0000000000aa5fb1",
        "sk_repo:https://skia.googlesource.com/skia.git",
        "sk_retry_of:",
        "sk_revision:1a237195b527640c6a365affa7d9941d71321c98",
        "source_repo:https://skia.googlesource.com/skia.git/+/%s",
        "source_revision:1a237195b527640c6a365affa7d9941d71321c98",
        "swarming.pool.template:none",
        "swarming.pool.version:a951373af11284d7583d5fd2ca25a760bc744af6",
        "user:skiabot@example.com",
        "valgrind:1"
      ],
      "failure": false,
      "modified_ts": "2018-12-19T14:19:39.025593",
      "user": "skiabot@example.com",
      "created_ts": "2018-12-19T13:09:32.739908",
      "name": "Test-Ubuntu17-GCC-Golo-GPU-QuadroP400-x86_64-Release-All-Valgrind_PreAbandonGpuContext_SK_CPU_LIMIT_SSE41",
      "task_id": "41df677202f20310",
      "bot_dimensions": [
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
            "x86-64-E3-1230_v5",
            "x86-64-avx2"
          ],
          "key": "cpu"
        },
        {
          "value": [
            "powersave"
          ],
          "key": "cpu_governor"
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
            "10de:1cb3-384.59"
          ],
          "key": "gpu"
        },
        {
          "value": [
            "build3-a9"
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
            "1"
          ],
          "key": "kvm"
        },
        {
          "value": [
            "en_US.ISO8859-1"
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
            "Linux",
            "Ubuntu",
            "Ubuntu-17.04"
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
            "3945-402d3be"
          ],
          "key": "server_version"
        },
        {
          "value": [
            "1"
          ],
          "key": "ssd"
        },
        {
          "value": [
            "1"
          ],
          "key": "valgrind"
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
      "try_number": "1",
      "current_task_slice": "0",
      "costs_usd": [
        0.5098954997358307
      ],
      "bot_id": "build3-a9"
    }
  ],
  "now": "2018-12-19T17:31:28.965233"
}