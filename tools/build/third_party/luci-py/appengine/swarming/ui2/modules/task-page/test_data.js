// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

/** taskResult is a helper function for the demo page that parses
 *  the url and returns result data based on the trailing 3 numbers.
 *  This allows the demo page dynamically change based on some
 *  user input, just like the real thing.
 */
export function taskResult(url, opts) {
  let taskId = url.match('/task/(.+)/result');
  taskId = taskId[1] || '000';
  let idx = parseInt(taskId.slice(-3));
  if (!idx) {
    idx = 0;
  }
  return taskResults[idx];
}

export const taskResults = [
  {
    "created_ts": "2019-02-04T16:05:17.601476",
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
          "gce-trusty-e833d7b0-us-east1-b-s2c5"
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
          "4064-3687a02"
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
    "bot_version": "31e15677c83a483c3fc713eb537f60555797bef859c50bbe39c1de2a413adf38",
    "task_id": "testid000",
    "run_id": "42e18650b1b4e411",
    "internal_failure": false,
    "tags": [
      "build_is_experimental:false",
      "buildername:Linux ChromiumOS MSan Tests",
      "buildnumber:11160",
      "cpu:x86-64",
      "data:cdf03f96d6b922b0ef716a69567c7e29014f70d0",
      "gpu:none",
      "master:chromium.memory",
      "name:webui_polymer2_browser_tests",
      "os:Ubuntu-14.04",
      "pool:Chrome",
      "priority:25",
      "project:chromium",
      "purpose:CI",
      "purpose:luci",
      "purpose:post-commit",
      "service_account:none",
      "slavename:swarm2374-c4",
      "spec_name:chromium.ci:Linux ChromiumOS MSan Tests",
      "stepname:webui_polymer2_browser_tests",
      "swarming.pool.template:none",
      "swarming.pool.version:decf85fc72c7df6f8d2d10fd8ede6d81a9699677",
      "user:None"
    ],
    "server_versions": [
      "4064-3687a02"
    ],
    "costs_usd": [
      0.03154403773658218
    ],
    "name": "running task on try number 3",
    "failure": false,
    "state": "RUNNING",
    "modified_ts": "2019-02-04T16:45:51.928202",
    "user": "",
    "bot_id": "gce-trusty-e833d7b0-us-east1-b-s2c5",
    "current_task_slice": "0",
    "try_number": "3",
    "started_ts": "2019-02-04T16:05:49.489858"
  },
  {
    "cipd_pins": {
      "packages": [
        {
          "path": ".",
          "version": "7JNHoA8j-byynAnNNfD93zYxvCrfS_q57UeUhC7oH6YC",
          "package_name": "infra/tools/luci/kitchen/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "b245a31a4df87bd38f7e7d0cf19d492695bd7a7e",
          "package_name": "infra/git/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "46c0c897ca0f053799ee41fd148bb7a47232df47",
          "package_name": "infra/python/cpython/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "J8IGFTojudB9c6rtwsCmlcUA0eCvuf173AdsfeAFe9YC",
          "package_name": "infra/tools/buildbucket/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "_EmLtOFqma-Fdw0ExhHST4uRG3IDfFe8vkba2_1NGZAC",
          "package_name": "infra/tools/cloudtail/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "CCUPRoUSIMjB0H9RYWX4yK7kKAAoCUK864mWSDQdzXIC",
          "package_name": "infra/tools/git/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "cexxITLLto0E5R-VwXpZWQUq1mXCXXGjGbew22M66cMC",
          "package_name": "infra/tools/luci-auth/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "xka0wl1vmSqAJOB7looVmSpSXn_1ztxBtzMc5nN3rqcC",
          "package_name": "infra/tools/luci/docker-credential-luci/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "2FiJ5AgpUA0ardjEakl6gtMPLKwd3X_iQ3HkzFgPNt8C",
          "package_name": "infra/tools/luci/git-credential-luci/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "CxDAdPUaDvFK4dPug3SkX03Cf2Oe2ir67g4I1ZMZ58IC",
          "package_name": "infra/tools/luci/vpython-native/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "uCjugbKg6wMIF6_H_BHECZQdcGRebhnZ6LzSodPHQ7AC",
          "package_name": "infra/tools/luci/vpython/linux-amd64"
        },
        {
          "path": "cipd_bin_packages",
          "version": "qIKuSNcuWDXDxEsV459Y9O38lFmjI0zSFf9fv8bCZ1cC",
          "package_name": "infra/tools/prpc/linux-amd64"
        },
        {
          "path": "kitchen-checkout",
          "version": "KLmG5i5Hnx_RXGGwkowc4S44nF8FXji5guMhHU-pTuMC",
          "package_name": "infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build"
        }
      ],
      "client_package": {
        "version": "yIT5zb0Ieo_5PolHxSBu03UOOGA1iEEXpNISFEoSd-8C",
        "package_name": "infra/tools/cipd/linux-amd64"
      }
    },
    "run_id": "40110b3c0fac7811",
    "outputs_ref": {
      "isolatedserver": "https://isolateserver.appspot.com",
      "namespace": "default-gzip",
      "isolated": "9b7e9a85c14f3012e0395ae1d92d1d7339f5f99e"
    },
    "server_versions": [
      "3779-c5c026e"
    ],
    "performance_stats": {
      "isolated_download": {
        "initial_size": "0",
        "initial_number_items": "0"
      },
      "isolated_upload": {
        "num_items_cold": "2",
        "duration": 0.5382578372955322,
        "total_bytes_items_cold": "12617",
        "items_cold": "eJxrZdyfAAAD+QGm"
      },
      "bot_overhead": 12.625049114227295
    },
    "duration": 881.5171999931335,
    "completed_ts": "2019-01-21T10:42:33.353190",
    "started_ts": "2019-01-21T10:27:38.055897",
    "internal_failure": false,
    "exit_code": "0",
    "state": "COMPLETED",
    "bot_version": "a601d60342c4e8aab332d42ad036f481fab9c080a89f92726c56a2c813228a51",
    "tags": [
      "build_address:luci.chromium.try/linux_chromium_cfi_rel_ng/608",
      "buildbucket_bucket:luci.chromium.try",
      "buildbucket_build_id:8934841822195451424",
      "buildbucket_hostname:cr-buildbucket.appspot.com",
      "buildbucket_template_canary:0",
      "buildbucket_template_revision:1630ff158d8d4118027817e4d74c356b46464ed9",
      "builder:linux_chromium_cfi_rel_ng",
      "buildset:patch/gerrit/chromium-review.googlesource.com/1237113/1",
      "caches:builder_86e11e72bf6f8c2c424eb2189ffc073b483485cf12a42b403fb5526a59936253_v2",
      "cores:32",
      "cpu:x86-64",
      "log_location:logdog://logs.chromium.org/chromium/buildbucket/cr-buildbucket.appspot.com/8934841822195451424/+/annotations",
      "luci_project:chromium",
      "os:Ubuntu-14.04",
      "pool:luci.chromium.try",
      "priority:30",
      "source_repo:https://example.com/repo/%s",
      "source_revision:65432abcdef",
      "recipe_name:chromium_trybot",
      "recipe_package:infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build",
      "service_account:chromium-try-builder@example.iam.gserviceaccount.com",
      "swarming.pool.template:skip",
      "swarming.pool.version:a636fa546b9b663cc0d60eefebb84621a4dfa011",
      "user:None",
      "user_agent:git_cl_try",
      "vpython:native-python-wrapper"
    ],
    "failure": false,
    "modified_ts": "2019-01-21T10:42:33.353190",
    "user": "",
    "created_ts": "2019-01-21T10:24:15.851434",
    "name": "Completed task with 2 slices",
    "task_id": "testid001",
    "bot_dimensions": [
      {
        "value": [
          "linux_chromium_cfi_rel_ng"
        ],
        "key": "builder"
      },
      {
        "value": [
          "32"
        ],
        "key": "cores"
      },
      {
        "value": [
          "x86",
          "x86-64",
          "x86-64-Haswell_GCE",
          "x86-64-avx2"
        ],
        "key": "cpu"
      },
      {
        "value": [
          "8086", "8086:0102"
        ],
        "key": "gpu"
      },
      {
        "value": [
          "swarm1931-c4"
        ],
        "key": "id"
      },
      {
        "value": [
          "chrome-trusty-18042300-b7223b463e3"
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
          "n1-highmem-32"
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
          "luci.chromium.try"
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
          "3779-c5c026e"
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
    "current_task_slice": "1",
    "costs_usd": [
      0.5054023369562476
    ],
    "bot_id": "swarm1931-c4"
  },
  {
    "created_ts": "2019-02-04T15:57:17.067389",
    "name": "Pending task - 1 slice - no rich logs",
    "task_id": "testid002",
    "tags": [
      "build_is_experimental:false",
      "buildername:Android FYI Release (NVIDIA Shield TV)",
      "buildnumber:12247",
      "data:a79744f6cd528bb345b6c79e001523a17e5c83b8",
      "device_os:N",
      "device_type:foster",
      "master:chromium.gpu.fyi",
      "name:gl_tests",
      "os:Android",
      "pool:Chrome-GPU",
      "priority:25",
      "project:chromium",
      "purpose:CI",
      "purpose:luci",
      "purpose:post-commit",
      "service_account:none",
      "slavename:swarm571-c4",
      "spec_name:chromium.ci:Android FYI Release (NVIDIA Shield TV)",
      "stepname:gl_tests on Android device NVIDIA Shield",
      "swarming.pool.template:none",
      "swarming.pool.version:b5e45b934fd19ff0d75d58eb11cdcb149344e3f2",
      "user:None"
    ],
    "internal_failure": false,
    "server_versions": [
      "4055-721ffb4"
    ],
    "failure": false,
    "state": "PENDING",
    "modified_ts": "2019-02-04T15:57:17.157718",
    "user": "",
    "current_task_slice": "0"
  },
  {
    "cipd_pins": {
      "packages": [
        {
          "path": ".swarming_module",
          "version": "1ba7d485930b05eb07f6bc7724447d6a7c22a6b6",
          "package_name": "infra/python/cpython/windows-amd64"
        },
        {
          "path": ".swarming_module",
          "version": "6ebe1bb92c2ff24f74be618f56f4219b8eba551b",
          "package_name": "infra/tools/luci/logdog/butler/windows-amd64"
        },
        {
          "path": ".swarming_module",
          "version": "gdyQzhhSN4yori6wIMZjsqGpgDrkuaB-NREYz4BZ_rMC",
          "package_name": "infra/tools/luci/vpython-native/windows-amd64"
        },
        {
          "path": ".swarming_module",
          "version": "EUJh_9q20TnqjtRumVX8fcDubyBDjOpzAl-sJSKGN2EC",
          "package_name": "infra/tools/luci/vpython/windows-amd64"
        }
      ],
      "client_package": {
        "version": "zdnhfpa9SEHKowDgpeM5nc673_9w-3_EmegrKl-VwPcC",
        "package_name": "infra/tools/cipd/windows-amd64"
      }
    },
    "run_id": "42e0ec5f54b04411",
    "outputs_ref": {
      "isolatedserver": "https://isolateserver.appspot.com",
      "namespace": "default-gzip",
      "isolated": "d50a6620dd3d7907be0d8f79d9aab89c4a7d07e7"
    },
    "server_versions": [
      "4064-3687a02"
    ],
    "duration": 5.328999996185303,
    "completed_ts": "2019-02-04T13:17:34.704866",
    "started_ts": "2019-02-04T13:17:13.632384",
    "cost_saved_usd": 0.002234630678665479,
    "internal_failure": false,
    "exit_code": "0",
    "state": "COMPLETED",
    "bot_version": "31e15677c83a483c3fc713eb537f60555797bef859c50bbe39c1de2a413adf38",
    "tags": [
      "build_is_experimental:false",
      "buildername:Win32 Release (Clang)",
      "buildnumber:15083",
      "gpu:8086:0102",
      "data:4d4a0d0e1d2c04e3530d07f190911235e1209e44",
      "master:client.webrtc",
      "name:video_capture_tests",
      "os:Windows",
      "pool:WebRTC-baremetal",
      "priority:25",
      "project:webrtc",
      "purpose:CI",
      "purpose:luci",
      "purpose:post-commit",
      "service_account:none",
      "slavename:win10-webrtc-8983f7d1-us-central1-c-n32z",
      "spec_name:webrtc.ci:Win32 Release (Clang)",
      "stepname:video_capture_tests on Windows",
      "swarming.pool.template:none",
      "swarming.pool.version:decf85fc72c7df6f8d2d10fd8ede6d81a9699677",
      "user:None"
    ],
    "deduped_from": "42e0ec5f54b04411",
    "failure": false,
    "modified_ts": "2019-02-04T14:28:06.873956",
    "user": "",
    "created_ts": "2019-02-04T14:28:06.823317",
    "name": "deduplicated task with gpu dim",
    "task_id": "testid003",
    "bot_dimensions": [
      {
        "value": [
          "swarming_module_cache_vpython"
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
          "x86-64-E3-1220_V2"
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
          "8086:0102",
          "8086:0102",
          "8086:0102-6.1.7600.16385"
        ],
        "key": "gpu"
      },
      {
        "value": [
          "build20-b3"
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
          "n1-standard-4"
        ],
        "key": "machine_type"
      },
      {
        "value": [
          "Windows",
          "Windows-2008ServerR2",
          "Windows-2008ServerR2-SP1"
        ],
        "key": "os"
      },
      {
        "value": [
          "WebRTC-baremetal"
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
          "4064-3687a02"
        ],
        "key": "server_version"
      },
      {
        "value": [
          "us",
          "us-mtv",
          "us-mtv-chops",
          "us-mtv-chops-b",
          "us-mtv-chops-b-3"
        ],
        "key": "zone"
      }
    ],
    "try_number": "0",
    "current_task_slice": "0",
    "bot_id": "build20-b3"
  },
  {
    "created_ts": "2019-02-04T13:27:06.891224",
    "name": "Expired Task",
    "task_id": "42f58eef9464ab10",
    "tags": [
      "background_task:Repair_2edf5fd5-8898-46b4-b9af-bdc41cba65ea",
      "dut_id:2edf5fd5-8898-46b4-b9af-bdc41cba65ea",
      "dut_state:needs_repair",
      "log_location:logdog://example.com/chromeos/moonshark/86c6d31f-267d-4749-8fcb-18397e3eac7a/+/annotations",
      "luci_project:chromeos",
      "pool:ChromeOSSkylab",
      "priority:30",
      "service_account:none",
      "moonshark:fleet_admin",
      "swarming.pool.template:none",
      "swarming.pool.version:1c55a1fcfe44ea9af5180cbc762b83a830b34e39",
      "user:None"
    ],
    "internal_failure": false,
    "server_versions": [
      "4080-d2e3428"
    ],
    "abandoned_ts": "2019-02-04T13:37:13.437068",
    "failure": false,
    "state": "EXPIRED",
    "modified_ts": "2019-02-04T13:37:13.437068",
    "user": "",
    "completed_ts": "2019-02-04T13:37:13.437068",
    "current_task_slice": "0"
  },
];


/** taskRequest is a helper function for the demo page that parses
 *  the url and returns request data based on the trailing 3 numbers.
 *  This allows the demo page dynamically change based on some
 *  user input, just like the real thing.
 */
export function taskRequest(url, opts) {
  let taskId = url.match('/task/(.+)/request');
  taskId = taskId[1] || '000';
  let idx = parseInt(taskId.slice(-3));
  if (!idx) {
    idx = 0;
  }
  return taskRequests[idx];
}

export const taskRequests = [
  {
    "created_ts": "2019-02-04T16:05:17.601476",
    "authenticated": "user:chromium-ci-builder@chops-service-accounts.iam.gserviceaccount.com",
    "name": "running task on try number 3",
    "tags": [
      "build_is_experimental:false",
      "buildername:Linux ChromiumOS MSan Tests",
      "buildnumber:11160",
      "cpu:x86-64",
      "data:cdf03f96d6b922b0ef716a69567c7e29014f70d0",
      "gpu:none",
      "master:chromium.memory",
      "name:webui_polymer2_browser_tests",
      "os:Ubuntu-14.04",
      "pool:Chrome",
      "priority:25",
      "project:chromium",
      "purpose:CI",
      "purpose:luci",
      "purpose:post-commit",
      "service_account:none",
      "slavename:swarm2374-c4",
      "spec_name:chromium.ci:Linux ChromiumOS MSan Tests",
      "stepname:webui_polymer2_browser_tests",
      "swarming.pool.template:none",
      "swarming.pool.version:decf85fc72c7df6f8d2d10fd8ede6d81a9699677",
      "user:None"
    ],
    "priority": "25",
    "parent_task_id": "42e182c20fc94311",
    "user": "",
    "service_account": "none",
    "task_slices": [
      {
        "expiration_secs": "3600",
        "wait_for_capacity": false,
        "properties": {
          "dimensions": [
            {
              "value": "none",
              "key": "gpu"
            },
            {
              "value": "Ubuntu-14.04",
              "key": "os"
            },
            {
              "value": "x86-64",
              "key": "cpu"
            },
            {
              "value": "Chrome",
              "key": "pool"
            }
          ],
          "idempotent": true,
          "cipd_input": {
            "packages": [
              {
                "path": ".swarming_module",
                "version": "version:2.7.14.chromium14",
                "package_name": "infra/python/cpython/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:e1abc57be62d198b5c2f487bfb2fa2d2eb0e867c",
                "package_name": "infra/tools/luci/logdog/butler/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
                "package_name": "infra/tools/luci/vpython-native/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
                "package_name": "infra/tools/luci/vpython/${platform}"
              }
            ],
            "client_package": {
              "version": "git_revision:6e4acf51a635665e54acaceb8bd073e5c7b8259a",
              "package_name": "infra/tools/cipd/${platform}"
            },
            "server": "https://chrome-infra-packages.appspot.com"
          },
          "extra_args": [
            "--enable-features=WebUIPolymer2",
            "--disable-features=WebUIPolymer2Exceptions",
            "--test-launcher-filter-file=../../testing/buildbot/filters/webui_polymer2_browser_tests.filter",
            "--test-launcher-print-test-stdio=always",
            "--test-launcher-summary-output=${ISOLATED_OUTDIR}/output.json"
          ],
          "io_timeout_secs": "1200",
          "env_prefixes": [
            {
              "value": [
                ".swarming_module",
                ".swarming_module/bin"
              ],
              "key": "PATH"
            },
            {
              "value": [
                ".swarming_module_cache/vpython"
              ],
              "key": "VPYTHON_VIRTUALENV_ROOT"
            }
          ],
          "env": [
            {
              "value": "2",
              "key": "GTEST_SHARD_INDEX"
            },
            {
              "value": "4",
              "key": "GTEST_TOTAL_SHARDS"
            }
          ],
          "execution_timeout_secs": "3600",
          "inputs_ref": {
            "isolatedserver": "https://isolateserver.appspot.com",
            "namespace": "default-gzip",
            "isolated": "cdf03f96d6b922b0ef716a69567c7e29014f70d0"
          },
          "grace_period_secs": "30",
          "caches": [
            {
              "path": ".swarming_module_cache/vpython",
              "name": "swarming_module_cache_vpython"
            }
          ]
        }
      }
    ],
    "expiration_secs": "3600",
    "properties": {
      "dimensions": [
        {
          "value": "none",
          "key": "gpu"
        },
        {
          "value": "Ubuntu-14.04",
          "key": "os"
        },
        {
          "value": "x86-64",
          "key": "cpu"
        },
        {
          "value": "Chrome",
          "key": "pool"
        }
      ],
      "idempotent": true,
      "cipd_input": {
        "packages": [
          {
            "path": ".swarming_module",
            "version": "version:2.7.14.chromium14",
            "package_name": "infra/python/cpython/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:e1abc57be62d198b5c2f487bfb2fa2d2eb0e867c",
            "package_name": "infra/tools/luci/logdog/butler/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
            "package_name": "infra/tools/luci/vpython-native/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
            "package_name": "infra/tools/luci/vpython/${platform}"
          }
        ],
        "client_package": {
          "version": "git_revision:6e4acf51a635665e54acaceb8bd073e5c7b8259a",
          "package_name": "infra/tools/cipd/${platform}"
        },
        "server": "https://chrome-infra-packages.appspot.com"
      },
      "extra_args": [
        "--enable-features=WebUIPolymer2",
        "--disable-features=WebUIPolymer2Exceptions",
        "--test-launcher-filter-file=../../testing/buildbot/filters/webui_polymer2_browser_tests.filter",
        "--test-launcher-print-test-stdio=always",
        "--test-launcher-summary-output=${ISOLATED_OUTDIR}/output.json"
      ],
      "io_timeout_secs": "1200",
      "env_prefixes": [
        {
          "value": [
            ".swarming_module",
            ".swarming_module/bin"
          ],
          "key": "PATH"
        },
        {
          "value": [
            ".swarming_module_cache/vpython"
          ],
          "key": "VPYTHON_VIRTUALENV_ROOT"
        }
      ],
      "env": [
        {
          "value": "2",
          "key": "GTEST_SHARD_INDEX"
        },
        {
          "value": "4",
          "key": "GTEST_TOTAL_SHARDS"
        }
      ],
      "execution_timeout_secs": "3600",
      "inputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "cdf03f96d6b922b0ef716a69567c7e29014f70d0"
      },
      "grace_period_secs": "30",
      "caches": [
        {
          "path": ".swarming_module_cache/vpython",
          "name": "swarming_module_cache_vpython"
        }
      ]
    }
  },
  {
    "created_ts": "2019-01-21T10:24:15.851434",
    "authenticated": "user:iamuser@example.com",
    "name": "Completed task with 2 slices",
    "tags": [
      "build_address:luci.chromium.try/linux_chromium_cfi_rel_ng/608",
      "buildbucket_bucket:luci.chromium.try",
      "buildbucket_build_id:8934841822195451424",
      "buildbucket_hostname:cr-buildbucket.appspot.com",
      "buildbucket_template_canary:0",
      "buildbucket_template_revision:1630ff158d8d4118027817e4d74c356b46464ed9",
      "builder:linux_chromium_cfi_rel_ng",
      "buildset:patch/gerrit/chromium-review.googlesource.com/1237113/1",
      "caches:builder_86e11e72bf6f8c2c424eb2189ffc073b483485cf12a42b403fb5526a59936253_v2",
      "cores:32",
      "cpu:x86-64",
      "log_location:logdog://logs.chromium.org/chromium/buildbucket/cr-buildbucket.appspot.com/8934841822195451424/+/annotations",
      "luci_project:chromium",
      "os:Ubuntu-14.04",
      "pool:luci.chromium.try",
      "priority:30",
      "recipe_name:chromium_trybot",
      "recipe_package:infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build",
      "service_account:chromium-try-builder@example.iam.gserviceaccount.com",
      "swarming.pool.template:skip",
      "swarming.pool.version:a636fa546b9b663cc0d60eefebb84621a4dfa011",
      "source_repo:https://example.com/repo/%s",
      "source_revision:65432abcdef",
      "user:None",
      "user_agent:git_cl_try",
      "vpython:native-python-wrapper"
    ],
    "pubsub_topic": "projects/cr-buildbucket/topics/swarming",
    "priority": "30",
    "pubsub_userdata": "{\"build_id\": 8934841822195451424, \"created_ts\": 1537467855732287, \"swarming_hostname\": \"chromium-swarm.appspot.com\"}",
    "user": "",
    "service_account": "chromium-try-builder@example.iam.gserviceaccount.com",
    "task_slices": [
      {
        "expiration_secs": "120",
        "wait_for_capacity": false,
        "properties": {
          "dimensions": [
            {
              "value": "linux_chromium_cfi_rel_ng",
              "key": "builder"
            },
            {
              "value": "32",
              "key": "cores"
            },
            {
              "value": "Ubuntu-14.04",
              "key": "os"
            },
            {
              "value": "x86-64",
              "key": "cpu"
            },
            {
              "value": "luci.chromium.try",
              "key": "pool"
            },
            {
              "value": "builder_86e11e72bf6f8c2c424eb2189ffc073b483485cf12a42b403fb5526a59936253_v2",
              "key": "caches"
            }
          ],
          "idempotent": false,
          "outputs": ["first_output", "second_output"],
          "cipd_input": {
            "packages": [
              {
                "path": ".",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci/kitchen/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "version:2.17.1.chromium15",
                "package_name": "infra/git/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "version:2.7.14.chromium14",
                "package_name": "infra/python/cpython/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/buildbucket/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/cloudtail/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/git/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci-auth/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:770bd591835116b72af3b6932c8bce3f11c5c6a8",
                "package_name": "infra/tools/luci/docker-credential-luci/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci/git-credential-luci/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci/vpython-native/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci/vpython/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/prpc/${platform}"
              },
              {
                "path": "kitchen-checkout",
                "version": "refs/heads/master",
                "package_name": "infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build"
              }
            ],
            "client_package": {
              "version": "git_revision:fb963f0f43e265a65fb7f1f202e17ea23e947063",
              "package_name": "infra/tools/cipd/${platform}"
            },
            "server": "https://chrome-infra-packages.appspot.com"
          },
          "env_prefixes": [
            {
              "value": [
                "cipd_bin_packages",
                "cipd_bin_packages/bin"
              ],
              "key": "PATH"
            },
            {
              "value": [
                "cache/vpython"
              ],
              "key": "VPYTHON_VIRTUALENV_ROOT"
            }
          ],
          "command": [
            "kitchen${EXECUTABLE_SUFFIX}",
            "cook",
            "-mode",
            "swarming",
            "-build-url",
            "https://ci.chromium.org/p/chromium/builders/luci.chromium.try/linux_chromium_cfi_rel_ng/608",
            "-luci-system-account",
            "system",
            "-repository",
            "",
            "-revision",
            "",
            "-recipe",
            "chromium_trybot",
            "-cache-dir",
            "cache",
            "-checkout-dir",
            "kitchen-checkout",
            "-temp-dir",
            "tmp",
            "-properties",
            "{\"$depot_tools/bot_update\": {\"apply_patch_on_gclient\": true}, \"$kitchen\": {\"devshell\": true, \"git_auth\": true}, \"$recipe_engine/runtime\": {\"is_experimental\": false, \"is_luci\": true}, \"blamelist\": [\"iamuser@example.com\"], \"buildbucket\": {\"build\": {\"bucket\": \"luci.chromium.try\", \"created_by\": \"user:iamuser@example.com\", \"created_ts\": 1537467855227638, \"id\": \"8934841822195451424\", \"project\": \"chromium\", \"tags\": [\"builder:linux_chromium_cfi_rel_ng\", \"buildset:patch/gerrit/chromium-review.googlesource.com/1237113/1\", \"user_agent:git_cl_try\"]}, \"hostname\": \"cr-buildbucket.appspot.com\"}, \"buildername\": \"linux_chromium_cfi_rel_ng\", \"buildnumber\": 608, \"category\": \"git_cl_try\", \"mastername\": \"tryserver.chromium.linux\", \"patch_gerrit_url\": \"https://chromium-review.googlesource.com\", \"patch_issue\": 1237113, \"patch_project\": \"chromium/src\", \"patch_ref\": \"refs/changes/13/1237113/1\", \"patch_repository_url\": \"https://chromium.googlesource.com/chromium/src\", \"patch_set\": 1, \"patch_storage\": \"gerrit\"}",
            "-known-gerrit-host",
            "android.googlesource.com",
            "-known-gerrit-host",
            "boringssl.googlesource.com",
            "-known-gerrit-host",
            "chromium.googlesource.com",
            "-known-gerrit-host",
            "dart.googlesource.com",
            "-known-gerrit-host",
            "fuchsia.googlesource.com",
            "-known-gerrit-host",
            "gn.googlesource.com",
            "-known-gerrit-host",
            "go.googlesource.com",
            "-known-gerrit-host",
            "llvm.googlesource.com",
            "-known-gerrit-host",
            "pdfium.googlesource.com",
            "-known-gerrit-host",
            "skia.googlesource.com",
            "-known-gerrit-host",
            "webrtc.googlesource.com",
            "-logdog-annotation-url",
            "logdog://logs.chromium.org/chromium/buildbucket/cr-buildbucket.appspot.com/8934841822195451424/+/annotations",
            "-output-result-json",
            "${ISOLATED_OUTDIR}/build-run-result.json"
          ],
          "env": [
            {
              "value": "FALSE",
              "key": "BUILDBUCKET_EXPERIMENTAL"
            },
            {
              "value": "/tmp/frobulation",
              "key": "TURBO_ENCAPSULATOR"
            },
          ],
          "execution_timeout_secs": "10800",
          "inputs_ref": {
            "isolatedserver": "https://isolateserver.appspot.com",
            "namespace": "default-gzip"
          },
          "grace_period_secs": "30",
          "caches": [
            {
              "path": "cache/builder",
              "name": "builder_86e11e72bf6f8c2c424eb2189ffc073b483485cf12a42b403fb5526a59936253_v2"
            },
            {
              "path": "cache/git",
              "name": "git"
            },
            {
              "path": "cache/goma",
              "name": "goma_v2"
            },
            {
              "path": "cache/vpython",
              "name": "vpython"
            },
            {
              "path": "cache/win_toolchain",
              "name": "win_toolchain"
            }
          ]
        }
      },
      {
        "expiration_secs": "21480",
        "wait_for_capacity": false,
        "properties": {
          "dimensions": [
            {
              "value": "32",
              "key": "cores"
            },
            {
              "value": "linux_chromium_cfi_rel_ng",
              "key": "builder"
            },
            {
              "value": "Ubuntu-14.04",
              "key": "os"
            },
            {
              "value": "x86-64",
              "key": "cpu"
            },
            {
              "value": "luci.chromium.try",
              "key": "pool"
            }
          ],
          "idempotent": false,
          "outputs": ["first_output", "second_output"],
          "cipd_input": {
            "packages": [
              {
                "path": ".",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci/kitchen/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "version:2.17.1.chromium15",
                "package_name": "infra/git/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "version:2.7.14.chromium14",
                "package_name": "infra/python/cpython/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/buildbucket/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/cloudtail/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/git/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci-auth/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:770bd591835116b72af3b6932c8bce3f11c5c6a8",
                "package_name": "infra/tools/luci/docker-credential-luci/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci/git-credential-luci/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci/vpython-native/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/luci/vpython/${platform}"
              },
              {
                "path": "cipd_bin_packages",
                "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
                "package_name": "infra/tools/prpc/${platform}"
              },
              {
                "path": "kitchen-checkout",
                "version": "refs/heads/master",
                "package_name": "infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build"
              }
            ],
            "client_package": {
              "version": "git_revision:fb963f0f43e265a65fb7f1f202e17ea23e947063",
              "package_name": "infra/tools/cipd/${platform}"
            },
            "server": "https://chrome-infra-packages.appspot.com"
          },
          "env_prefixes": [
            {
              "value": [
                "cipd_bin_packages",
                "cipd_bin_packages/bin"
              ],
              "key": "PATH"
            },
            {
              "value": [
                "cache/vpython"
              ],
              "key": "VPYTHON_VIRTUALENV_ROOT"
            }
          ],
          "command": [
            "kitchen${EXECUTABLE_SUFFIX}",
            "cook",
            "-mode",
            "swarming",
            "-build-url",
            "https://ci.chromium.org/p/chromium/builders/luci.chromium.try/linux_chromium_cfi_rel_ng/608",
            "-luci-system-account",
            "system",
            "-repository",
            "",
            "-revision",
            "",
            "-recipe",
            "chromium_trybot",
            "-cache-dir",
            "cache",
            "-checkout-dir",
            "kitchen-checkout",
            "-temp-dir",
            "tmp",
            "-properties",
            "{\"$depot_tools/bot_update\": {\"apply_patch_on_gclient\": true}, \"$kitchen\": {\"devshell\": true, \"git_auth\": true}, \"$recipe_engine/runtime\": {\"is_experimental\": false, \"is_luci\": true}, \"blamelist\": [\"iamuser@example.com\"], \"buildbucket\": {\"build\": {\"bucket\": \"luci.chromium.try\", \"created_by\": \"user:iamuser@example.com\", \"created_ts\": 1537467855227638, \"id\": \"8934841822195451424\", \"project\": \"chromium\", \"tags\": [\"builder:linux_chromium_cfi_rel_ng\", \"buildset:patch/gerrit/chromium-review.googlesource.com/1237113/1\", \"user_agent:git_cl_try\"]}, \"hostname\": \"cr-buildbucket.appspot.com\"}, \"buildername\": \"linux_chromium_cfi_rel_ng\", \"buildnumber\": 608, \"category\": \"git_cl_try\", \"mastername\": \"tryserver.chromium.linux\", \"patch_gerrit_url\": \"https://chromium-review.googlesource.com\", \"patch_issue\": 1237113, \"patch_project\": \"chromium/src\", \"patch_ref\": \"refs/changes/13/1237113/1\", \"patch_repository_url\": \"https://chromium.googlesource.com/chromium/src\", \"patch_set\": 1, \"patch_storage\": \"gerrit\"}",
            "-known-gerrit-host",
            "android.googlesource.com",
            "-known-gerrit-host",
            "boringssl.googlesource.com",
            "-known-gerrit-host",
            "chromium.googlesource.com",
            "-known-gerrit-host",
            "dart.googlesource.com",
            "-known-gerrit-host",
            "fuchsia.googlesource.com",
            "-known-gerrit-host",
            "gn.googlesource.com",
            "-known-gerrit-host",
            "go.googlesource.com",
            "-known-gerrit-host",
            "llvm.googlesource.com",
            "-known-gerrit-host",
            "pdfium.googlesource.com",
            "-known-gerrit-host",
            "skia.googlesource.com",
            "-known-gerrit-host",
            "webrtc.googlesource.com",
            "-logdog-annotation-url",
            "logdog://logs.chromium.org/chromium/buildbucket/cr-buildbucket.appspot.com/8934841822195451424/+/annotations",
            "-output-result-json",
            "${ISOLATED_OUTDIR}/build-run-result.json"
          ],
          "env": [
            {
              "value": "FALSE",
              "key": "BUILDBUCKET_EXPERIMENTAL"
            },
            {
              "value": "/tmp/frobulation",
              "key": "TURBO_ENCAPSULATOR"
            },
          ],
          "execution_timeout_secs": "10800",
          "inputs_ref": {
            "isolatedserver": "https://isolateserver.appspot.com",
            "namespace": "default-gzip"
          },
          "grace_period_secs": "30",
          "caches": [
            {
              "path": "cache/builder",
              "name": "builder_86e11e72bf6f8c2c424eb2189ffc073b483485cf12a42b403fb5526a59936253_v2"
            },
            {
              "path": "cache/git",
              "name": "git"
            },
            {
              "path": "cache/goma",
              "name": "goma_v2"
            },
            {
              "path": "cache/vpython",
              "name": "vpython"
            },
            {
              "path": "cache/win_toolchain",
              "name": "win_toolchain"
            }
          ]
        }
      }
    ],
    "expiration_secs": "21600",
    "properties": {
      "dimensions": [
        {
          "value": "linux_chromium_cfi_rel_ng",
          "key": "builder"
        },
        {
          "value": "32",
          "key": "cores"
        },
        {
          "value": "Ubuntu-14.04",
          "key": "os"
        },
        {
          "value": "x86-64",
          "key": "cpu"
        },
        {
          "value": "luci.chromium.try",
          "key": "pool"
        },
        {
          "value": "builder_86e11e72bf6f8c2c424eb2189ffc073b483485cf12a42b403fb5526a59936253_v2",
          "key": "caches"
        }
      ],
      "idempotent": false,
      "cipd_input": {
        "packages": [
          {
            "path": ".",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/luci/kitchen/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "version:2.17.1.chromium15",
            "package_name": "infra/git/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "version:2.7.14.chromium14",
            "package_name": "infra/python/cpython/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/buildbucket/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/cloudtail/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/git/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/luci-auth/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:770bd591835116b72af3b6932c8bce3f11c5c6a8",
            "package_name": "infra/tools/luci/docker-credential-luci/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/luci/git-credential-luci/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/luci/vpython-native/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/luci/vpython/${platform}"
          },
          {
            "path": "cipd_bin_packages",
            "version": "git_revision:2c805f1c716f6c5ad2126b27ec88b8585a09481e",
            "package_name": "infra/tools/prpc/${platform}"
          },
          {
            "path": "kitchen-checkout",
            "version": "refs/heads/master",
            "package_name": "infra/recipe_bundles/chromium.googlesource.com/chromium/tools/build"
          }
        ],
        "client_package": {
          "version": "git_revision:fb963f0f43e265a65fb7f1f202e17ea23e947063",
          "package_name": "infra/tools/cipd/${platform}"
        },
        "server": "https://chrome-infra-packages.appspot.com"
      },
      "env_prefixes": [
        {
          "value": [
            "cipd_bin_packages",
            "cipd_bin_packages/bin"
          ],
          "key": "PATH"
        },
        {
          "value": [
            "cache/vpython"
          ],
          "key": "VPYTHON_VIRTUALENV_ROOT"
        }
      ],
      "command": [
        "kitchen${EXECUTABLE_SUFFIX}",
        "cook",
        "-mode",
        "swarming",
        "-build-url",
        "https://ci.chromium.org/p/chromium/builders/luci.chromium.try/linux_chromium_cfi_rel_ng/608",
        "-luci-system-account",
        "system",
        "-repository",
        "",
        "-revision",
        "",
        "-recipe",
        "chromium_trybot",
        "-cache-dir",
        "cache",
        "-checkout-dir",
        "kitchen-checkout",
        "-temp-dir",
        "tmp",
        "-properties",
        "{\"$depot_tools/bot_update\": {\"apply_patch_on_gclient\": true}, \"$kitchen\": {\"devshell\": true, \"git_auth\": true}, \"$recipe_engine/runtime\": {\"is_experimental\": false, \"is_luci\": true}, \"blamelist\": [\"iamuser@example.com\"], \"buildbucket\": {\"build\": {\"bucket\": \"luci.chromium.try\", \"created_by\": \"user:iamuser@example.com\", \"created_ts\": 1537467855227638, \"id\": \"8934841822195451424\", \"project\": \"chromium\", \"tags\": [\"builder:linux_chromium_cfi_rel_ng\", \"buildset:patch/gerrit/chromium-review.googlesource.com/1237113/1\", \"user_agent:git_cl_try\"]}, \"hostname\": \"cr-buildbucket.appspot.com\"}, \"buildername\": \"linux_chromium_cfi_rel_ng\", \"buildnumber\": 608, \"category\": \"git_cl_try\", \"mastername\": \"tryserver.chromium.linux\", \"patch_gerrit_url\": \"https://chromium-review.googlesource.com\", \"patch_issue\": 1237113, \"patch_project\": \"chromium/src\", \"patch_ref\": \"refs/changes/13/1237113/1\", \"patch_repository_url\": \"https://chromium.googlesource.com/chromium/src\", \"patch_set\": 1, \"patch_storage\": \"gerrit\"}",
        "-known-gerrit-host",
        "android.googlesource.com",
        "-known-gerrit-host",
        "boringssl.googlesource.com",
        "-known-gerrit-host",
        "chromium.googlesource.com",
        "-known-gerrit-host",
        "dart.googlesource.com",
        "-known-gerrit-host",
        "fuchsia.googlesource.com",
        "-known-gerrit-host",
        "gn.googlesource.com",
        "-known-gerrit-host",
        "go.googlesource.com",
        "-known-gerrit-host",
        "llvm.googlesource.com",
        "-known-gerrit-host",
        "pdfium.googlesource.com",
        "-known-gerrit-host",
        "skia.googlesource.com",
        "-known-gerrit-host",
        "webrtc.googlesource.com",
        "-logdog-annotation-url",
        "logdog://logs.chromium.org/chromium/buildbucket/cr-buildbucket.appspot.com/8934841822195451424/+/annotations",
        "-output-result-json",
        "${ISOLATED_OUTDIR}/build-run-result.json"
      ],
      "env": [
        {
          "value": "FALSE",
          "key": "BUILDBUCKET_EXPERIMENTAL"
        }
      ],
      "execution_timeout_secs": "10800",
      "inputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip"
      },
      "grace_period_secs": "30",
      "caches": [
        {
          "path": "cache/builder",
          "name": "builder_86e11e72bf6f8c2c424eb2189ffc073b483485cf12a42b403fb5526a59936253_v2"
        },
        {
          "path": "cache/git",
          "name": "git"
        },
        {
          "path": "cache/goma",
          "name": "goma_v2"
        },
        {
          "path": "cache/vpython",
          "name": "vpython"
        },
        {
          "path": "cache/win_toolchain",
          "name": "win_toolchain"
        }
      ]
    }
  },
  {
    "created_ts": "2019-02-04T15:57:17.067389",
    "authenticated": "user:chromium-ci-gpu-builder@example.iam.gserviceaccount.com",
    "name": "Pending task - 1 slice - no rich logs",
    "tags": [
      "build_is_experimental:false",
      "buildername:Android FYI Release (NVIDIA Shield TV)",
      "buildnumber:12247",
      "data:a79744f6cd528bb345b6c79e001523a17e5c83b8",
      "device_os:N",
      "device_type:foster",
      "master:chromium.gpu.fyi",
      "name:gl_tests",
      "os:Android",
      "pool:Chrome-GPU",
      "priority:25",
      "project:chromium",
      "purpose:CI",
      "purpose:luci",
      "purpose:post-commit",
      "service_account:none",
      "slavename:swarm571-c4",
      "spec_name:chromium.ci:Android FYI Release (NVIDIA Shield TV)",
      "stepname:gl_tests on Android device NVIDIA Shield",
      "swarming.pool.template:none",
      "swarming.pool.version:b5e45b934fd19ff0d75d58eb11cdcb149344e3f2",
      "user:None"
    ],
    "priority": "25",
    "parent_task_id": "42d7b13b82d74511",
    "user": "",
    "service_account": "none",
    "task_slices": [
      {
        "expiration_secs": "3600",
        "wait_for_capacity": false,
        "properties": {
          "dimensions": [
            {
              "value": "N",
              "key": "device_os"
            },
            {
              "value": "Android",
              "key": "os"
            },
            {
              "value": "Chrome-GPU",
              "key": "pool"
            },
            {
              "value": "foster",
              "key": "device_type"
            }
          ],
          "idempotent": true,
          "cipd_input": {
            "packages": [
              {
                "path": ".swarming_module",
                "version": "version:2.7.14.chromium14",
                "package_name": "infra/python/cpython/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:e1abc57be62d198b5c2f487bfb2fa2d2eb0e867c",
                "package_name": "infra/tools/luci/logdog/butler/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
                "package_name": "infra/tools/luci/vpython-native/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
                "package_name": "infra/tools/luci/vpython/${platform}"
              },
              {
                "path": "bin",
                "version": "git_revision:ff387eadf445b24c935f1cf7d6ddd279f8a6b04c",
                "package_name": "infra/tools/luci/logdog/butler/${platform}"
              }
            ],
            "client_package": {
              "version": "git_revision:6e4acf51a635665e54acaceb8bd073e5c7b8259a",
              "package_name": "infra/tools/cipd/${platform}"
            },
            "server": "https://chrome-infra-packages.appspot.com"
          },
          "extra_args": [
            "--gs-results-bucket=chromium-result-details",
            "--recover-devices",
            "--test-launcher-summary-output=${ISOLATED_OUTDIR}/output.json"
          ],
          "io_timeout_secs": "1200",
          "env_prefixes": [
            {
              "value": [
                ".swarming_module",
                ".swarming_module/bin"
              ],
              "key": "PATH"
            },
            {
              "value": [
                ".swarming_module_cache/vpython"
              ],
              "key": "VPYTHON_VIRTUALENV_ROOT"
            }
          ],
          "execution_timeout_secs": "3600",
          "inputs_ref": {
            "isolatedserver": "https://isolateserver.appspot.com",
            "namespace": "default-gzip",
            "isolated": "a79744f6cd528bb345b6c79e001523a17e5c83b8"
          },
          "grace_period_secs": "30",
          "caches": [
            {
              "path": ".swarming_module_cache/vpython",
              "name": "swarming_module_cache_vpython"
            }
          ]
        }
      }
    ],
    "expiration_secs": "3600",
    "properties": {
      "dimensions": [
        {
          "value": "N",
          "key": "device_os"
        },
        {
          "value": "Android",
          "key": "os"
        },
        {
          "value": "Chrome-GPU",
          "key": "pool"
        },
        {
          "value": "foster",
          "key": "device_type"
        }
      ],
      "idempotent": true,
      "cipd_input": {
        "packages": [
          {
            "path": ".swarming_module",
            "version": "version:2.7.14.chromium14",
            "package_name": "infra/python/cpython/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:e1abc57be62d198b5c2f487bfb2fa2d2eb0e867c",
            "package_name": "infra/tools/luci/logdog/butler/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
            "package_name": "infra/tools/luci/vpython-native/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
            "package_name": "infra/tools/luci/vpython/${platform}"
          },
          {
            "path": "bin",
            "version": "git_revision:ff387eadf445b24c935f1cf7d6ddd279f8a6b04c",
            "package_name": "infra/tools/luci/logdog/butler/${platform}"
          }
        ],
        "client_package": {
          "version": "git_revision:6e4acf51a635665e54acaceb8bd073e5c7b8259a",
          "package_name": "infra/tools/cipd/${platform}"
        },
        "server": "https://chrome-infra-packages.appspot.com"
      },
      "extra_args": [
        "--gs-results-bucket=chromium-result-details",
        "--recover-devices",
        "--test-launcher-summary-output=${ISOLATED_OUTDIR}/output.json"
      ],
      "io_timeout_secs": "1200",
      "env_prefixes": [
        {
          "value": [
            ".swarming_module",
            ".swarming_module/bin"
          ],
          "key": "PATH"
        },
        {
          "value": [
            ".swarming_module_cache/vpython"
          ],
          "key": "VPYTHON_VIRTUALENV_ROOT"
        }
      ],
      "execution_timeout_secs": "3600",
      "inputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "a79744f6cd528bb345b6c79e001523a17e5c83b8"
      },
      "grace_period_secs": "30",
      "caches": [
        {
          "path": ".swarming_module_cache/vpython",
          "name": "swarming_module_cache_vpython"
        }
      ]
    }
  },
  {
    "created_ts": "2019-02-04T14:28:06.823317",
    "authenticated": "user:webrtc-ci-builder@chops-service-accounts.iam.gserviceaccount.com",
    "name": "deduplicated task with gpu dim",
    "tags": [
      "build_is_experimental:false",
      "buildername:Win32 Release (Clang)",
      "buildnumber:15083",
      "cpu:x86",
      "data:4d4a0d0e1d2c04e3530d07f190911235e1209e44",
      "master:client.webrtc",
      "name:video_capture_tests",
      "os:Windows",
      "pool:WebRTC-baremetal",
      "priority:25",
      "project:webrtc",
      "purpose:CI",
      "purpose:luci",
      "purpose:post-commit",
      "service_account:none",
      "slavename:win10-webrtc-8983f7d1-us-central1-c-n32z",
      "spec_name:webrtc.ci:Win32 Release (Clang)",
      "stepname:video_capture_tests on Windows",
      "swarming.pool.template:none",
      "swarming.pool.version:decf85fc72c7df6f8d2d10fd8ede6d81a9699677",
      "user:None"
    ],
    "priority": "25",
    "parent_task_id": "42e129d60f62f911",
    "user": "",
    "service_account": "none",
    "task_slices": [
      {
        "expiration_secs": "3600",
        "wait_for_capacity": false,
        "properties": {
          "dimensions": [
            {
              "value": "Windows",
              "key": "os"
            },
            {
              "value": "8086:0102",
              "key": "gpu"
            },
            {
              "value": "WebRTC-baremetal",
              "key": "pool"
            }
          ],
          "idempotent": true,
          "cipd_input": {
            "packages": [
              {
                "path": ".swarming_module",
                "version": "version:2.7.14.chromium14",
                "package_name": "infra/python/cpython/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:e1abc57be62d198b5c2f487bfb2fa2d2eb0e867c",
                "package_name": "infra/tools/luci/logdog/butler/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
                "package_name": "infra/tools/luci/vpython-native/${platform}"
              },
              {
                "path": ".swarming_module",
                "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
                "package_name": "infra/tools/luci/vpython/${platform}"
              }
            ],
            "client_package": {
              "version": "git_revision:6e4acf51a635665e54acaceb8bd073e5c7b8259a",
              "package_name": "infra/tools/cipd/${platform}"
            },
            "server": "https://chrome-infra-packages.appspot.com"
          },
          "extra_args": [
            "--isolated-script-test-output=${ISOLATED_OUTDIR}/output.json",
            "--isolated-script-test-perf-output=${ISOLATED_OUTDIR}/perftest-output.json"
          ],
          "io_timeout_secs": "1200",
          "env_prefixes": [
            {
              "value": [
                ".swarming_module",
                ".swarming_module/bin"
              ],
              "key": "PATH"
            },
            {
              "value": [
                ".swarming_module_cache/vpython"
              ],
              "key": "VPYTHON_VIRTUALENV_ROOT"
            }
          ],
          "execution_timeout_secs": "3600",
          "inputs_ref": {
            "isolatedserver": "https://isolateserver.appspot.com",
            "namespace": "default-gzip",
            "isolated": "4d4a0d0e1d2c04e3530d07f190911235e1209e44"
          },
          "grace_period_secs": "30",
          "caches": [
            {
              "path": ".swarming_module_cache/vpython",
              "name": "swarming_module_cache_vpython"
            }
          ]
        }
      }
    ],
    "expiration_secs": "3600",
    "properties": {
      "dimensions": [
        {
          "value": "Windows",
          "key": "os"
        },
        {
          "value": "8086:0102",
          "key": "gpu"
        },
        {
          "value": "WebRTC-baremetal",
          "key": "pool"
        }
      ],
      "idempotent": true,
      "cipd_input": {
        "packages": [
          {
            "path": ".swarming_module",
            "version": "version:2.7.14.chromium14",
            "package_name": "infra/python/cpython/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:e1abc57be62d198b5c2f487bfb2fa2d2eb0e867c",
            "package_name": "infra/tools/luci/logdog/butler/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
            "package_name": "infra/tools/luci/vpython-native/${platform}"
          },
          {
            "path": ".swarming_module",
            "version": "git_revision:96f81e737868d43124b4661cf1c325296ca04944",
            "package_name": "infra/tools/luci/vpython/${platform}"
          }
        ],
        "client_package": {
          "version": "git_revision:6e4acf51a635665e54acaceb8bd073e5c7b8259a",
          "package_name": "infra/tools/cipd/${platform}"
        },
        "server": "https://chrome-infra-packages.appspot.com"
      },
      "extra_args": [
        "--isolated-script-test-output=${ISOLATED_OUTDIR}/output.json",
        "--isolated-script-test-perf-output=${ISOLATED_OUTDIR}/perftest-output.json"
      ],
      "io_timeout_secs": "1200",
      "env_prefixes": [
        {
          "value": [
            ".swarming_module",
            ".swarming_module/bin"
          ],
          "key": "PATH"
        },
        {
          "value": [
            ".swarming_module_cache/vpython"
          ],
          "key": "VPYTHON_VIRTUALENV_ROOT"
        }
      ],
      "execution_timeout_secs": "3600",
      "inputs_ref": {
        "isolatedserver": "https://isolateserver.appspot.com",
        "namespace": "default-gzip",
        "isolated": "4d4a0d0e1d2c04e3530d07f190911235e1209e44"
      },
      "grace_period_secs": "30",
      "caches": [
        {
          "path": ".swarming_module_cache/vpython",
          "name": "swarming_module_cache_vpython"
        }
      ]
    }
  },
  {
    "created_ts": "2019-02-04T13:27:06.891224",
    "authenticated": "user:staging-user@appspot.gserviceaccount.com",
    "name": "Expired Task",
    "tags": [
      "background_task:Repair_2edf5fd5-8898-46b4-b9af-bdc41cba65ea",
      "dut_id:2edf5fd5-8898-46b4-b9af-bdc41cba65ea",
      "dut_state:needs_repair",
      "log_location:logdog://example.com/chromeos/moonshark/86c6d31f-267d-4749-8fcb-18397e3eac7a/+/annotations",
      "luci_project:chromeos",
      "moonshark:fleet_admin",
      "pool:ChromeOSSkylab",
      "priority:30",
      "service_account:none",
      "swarming.pool.template:none",
      "swarming.pool.version:1c55a1fcfe44ea9af5180cbc762b83a830b34e39",
      "user:None"
    ],
    "priority": "30",
    "user": "",
    "service_account": "none",
    "task_slices": [
      {
        "expiration_secs": "600",
        "wait_for_capacity": true,
        "properties": {
          "dimensions": [
            {
              "value": "needs_repair",
              "key": "dut_state"
            },
            {
              "value": "ChromeOSSkylab",
              "key": "pool"
            },
            {
              "value": "2edf5fd5-8898-46b4-b9af-bdc41cba65ea",
              "key": "dut_id"
            }
          ],
          "idempotent": false,
          "inputs_ref": {
            "isolatedserver": "https://isolateserver-dev.appspot.com",
            "namespace": "default-gzip"
          },
          "command": [
            "/opt/infra-tools/moonshark_swarming_worker",
            "-task-name",
            "admin_repair",
            "-logdog-annotation-url",
            "logdog://example.com/chromeos/moonshark/86c6d31f-267d-4749-8fcb-18397e3eac7a/+/annotations"
          ],
          "execution_timeout_secs": "5400",
          "grace_period_secs": "30"
        }
      }
    ],
    "expiration_secs": "600",
    "properties": {
      "dimensions": [
        {
          "value": "needs_repair",
          "key": "dut_state"
        },
        {
          "value": "ChromeOSSkylab",
          "key": "pool"
        },
        {
          "value": "2edf5fd5-8898-46b4-b9af-bdc41cba65ea",
          "key": "dut_id"
        }
      ],
      "idempotent": false,
      "inputs_ref": {
        "isolatedserver": "https://isolateserver-dev.appspot.com",
        "namespace": "default-gzip"
      },
      "command": [
        "/opt/infra-tools/moonshark_swarming_worker",
        "-task-name",
        "admin_repair",
        "-logdog-annotation-url",
        "logdog://example.com/chromeos/moonshark/86c6d31f-267d-4749-8fcb-18397e3eac7a/+/annotations"
      ],
      "execution_timeout_secs": "5400",
      "grace_period_secs": "30"
    }
  },
];

export const taskOutput = {
  'output': `Lorem ipsum dolor sit amet, consectetur adipiscing elit. In venenatis aliquet nunc non faucibus. Mauris ornare ligula eu arcu sagittis vulputate. Nullam cursus vulputate odio venenatis pretium. Suspendisse imperdiet metus eros, in vulputate lacus fringilla non. Suspendisse hendrerit tellus eu laoreet ornare. Maecenas metus ipsum, consectetur a consectetur efficitur, lobortis dapibus turpis. Donec iaculis enim lacus, pulvinar tempor elit dignissim ut. Integer dapibus lorem id ante consequat rutrum. Vestibulum varius neque non dolor tincidunt, id tincidunt lacus finibus. Integer aliquam tellus a suscipit lacinia. Fusce rutrum scelerisque mauris, posuere tempus magna. Suspendisse nec nibh pulvinar, convallis purus ut, semper libero. Sed dapibus velit sed porta auctor. Proin molestie tincidunt odio, a tristique libero ullamcorper ac. Praesent elementum nec enim et ultricies.

Donec ligula orci, placerat a pharetra aliquet, tincidunt sed diam. Mauris venenatis aliquam erat, et egestas ligula congue nec. Morbi accumsan arcu et nibh facilisis, a pretium sapien commodo. Nullam iaculis sit amet purus sit amet bibendum. Sed commodo purus et justo euismod, at dapibus arcu tristique. Etiam pharetra sapien eu quam molestie fringilla. Quisque dignissim tristique enim, non gravida ligula elementum eget. Suspendisse elit elit, molestie vitae consequat non, malesuada sed dolor. Phasellus tempor tellus placerat accumsan posuere. Praesent id diam arcu. Praesent imperdiet nibh vel justo vehicula lacinia. Maecenas sed dolor ac arcu dapibus suscipit sit amet non libero. Nulla auctor turpis non urna aliquam facilisis. Cras quam ex, placerat ut leo a, hendrerit tempus urna. Mauris vitae condimentum mi.

Curabitur et fermentum justo, eu mollis lectus. Morbi velit metus, rhoncus at molestie at, euismod ut nisi. Morbi vitae libero cursus, consequat ante vel, cursus neque. Sed nec nibh non elit mattis iaculis in ac mauris. Vivamus feugiat porta urna. Cras quis ligula fringilla, mattis arcu at, faucibus velit. Pellentesque a efficitur nunc. Sed volutpat dignissim tortor, a facilisis risus mattis quis. Donec vitae justo pellentesque, lobortis turpis bibendum, semper ipsum. Suspendisse convallis risus sed justo mattis, a imperdiet tellus consectetur. Sed porta dui nec justo ultrices ultrices. Donec eget blandit est. Suspendisse vestibulum lobortis enim, at congue orci rhoncus at. Nunc pellentesque enim a semper suscipit. Nunc auctor accumsan nisi ac mollis. Duis lobortis sapien eu felis aliquam pellentesque.

Suspendisse non auctor quam. Etiam dui mauris, iaculis et eros in, egestas pretium purus. Nunc aliquam non felis eget tincidunt. Vivamus interdum hendrerit elementum. Morbi euismod vel nulla quis bibendum. In gravida orci accumsan hendrerit sollicitudin. Nullam pharetra est bibendum felis sagittis placerat. Aenean vehicula, dolor sed sollicitudin viverra, magna quam semper ligula, quis elementum ipsum dui vel est. Mauris convallis libero sit amet augue cursus, quis dignissim turpis venenatis.

Fusce sit amet posuere orci, eget fringilla sapien. Morbi volutpat ante commodo diam tempor, id cursus nulla porta. Ut id lobortis leo, volutpat aliquet leo. Duis auctor purus id odio laoreet congue. Proin luctus velit at augue fringilla, sit amet feugiat eros euismod. Praesent vel lacus mi. Duis euismod sapien at nulla blandit, in pellentesque turpis vehicula. Donec suscipit congue augue.

Aliquam sollicitudin nisl vitae blandit imperdiet. In tempus, felis ac placerat laoreet, tortor ante rhoncus risus, nec efficitur neque orci a diam. Vivamus eleifend auctor magna et consequat. Ut ligula erat, faucibus nec aliquam euismod, facilisis luctus nisl. Cras commodo hendrerit malesuada. Sed sollicitudin in tortor sit amet venenatis. Quisque placerat vel magna vel pretium. Donec consectetur, ante vitae sagittis fringilla, elit nisl faucibus est, id pharetra nibh lectus vitae ligula. Maecenas diam arcu, dignissim eu turpis et, congue placerat nibh. Sed ipsum nisi, iaculis venenatis felis ac, consequat dictum risus. Etiam tristique tempus ligula, at eleifend ex viverra et. Cras velit arcu, dapibus id ante et, tincidunt ornare ante. Quisque vel interdum tellus. Donec tellus nulla, semper quis nisl ac, facilisis luctus nisi. Nunc tincidunt urna ac porttitor interdum.

Nunc elementum suscipit velit eu ultrices. Nam congue posuere lorem a accumsan. Integer vulputate tortor a lorem euismod tempus. Donec ultrices nulla lectus, sed dapibus sapien viverra auctor. Aenean gravida vel ante vitae convallis. Nulla erat dui, semper at odio sed, sollicitudin tincidunt leo. Nulla facilisi. Quisque mollis porttitor tempor. Cras vel sodales est. Donec eleifend ultrices commodo. In mollis, nisi ut vulputate imperdiet, urna erat convallis felis, in eleifend felis dolor non odio. Pellentesque eleifend finibus nulla. Vivamus pharetra ante vitae purus cursus, at tincidunt nisl aliquam.

Nam non sem a nisl dignissim facilisis id eget nibh. Quisque accumsan vulputate lobortis. Mauris fermentum tristique sapien, id tincidunt sapien. Vestibulum quis mi nec tellus sollicitudin gravida. Aenean tempus risus vitae neque consectetur imperdiet. Donec a interdum magna, sit amet finibus eros. Suspendisse egestas pellentesque ipsum, quis dignissim metus pretium a. Nam mattis ex quam, in aliquam arcu interdum sed. Sed sapien justo, condimentum vel auctor sit amet, ultricies et orci.

Aliquam elementum rhoncus lorem ut lobortis. Proin congue varius velit. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Fusce fermentum lacus erat, eget fermentum urna ullamcorper eu. Phasellus posuere arcu quis augue porta, nec facilisis mi posuere. Pellentesque finibus metus quis odio tempor semper. Praesent sollicitudin molestie velit, in convallis arcu. Nullam sed risus a turpis placerat dignissim. In mattis justo sed est egestas tincidunt.

Maecenas ornare tortor ac risus tincidunt feugiat. Nulla facilisi. Morbi a mi laoreet, lacinia nisl eget, dignissim elit. Aenean placerat, lacus at aliquet elementum, quam sem iaculis mauris, quis pellentesque neque nunc ac nibh. Vivamus a dolor ut orci porttitor condimentum. Vestibulum malesuada dui ac risus iaculis interdum. Vivamus facilisis nulla neque, eu hendrerit erat lacinia at. Sed purus justo, imperdiet sed mauris et, cursus vehicula justo. In mollis malesuada lectus sit amet venenatis. Vivamus ullamcorper ipsum diam.

Aliquam ut sapien turpis. Nulla posuere dignissim augue, id facilisis nisl sodales ut. Maecenas vitae lorem id libero dignissim mollis. Phasellus ullamcorper ante non condimentum laoreet. Pellentesque sit amet semper mauris, ut fringilla est. Curabitur lorem eros, interdum id varius non, tristique in dui. Praesent a ante non purus vulputate sagittis. Cras aliquet, lectus sit amet tristique lacinia, leo massa accumsan nisl, id consequat orci mauris et dolor. Cras semper ante quis elementum lobortis. Mauris vitae blandit nisi. Aliquam sit amet tellus non nunc pellentesque laoreet. Maecenas vehicula massa in egestas elementum. Phasellus nisl magna, congue sit amet augue ac, blandit tempor erat.

Curabitur bibendum ultricies ante vel commodo. Pellentesque efficitur, tortor at rutrum dapibus, elit erat efficitur neque, at laoreet risus nibh vel eros. In vel finibus eros, id blandit mauris. Proin ultricies placerat enim et dignissim. Mauris id nunc congue, aliquet risus non, tempor libero. Nam accumsan risus justo, et consectetur lacus pretium et. Duis lobortis, lectus non fringilla aliquet, erat odio malesuada ex, quis placerat diam mauris at purus. Fusce ligula justo, suscipit a lorem nec, finibus laoreet sem.

`};