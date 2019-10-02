// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This file has some data used for bot-list tests.

export const fleetCount = {
  "count": "11434",
  "busy": "11223",
  "dead": "55",
  "quarantined": "83",
  "maintenance": "6",
  "now": "2018-05-14T12:28:07.316441"
};

export const queryCount = {
  "count": "434",
  "busy": "223",
  "dead": "5",
  "quarantined": "3",
  "maintenance": "0",
  "now": "2018-05-14T12:28:06.316441"
};

export const bots_10 = {
    "death_timeout": "600",
    "items": [
        {
            "authenticated_as": "bot:somebot77-a3.fleet.example.com",
            "bot_id": "somebot77-a3",
            "deleted": false,
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["git", "git_cache", "vpython", "work"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-E3-1230_v5", "x86-64-avx2"]
                },
                {
                    "key": "cpu_governor",
                    "value": ["powersave"]
                },
                {
                    "key": "gpu",
                    "value": ["10de", "10de:1cb3", "10de:1cb3-384.59"]
                },
                {
                    "key": "id",
                    "value": ["somebot77-a3"]
                },
                {
                    "key": "inside_docker",
                    "value": ["0"]
                },
                {
                    "key": "kvm",
                    "value": ["1"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.ISO8859-1"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-standard-8"]
                },
                {
                    "key": "os",
                    "value": ["Linux", "Ubuntu", "Ubuntu-17.04"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                },
                {
                    "key": "ssd",
                    "value": ["1"]
                },
                {
                    "key": "valgrind",
                    "value": ["1"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-nyc", "us-nyc-ellis", "us-nyc-ellis-a", "us-nyc-ellis-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-03T22:05:44.251330",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T12:44:09.854911",
            "quarantined": false,
            "state": "{\"audio\":[\"NVIDIA Corporation [10de]: Device [0fb9]\"],\"bot_group_cfg_version\":\"hash:111884aae9bc32\",\"cost_usd_hour\":0.4369786295572917,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"/b/s\",\"disks\":{\"/\":{\"free_mb\":680751.3,\"size_mb\":741467.3},\"/boot\":{\"free_mb\":842.2,\"size_mb\":922.0}},\"env\":{\"PATH\":\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin\"},\"files\":{\"/usr/share/fonts/truetype/\":[\"Gargi\",\"Gubbi\",\"Nakula\",\"Navilu\",\"Sahadeva\",\"Sarai\",\"abyssinica\",\"ancient-scripts\",\"dejavu\",\"droid\",\"fonts-beng-extra\",\"fonts-deva-extra\",\"fonts-gujr-extra\",\"fonts-guru-extra\",\"fonts-japanese-gothic.ttf\",\"fonts-japanese-mincho.ttf\",\"fonts-kalapi\",\"fonts-orya-extra\",\"fonts-telu-extra\",\"freefont\",\"kacst\",\"kacst-one\",\"lao\",\"lato\",\"liberation\",\"lohit-assamese\",\"lohit-bengali\",\"lohit-devanagari\",\"lohit-gujarati\",\"lohit-kannada\",\"lohit-malayalam\",\"lohit-oriya\",\"lohit-punjabi\",\"lohit-tamil\",\"lohit-tamil-classical\",\"lohit-telugu\",\"malayalam-fonts\",\"msttcorefonts\",\"nanum\",\"noto\",\"openoffice\",\"padauk\",\"pagul\",\"samyak\",\"samyak-fonts\",\"sinhala\",\"takao-gothic\",\"tibetan-machine\",\"tlwg\",\"ttf-dejavu\",\"ttf-khmeros-core\",\"ubuntu-font-family\"]},\"gpu\":[\"Nvidia Quadro P400 384.59\"],\"hostname\":\"somebot77-a3.fleet.example.com\",\"ip\":\"192.168.216.11\",\"nb_files_in_temp\":15,\"pid\":2162,\"python\":{\"executable\":\"/usr/bin/python\",\"packages\":[\"adium-theme-ubuntu==0.3.4\",\"CherryPy==3.5.0\",\"cryptography==1.7.1\",\"enum34==1.1.6\",\"httplib2==0.9.2\",\"idna==2.2\",\"ipaddress==1.0.17\",\"keyring==10.3.1\",\"keyrings.alt==2.2\",\"ndg-httpsclient==0.4.2\",\"numpy==1.12.1\",\"Pillow==4.0.0\",\"psutil==5.0.1\",\"pyasn1==0.1.9\",\"pycrypto==2.6.1\",\"pycurl==7.43.0\",\"pygobject==3.22.0\",\"pyOpenSSL==16.2.0\",\"pyserial==3.2.1\",\"python-apt==1.4.0b2\",\"pyxdg==0.25\",\"PyYAML==3.12\",\"repoze.lru==0.6\",\"Routes==2.2\",\"scour==0.32\",\"SecretStorage==2.3.1\",\"six==1.10.0\",\"unity-lens-photos==1.0\",\"WebOb==1.6.2\"],\"version\":\"2.7.13 (default, Jan 19 2017, 14:48:08) \\n[GCC 6.3.0 20170118]\"},\"ram\":32132,\"running_time\":37229,\"sleep_streak\":4,\"ssd\":[\"sda\"],\"started_ts\":1528940227,\"temp\":{\"thermal_zone0\":34.5,\"thermal_zone1\":35.0},\"uptime\":37248,\"user\":\"chrome-bot\"}",
            "task_id": "3e16f9f04070d611",
            "task_name": "Perf-Ubuntu17-GCC-Golo-GPU-QuadroP400-x86_64-Release-All-Valgrind_AbandonGpuContext_SK_CPU_LIMIT_SSE41",
            "version": "abcdoeraymeyouandme"
        },
        {
            "authenticated_as": "bot:somebot10-a9.fleet.example.com",
            "bot_id": "somebot10-a9",
            "deleted": false,
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["git", "git_cache", "vpython", "work"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-E3-1230_v5", "x86-64-avx2"]
                },
                {
                    "key": "cpu_governor",
                    "value": ["powersave"]
                },
                {
                    "key": "gpu",
                    "value": ["10de", "10de:1cb3", "10de:1cb3-384.59"]
                },
                {
                    "key": "id",
                    "value": ["somebot10-a9"]
                },
                {
                    "key": "inside_docker",
                    "value": ["0"]
                },
                {
                    "key": "kvm",
                    "value": ["1"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.ISO8859-1"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-standard-8"]
                },
                {
                    "key": "os",
                    "value": ["Linux", "Ubuntu", "Ubuntu-17.04"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                },
                {
                    "key": "ssd",
                    "value": ["1"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-nyc", "us-nyc-ellis", "us-nyc-ellis-a", "us-nyc-ellis-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-03T22:05:44.125130",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T12:44:05.804733",
            "quarantined": false,
            "state": "{\"audio\":[\"NVIDIA Corporation [10de]: Device [0fb9]\"],\"bot_group_cfg_version\":\"hash:04fbe8dc9d7c6b\",\"cost_usd_hour\":0.4369776801215278,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"/b/s\",\"disks\":{\"/\":{\"free_mb\":680777.4,\"size_mb\":741467.3},\"/boot\":{\"free_mb\":842.2,\"size_mb\":922.0}},\"env\":{\"PATH\":\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin\"},\"files\":{\"/usr/share/fonts/truetype/\":[\"Gargi\",\"Gubbi\",\"Nakula\",\"Navilu\",\"Sahadeva\",\"Sarai\",\"abyssinica\",\"ancient-scripts\",\"dejavu\",\"droid\",\"fonts-beng-extra\",\"fonts-deva-extra\",\"fonts-gujr-extra\",\"fonts-guru-extra\",\"fonts-japanese-gothic.ttf\",\"fonts-japanese-mincho.ttf\",\"fonts-kalapi\",\"fonts-orya-extra\",\"fonts-telu-extra\",\"freefont\",\"kacst\",\"kacst-one\",\"lao\",\"lato\",\"liberation\",\"lohit-assamese\",\"lohit-bengali\",\"lohit-devanagari\",\"lohit-gujarati\",\"lohit-kannada\",\"lohit-malayalam\",\"lohit-oriya\",\"lohit-punjabi\",\"lohit-tamil\",\"lohit-tamil-classical\",\"lohit-telugu\",\"malayalam-fonts\",\"msttcorefonts\",\"nanum\",\"noto\",\"openoffice\",\"padauk\",\"pagul\",\"samyak\",\"samyak-fonts\",\"sinhala\",\"takao-gothic\",\"tibetan-machine\",\"tlwg\",\"ttf-dejavu\",\"ttf-khmeros-core\",\"ubuntu-font-family\"]},\"gpu\":[\"Nvidia Quadro P400 384.59\"],\"hostname\":\"somebot10-a9.fleet.example.com\",\"ip\":\"192.168.216.20\",\"nb_files_in_temp\":15,\"pid\":2162,\"python\":{\"executable\":\"/usr/bin/python\",\"packages\":[\"adium-theme-ubuntu==0.3.4\",\"CherryPy==3.5.0\",\"cryptography==1.7.1\",\"enum34==1.1.6\",\"httplib2==0.9.2\",\"idna==2.2\",\"ipaddress==1.0.17\",\"keyring==10.3.1\",\"keyrings.alt==2.2\",\"ndg-httpsclient==0.4.2\",\"numpy==1.12.1\",\"Pillow==4.0.0\",\"psutil==5.0.1\",\"pyasn1==0.1.9\",\"pycrypto==2.6.1\",\"pycurl==7.43.0\",\"pygobject==3.22.0\",\"pyOpenSSL==16.2.0\",\"pyserial==3.2.1\",\"python-apt==1.4.0b2\",\"pyxdg==0.25\",\"PyYAML==3.12\",\"repoze.lru==0.6\",\"Routes==2.2\",\"scour==0.32\",\"SecretStorage==2.3.1\",\"six==1.10.0\",\"unity-lens-photos==1.0\",\"WebOb==1.6.2\"],\"version\":\"2.7.13 (default, Jan 19 2017, 14:48:08) \\n[GCC 6.3.0 20170118]\"},\"ram\":32132,\"running_time\":21192,\"sleep_streak\":10,\"ssd\":[\"sda\"],\"started_ts\":1528959051,\"temp\":{\"thermal_zone0\":30.5,\"thermal_zone1\":37.0},\"uptime\":21212,\"user\":\"chrome-bot\"}",
            "task_id": "",
            "version": "abcdoeraymeyouandme"
        },
        {
            "authenticated_as": "bot:somebot11-a9.fleet.example.com",
            "bot_id": "somebot11-a9",
            "deleted": false,
            "dimensions": [
                {
                    "key": "android_devices",
                    "value": ["3"]
                },
                {
                    "key": "caches",
                    "value": ["vpython"]
                },
                {
                    "key": "device_gms_core_version",
                    "value": ["10.9.32"]
                },
                {
                    "key": "device_os",
                    "value": ["O", "OPR6.170623.023"]
                },
                {
                    "key": "device_os_flavor",
                    "value": ["google"]
                },
                {
                    "key": "device_type",
                    "value": ["bullhead"]
                },
                {
                    "key": "id",
                    "value": ["somebot11-a9"]
                },
                {
                    "key": "os",
                    "value": ["Android"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.12"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-03T22:05:43.706260",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T12:44:21.392309",
            "quarantined": true,
            "state":"{\"audio\":null,\"bot_group_cfg_version\":\"hash:14d84b1c063e43\",\"cost_usd_hour\":0.42244482964409724,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"/b/swarming\",\"devices\":{\"Z01234567\":{\"battery\":{\"current\":null,\"health\":7,\"level\":3,\"power\":[\"AC\"],\"status\":3,\"temperature\":32,\"voltage\":3635},\"build\":{\"board.platform\":\"<missing>\",\"build.fingerprint\":\"google/bullhead/bullhead:8.0.0/OPR6.170623.023/4409485:userdebug/dev-keys\",\"build.id\":\"OPR6.170623.023\",\"build.product\":\"bullhead\",\"build.version.sdk\":\"26\",\"product.board\":\"bullhead\",\"product.cpu.abi\":\"arm64-v8a\",\"product.device\":\"bullhead\"},\"cpu\":{\"cur\":\"460800\",\"governor\":\"interactive\"},\"disk\":{\"cache\":{\"free_mb\":9840,\"size_mb\":10807.6},\"data\":{\"free_mb\":9840,\"size_mb\":10807.6},\"system\":{\"free_mb\":1109.7,\"size_mb\":3022.8}},\"imei\":\"redacted\",\"ip\":[],\"max_uid\":null,\"mem\":{},\"other_packages\":[],\"port_path\":\"1/86\",\"processes\":2,\"state\":\"low_battery\",\"temp\":{\"battery\":\"27.3\",\"bms\":29.5,\"pa_therm0\":32,\"pm8994_tz\":35.582,\"tsens_tz_sensor0\":34,\"tsens_tz_sensor1\":36,\"tsens_tz_sensor11\":36,\"tsens_tz_sensor12\":36,\"tsens_tz_sensor13\":37,\"tsens_tz_sensor14\":37,\"tsens_tz_sensor2\":35,\"tsens_tz_sensor3\":35,\"tsens_tz_sensor4\":34,\"tsens_tz_sensor5\":34,\"tsens_tz_sensor7\":35,\"tsens_tz_sensor9\":36},\"temp_skipped\":{\"tsens_tz_sensor10\":39},\"uptime\":180.59},\"89ABCDEF012\":{\"battery\":{\"current\":null,\"health\":2,\"level\":100,\"power\":[\"AC\"],\"status\":5,\"temperature\":295,\"voltage\":4335},\"build\":{\"board.platform\":\"<missing>\",\"build.fingerprint\":\"google/bullhead/bullhead:8.0.0/OPR6.170623.023/4409485:userdebug/dev-keys\",\"build.id\":\"OPR6.170623.023\",\"build.product\":\"bullhead\",\"build.version.sdk\":\"26\",\"product.board\":\"bullhead\",\"product.cpu.abi\":\"arm64-v8a\",\"product.device\":\"bullhead\"},\"cpu\":{\"cur\":\"460800\",\"governor\":\"interactive\"},\"disk\":{\"cache\":{\"free_mb\":9840,\"size_mb\":10807.6},\"data\":{\"free_mb\":9840,\"size_mb\":10807.6},\"system\":{\"free_mb\":1109.7,\"size_mb\":3022.8}},\"imei\":\"redacted\",\"ip\":[],\"max_uid\":null,\"mem\":{},\"other_packages\":[],\"port_path\":\"1/86\",\"processes\":2,\"state\":\"too_hot\",\"temp\":{\"battery\":\"57.3\",\"bms\":29.5,\"pa_therm0\":32,\"pm8994_tz\":35.582,\"tsens_tz_sensor0\":34,\"tsens_tz_sensor1\":36,\"tsens_tz_sensor11\":36,\"tsens_tz_sensor12\":36,\"tsens_tz_sensor13\":37,\"tsens_tz_sensor14\":37,\"tsens_tz_sensor2\":35,\"tsens_tz_sensor3\":35,\"tsens_tz_sensor4\":34,\"tsens_tz_sensor5\":34,\"tsens_tz_sensor7\":35,\"tsens_tz_sensor9\":36},\"temp_skipped\":{\"tsens_tz_sensor10\":39},\"uptime\":180.59},\"3456789ABC\":{\"battery\":{\"current\":null,\"health\":2,\"level\":100,\"power\":[\"AC\"],\"status\":5,\"temperature\":295,\"voltage\":4335},\"build\":{\"board.platform\":\"<missing>\",\"build.fingerprint\":\"google/bullhead/bullhead:8.0.0/OPR6.170623.023/4409485:userdebug/dev-keys\",\"build.id\":\"OPR6.170623.023\",\"build.product\":\"bullhead\",\"build.version.sdk\":\"26\",\"product.board\":\"bullhead\",\"product.cpu.abi\":\"arm64-v8a\",\"product.device\":\"bullhead\"},\"cpu\":{\"cur\":\"460800\",\"governor\":\"interactive\"},\"disk\":{\"cache\":{\"free_mb\":9840,\"size_mb\":10807.6},\"data\":{\"free_mb\":9840,\"size_mb\":10807.6},\"system\":{\"free_mb\":1109.7,\"size_mb\":3022.8}},\"imei\":\"redacted\",\"ip\":[],\"max_uid\":null,\"mem\":{},\"other_packages\":[],\"port_path\":\"1/86\",\"processes\":2,\"state\":\"available\",\"temp\":{\"battery\":\"28.3\",\"bms\":29.5,\"pa_therm0\":32,\"pm8994_tz\":35.582,\"tsens_tz_sensor0\":34,\"tsens_tz_sensor1\":36,\"tsens_tz_sensor11\":36,\"tsens_tz_sensor12\":36,\"tsens_tz_sensor13\":37,\"tsens_tz_sensor14\":37,\"tsens_tz_sensor2\":35,\"tsens_tz_sensor3\":35,\"tsens_tz_sensor4\":34,\"tsens_tz_sensor5\":34,\"tsens_tz_sensor7\":35,\"tsens_tz_sensor9\":36},\"temp_skipped\":{\"tsens_tz_sensor10\":39},\"uptime\":180.59}},\"disks\":{\"/b\":{\"free_mb\":413728.6,\"size_mb\":735012.1}},\"env\":{\"PATH\":\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games\"},\"gpu\":null,\"host_dimensions\":{\"caches\":[\"vpython\"],\"cores\":[\"8\"],\"cpu\":[\"x86\",\"x86-64\",\"x86-64-E3-1230_v5\",\"x86-64-avx2\"],\"gpu\":[\"none\"],\"id\":[\"somebot11-a9\"],\"kvm\":[\"0\"],\"machine_type\":[\"n1-standard-8\"],\"os\":[\"Linux\",\"Ubuntu\",\"Ubuntu-16.04\"],\"python\":[\"2.7.12\"],\"ssd\":[\"1\"]},\"hostname\":\"somebot11-a9\",\"ip\":\"172.17.0.4\",\"nb_files_in_temp\":1,\"pid\":46,\"python\":{\"executable\":\"/usr/bin/python\",\"packages\":null,\"version\":\"2.7.12 (default, Nov 19 2016, 06:48:10) \\n[GCC 5.4.0 20160609]\"},\"quarantined\":\"No available devices.\",\"ram\":32136,\"running_time\":8010,\"sleep_streak\":9,\"ssd\":[\"sda\"],\"started_ts\":1528993814,\"temp\":{\"thermal_zone0\":35},\"uptime\":39989,\"user\":\"chrome-bot\"}",
            "task_id": "",
            "version": "abcdoeraymeyouandme",
        },
        {
            "authenticated_as": "bot:somebot12-a9.fleet.example.com",
            "bot_id": "somebot12-a9",
            "deleted": false,
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["swarming_module_cache_vpython"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-i7-7700"]
                },
                {
                    "key": "gpu",
                    "value": ["8086", "8086:5912", "8086:5912-23.20.16.4877"]
                },
                {
                    "key": "id",
                    "value": ["build164-a9"]
                },
                {
                    "key": "integrity",
                    "value": ["high"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.cp1252"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-highcpu-8"]
                },
                {
                    "key": "os",
                    "value": ["Windows", "Windows-10", "Windows-10-16299.431"]
                },
                {
                    "key": "pool",
                    "value": ["Chrome-GPU"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["3637-1468930"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-mke", "us-mke-beer", "us-mke-beer-a", "us-mke-beer-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-03T22:05:44.179170",
            "is_dead": true,
            "last_seen_ts": "2018-06-07T12:44:12.941693",
            "quarantined": false,
            "state": "{\"audio\":[\"Realtek Audio\",\"Intel(R) Display Audio\"],\"bot_group_cfg_version\":\"hash:2862646523b83b\",\"cost_usd_hour\":0.6440780164930555,\"cpu_name\":\"Intel(R) Core(TM) i7-7700 CPU @ 3.60GHz\",\"cwd\":\"C:\\\\b\\\\s\",\"cygwin\":[false],\"disks\":{\"c:\\\\\":{\"free_mb\":370037.1,\"size_mb\":476478.1}},\"env\":{\"PATH\":\"C:\\\\WINDOWS\\\\system32;C:\\\\WINDOWS;C:\\\\WINDOWS\\\\System32\\\\Wbem;C:\\\\WINDOWS\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\;C:\\\\Program Files (x86)\\\\Windows Kits\\\\8.1\\\\Windows Performance Toolkit\\\\;C:\\\\Tools;C:\\\\b\\\\depot_tools;C:\\\\CMake\\\\bin;C:\\\\Program Files\\\\Puppet Labs\\\\Puppet\\\\bin;C:\\\\Users\\\\chrome-bot\\\\AppData\\\\Local\\\\Microsoft\\\\WindowsApps;\"},\"files\":{\"c:\\\\Users\\\\chrome-bot\\\\ntuser.dat\":9175040},\"gpu\":[\"Intel Kaby Lake HD Graphics 630 23.20.16.4877\"],\"hostname\":\"somebot12-a9.fleet.example.com\",\"ip\":\"192.168.216.173\",\"nb_files_in_temp\":1,\"pid\":6860,\"python\":{\"executable\":\"c:\\\\infra-system\\\\bin\\\\python.exe\",\"packages\":null,\"version\":\"2.7.13 (v2.7.13:a06454b1afa1, Dec 17 2016, 20:53:40) [MSC v.1500 64 bit (AMD64)]\"},\"ram\":16248,\"running_time\":5027,\"sleep_streak\":9,\"ssd\":[],\"started_ts\":1529062777,\"top_windows\":[],\"uptime\":5077,\"user\":\"chrome-bot\"}",
            "task_id": "3e17182091d7ae11",
            "task_name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-ASAN",
            "version": "old_version123456"
        },
        {
            "authenticated_as": "bot:somebot13-a9.fleet.example.com",
            "bot_id": "somebot13-a9",
            "deleted": false,
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["git", "git_cache", "vpython", "work"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-E3-1230_v5", "x86-64-avx2"]
                },
                {
                    "key": "cpu_governor",
                    "value": ["powersave"]
                },
                {
                    "key": "gpu",
                    "value": ["10de", "10de:1cb3", "10de:1cb3-384.59"]
                },
                {
                    "key": "id",
                    "value": ["somebot13-a9"]
                },
                {
                    "key": "inside_docker",
                    "value": ["0"]
                },
                {
                    "key": "kvm",
                    "value": ["1"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.ISO8859-1"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-standard-8"]
                },
                {
                    "key": "os",
                    "value": ["Linux", "Ubuntu", "Ubuntu-17.04"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                },
                {
                    "key": "ssd",
                    "value": ["1"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-nyc", "us-nyc-ellis", "us-nyc-ellis-a", "us-nyc-ellis-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-03T22:05:44.077550",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T12:44:24.889729",
            "quarantined": false,
            "state": "{\"audio\":[\"NVIDIA Corporation [10de]: Device [0fb9]\"],\"bot_group_cfg_version\":\"hash:04fbe8dc9d7c6b\",\"cost_usd_hour\":0.43697737630208333,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"/b/s\",\"disks\":{\"/\":{\"free_mb\":680616.0,\"size_mb\":741467.3},\"/boot\":{\"free_mb\":842.2,\"size_mb\":922.0}},\"env\":{\"PATH\":\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin\"},\"files\":{\"/usr/share/fonts/truetype/\":[\"Gargi\",\"Gubbi\",\"Nakula\",\"Navilu\",\"Sahadeva\",\"Sarai\",\"abyssinica\",\"ancient-scripts\",\"dejavu\",\"droid\",\"fonts-beng-extra\",\"fonts-deva-extra\",\"fonts-gujr-extra\",\"fonts-guru-extra\",\"fonts-japanese-gothic.ttf\",\"fonts-japanese-mincho.ttf\",\"fonts-kalapi\",\"fonts-orya-extra\",\"fonts-telu-extra\",\"freefont\",\"kacst\",\"kacst-one\",\"lao\",\"lato\",\"liberation\",\"lohit-assamese\",\"lohit-bengali\",\"lohit-devanagari\",\"lohit-gujarati\",\"lohit-kannada\",\"lohit-malayalam\",\"lohit-oriya\",\"lohit-punjabi\",\"lohit-tamil\",\"lohit-tamil-classical\",\"lohit-telugu\",\"malayalam-fonts\",\"msttcorefonts\",\"nanum\",\"noto\",\"openoffice\",\"padauk\",\"pagul\",\"samyak\",\"samyak-fonts\",\"sinhala\",\"takao-gothic\",\"tibetan-machine\",\"tlwg\",\"ttf-dejavu\",\"ttf-khmeros-core\",\"ubuntu-font-family\"]},\"gpu\":[\"Nvidia Quadro P400 384.59\"],\"hostname\":\"somebot13-a9.fleet.example.com\",\"ip\":\"192.168.216.23\",\"nb_files_in_temp\":14,\"pid\":2113,\"python\":{\"executable\":\"/usr/bin/python\",\"packages\":[\"adium-theme-ubuntu==0.3.4\",\"CherryPy==3.5.0\",\"cryptography==1.7.1\",\"enum34==1.1.6\",\"httplib2==0.9.2\",\"idna==2.2\",\"ipaddress==1.0.17\",\"keyring==10.3.1\",\"keyrings.alt==2.2\",\"ndg-httpsclient==0.4.2\",\"numpy==1.12.1\",\"Pillow==4.0.0\",\"psutil==5.0.1\",\"pyasn1==0.1.9\",\"pycrypto==2.6.1\",\"pycurl==7.43.0\",\"pygobject==3.22.0\",\"pyOpenSSL==16.2.0\",\"pyserial==3.2.1\",\"python-apt==1.4.0b2\",\"pyxdg==0.25\",\"PyYAML==3.12\",\"repoze.lru==0.6\",\"Routes==2.2\",\"scour==0.32\",\"SecretStorage==2.3.1\",\"six==1.10.0\",\"unity-lens-photos==1.0\",\"WebOb==1.6.2\"],\"version\":\"2.7.13 (default, Jan 19 2017, 14:48:08) \\n[GCC 6.3.0 20170118]\"},\"ram\":32132,\"running_time\":12234,\"sleep_streak\":0,\"ssd\":[\"sda\"],\"started_ts\":1528968031,\"temp\":{\"thermal_zone0\":29.5,\"thermal_zone1\":41.0},\"uptime\":12253,\"user\":\"chrome-bot\"}",
            "task_id": "",
            "version": "old_version123456"
        },
        {
            "authenticated_as": "bot:somebot13-a2.fleet.example.com",
            "bot_id": "somebot13-a2",
            "deleted": false,
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["git", "git_cache", "vpython", "work"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-E3-1230_v5", "x86-64-avx2"]
                },
                {
                    "key": "cpu_governor",
                    "value": ["powersave"]
                },
                {
                    "key": "gpu",
                    "value": ["10de", "10de:1cb3", "10de:1cb3-384.59"]
                },
                {
                    "key": "id",
                    "value": ["somebot13-a2"]
                },
                {
                    "key": "inside_docker",
                    "value": ["0"]
                },
                {
                    "key": "kvm",
                    "value": ["1"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.ISO8859-1"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-standard-8"]
                },
                {
                    "key": "os",
                    "value": ["Linux", "Ubuntu", "Ubuntu-17.04"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                },
                {
                    "key": "ssd",
                    "value": ["1"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-nyc", "us-nyc-ellis", "us-nyc-ellis-a", "us-nyc-ellis-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-03T22:05:43.791120",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T08:43:21.247109",
            "maintenance_msg": "Need to re-format the hard drive.",
            "quarantined": false,
            "state": "{\"audio\":[\"NVIDIA Corporation [10de]: Device [0fb9]\"],\"bot_group_cfg_version\":\"hash:04fbe8dc9d7c6b\",\"cost_usd_hour\":0.43697845594618057,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"/b/s\",\"disks\":{\"/\":{\"free_mb\":680839.4,\"size_mb\":741467.3},\"/boot\":{\"free_mb\":842.2,\"size_mb\":922.0}},\"env\":{\"PATH\":\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin\"},\"files\":{\"/usr/share/fonts/truetype/\":[\"Gargi\",\"Gubbi\",\"Nakula\",\"Navilu\",\"Sahadeva\",\"Sarai\",\"abyssinica\",\"ancient-scripts\",\"dejavu\",\"droid\",\"fonts-beng-extra\",\"fonts-deva-extra\",\"fonts-gujr-extra\",\"fonts-guru-extra\",\"fonts-japanese-gothic.ttf\",\"fonts-japanese-mincho.ttf\",\"fonts-kalapi\",\"fonts-orya-extra\",\"fonts-telu-extra\",\"freefont\",\"kacst\",\"kacst-one\",\"lao\",\"lato\",\"liberation\",\"lohit-assamese\",\"lohit-bengali\",\"lohit-devanagari\",\"lohit-gujarati\",\"lohit-kannada\",\"lohit-malayalam\",\"lohit-oriya\",\"lohit-punjabi\",\"lohit-tamil\",\"lohit-tamil-classical\",\"lohit-telugu\",\"malayalam-fonts\",\"msttcorefonts\",\"nanum\",\"noto\",\"openoffice\",\"padauk\",\"pagul\",\"samyak\",\"samyak-fonts\",\"sinhala\",\"takao-gothic\",\"tibetan-machine\",\"tlwg\",\"ttf-dejavu\",\"ttf-khmeros-core\",\"ubuntu-font-family\"]},\"gpu\":[\"Nvidia Quadro P400 384.59\"],\"hostname\":\"somebot13-a2.fleet.example.com\",\"ip\":\"192.168.216.24\",\"nb_files_in_temp\":15,\"pid\":2166,\"python\":{\"executable\":\"/usr/bin/python\",\"packages\":[\"adium-theme-ubuntu==0.3.4\",\"CherryPy==3.5.0\",\"cryptography==1.7.1\",\"enum34==1.1.6\",\"httplib2==0.9.2\",\"idna==2.2\",\"ipaddress==1.0.17\",\"keyring==10.3.1\",\"keyrings.alt==2.2\",\"ndg-httpsclient==0.4.2\",\"numpy==1.12.1\",\"Pillow==4.0.0\",\"psutil==5.0.1\",\"pyasn1==0.1.9\",\"pycrypto==2.6.1\",\"pycurl==7.43.0\",\"pygobject==3.22.0\",\"pyOpenSSL==16.2.0\",\"pyserial==3.2.1\",\"python-apt==1.4.0b2\",\"pyxdg==0.25\",\"PyYAML==3.12\",\"repoze.lru==0.6\",\"Routes==2.2\",\"scour==0.32\",\"SecretStorage==2.3.1\",\"six==1.10.0\",\"unity-lens-photos==1.0\",\"WebOb==1.6.2\"],\"version\":\"2.7.13 (default, Jan 19 2017, 14:48:08) \\n[GCC 6.3.0 20170118]\"},\"ram\":32132,\"running_time\":50872,\"sleep_streak\":0,\"ssd\":[\"sda\"],\"started_ts\":1528929329,\"temp\":{\"thermal_zone0\":28.5,\"thermal_zone1\":42.0},\"uptime\":50910,\"user\":\"chrome-bot\"}",
            "task_id": "",
            "version": "abcdoeraymeyouandme"
        },
        {
            "authenticated_as": "bot:somebot15-a9.fleet.example.com",
            "bot_id": "somebot15-a9",
            "deleted": false,
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["git", "git_cache", "vpython", "work"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-E3-1230_v5", "x86-64-avx2"]
                },
                {
                    "key": "cpu_governor",
                    "value": ["powersave"]
                },
                {
                    "key": "gpu",
                    "value": ["10de", "10de:1cb3", "10de:1cb3-384.59"]
                },
                {
                    "key": "id",
                    "value": ["somebot15-a9"]
                },
                {
                    "key": "inside_docker",
                    "value": ["0"]
                },
                {
                    "key": "kvm",
                    "value": ["1"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.ISO8859-1"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-standard-8"]
                },
                {
                    "key": "os",
                    "value": ["Linux", "Ubuntu", "Ubuntu-17.04"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                },
                {
                    "key": "ssd",
                    "value": ["1"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-nyc", "us-nyc-ellis", "us-nyc-ellis-a", "us-nyc-ellis-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-03T22:05:44.398660",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T12:44:11.512503",
            "quarantined": false,
            "state": "{\"audio\":[\"NVIDIA Corporation [10de]: Device [0fb9]\"],\"bot_group_cfg_version\":\"hash:04fbe8dc9d7c6b\",\"cost_usd_hour\":0.43697778862847225,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"/b/s\",\"disks\":{\"/\":{\"free_mb\":680777.1,\"size_mb\":741467.3},\"/boot\":{\"free_mb\":842.2,\"size_mb\":922.0}},\"env\":{\"PATH\":\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin\"},\"files\":{\"/usr/share/fonts/truetype/\":[\"Gargi\",\"Gubbi\",\"Nakula\",\"Navilu\",\"Sahadeva\",\"Sarai\",\"abyssinica\",\"ancient-scripts\",\"dejavu\",\"droid\",\"fonts-beng-extra\",\"fonts-deva-extra\",\"fonts-gujr-extra\",\"fonts-guru-extra\",\"fonts-japanese-gothic.ttf\",\"fonts-japanese-mincho.ttf\",\"fonts-kalapi\",\"fonts-orya-extra\",\"fonts-telu-extra\",\"freefont\",\"kacst\",\"kacst-one\",\"lao\",\"lato\",\"liberation\",\"lohit-assamese\",\"lohit-bengali\",\"lohit-devanagari\",\"lohit-gujarati\",\"lohit-kannada\",\"lohit-malayalam\",\"lohit-oriya\",\"lohit-punjabi\",\"lohit-tamil\",\"lohit-tamil-classical\",\"lohit-telugu\",\"malayalam-fonts\",\"msttcorefonts\",\"nanum\",\"noto\",\"openoffice\",\"padauk\",\"pagul\",\"samyak\",\"samyak-fonts\",\"sinhala\",\"takao-gothic\",\"tibetan-machine\",\"tlwg\",\"ttf-dejavu\",\"ttf-khmeros-core\",\"ubuntu-font-family\"]},\"gpu\":[\"Nvidia Quadro P400 384.59\"],\"hostname\":\"somebot15-a9.fleet.example.com\",\"ip\":\"192.168.216.25\",\"nb_files_in_temp\":15,\"pid\":2329,\"python\":{\"executable\":\"/usr/bin/python\",\"packages\":[\"adium-theme-ubuntu==0.3.4\",\"CherryPy==3.5.0\",\"cryptography==1.7.1\",\"enum34==1.1.6\",\"httplib2==0.9.2\",\"idna==2.2\",\"ipaddress==1.0.17\",\"keyring==10.3.1\",\"keyrings.alt==2.2\",\"ndg-httpsclient==0.4.2\",\"numpy==1.12.1\",\"Pillow==4.0.0\",\"psutil==5.0.1\",\"pyasn1==0.1.9\",\"pycrypto==2.6.1\",\"pycurl==7.43.0\",\"pygobject==3.22.0\",\"pyOpenSSL==16.2.0\",\"pyserial==3.2.1\",\"python-apt==1.4.0b2\",\"pyxdg==0.25\",\"PyYAML==3.12\",\"repoze.lru==0.6\",\"Routes==2.2\",\"scour==0.32\",\"SecretStorage==2.3.1\",\"six==1.10.0\",\"unity-lens-photos==1.0\",\"WebOb==1.6.2\"],\"version\":\"2.7.13 (default, Jan 19 2017, 14:48:08) \\n[GCC 6.3.0 20170118]\"},\"ram\":32132,\"running_time\":12846,\"sleep_streak\":6,\"ssd\":[\"sda\"],\"started_ts\":1528967405,\"temp\":{\"thermal_zone0\":30.0,\"thermal_zone1\":39.0},\"uptime\":12889,\"user\":\"chrome-bot\"}",
            "task_id": "",
            "version": "abcdoeraymeyouandme"
        },
        {
            "authenticated_as": "bot:somebot16-a9.fleet.example.com",
            "bot_id": "somebot16-a9",
            "deleted": false,
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["vpython"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-E3-1230_v5"]
                },
                {
                    "key": "gpu",
                    "value": ["10de", "10de:1cb3", "10de:1cb3-23.21.13.9103"]
                },
                {
                    "key": "id",
                    "value": ["somebot16-a9"]
                },
                {
                    "key": "integrity",
                    "value": ["high"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.cp1252"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-standard-8"]
                },
                {
                    "key": "os",
                    "value": ["Windows", "Windows-10", "Windows-10-16299.309"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-nyc", "us-nyc-ellis", "us-nyc-ellis-a", "us-nyc-ellis-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-02T23:12:16.365500",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T12:44:04.195565",
            "quarantined": false,
            "state": "{\"audio\":[\"NVIDIA High Definition Audio\"],\"bot_group_cfg_version\":\"hash:04fbe8dc9d7c6b\",\"cost_usd_hour\":0.7554625108506944,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"C:\\\\b\\\\s\",\"cygwin\":[false],\"disks\":{\"c:\\\\\":{\"free_mb\":653663.1,\"size_mb\":762633.6}},\"env\":{\"PATH\":\"C:\\\\WINDOWS\\\\system32;C:\\\\WINDOWS;C:\\\\WINDOWS\\\\System32\\\\Wbem;C:\\\\WINDOWS\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\;C:\\\\Program Files (x86)\\\\Windows Kits\\\\8.1\\\\Windows Performance Toolkit\\\\;C:\\\\Tools;C:\\\\b\\\\depot_tools;C:\\\\CMake\\\\bin;C:\\\\Program Files\\\\Puppet Labs\\\\Puppet\\\\bin;C:\\\\Users\\\\chrome-bot\\\\AppData\\\\Local\\\\Microsoft\\\\WindowsApps;\"},\"files\":{\"C:/Users/chrome-bot\\\\ntuser.dat\":5767168},\"gpu\":[\"Nvidia Quadro P400 23.21.13.9103\"],\"hostname\":\"somebot16-a9.fleet.example.com\",\"ip\":\"192.168.216.26\",\"nb_files_in_temp\":3,\"pid\":7976,\"python\":{\"executable\":\"c:\\\\infra-system\\\\bin\\\\python.exe\",\"packages\":null,\"version\":\"2.7.13 (v2.7.13:a06454b1afa1, Dec 17 2016, 20:53:40) [MSC v.1500 64 bit (AMD64)]\"},\"ram\":32726,\"running_time\":38494,\"sleep_streak\":4,\"ssd\":[],\"started_ts\":1528941707,\"top_windows\":[],\"uptime\":38514,\"user\":\"chrome-bot\"}",
            "task_id": "3e1723e23fd70811",
            "task_name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All",
            "version": "abcdoeraymeyouandme"
        },
        {
            "authenticated_as": "bot:somebot17-a9.fleet.example.com",
            "bot_id": "somebot17-a9",
            "deleted": false,
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["vpython"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-E3-1230_v5"]
                },
                {
                    "key": "gpu",
                    "value": ["10de", "10de:1cb3", "10de:1cb3-23.21.13.9103"]
                },
                {
                    "key": "id",
                    "value": ["somebot17-a9"]
                },
                {
                    "key": "integrity",
                    "value": ["high"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.cp1252"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-standard-8"]
                },
                {
                    "key": "os",
                    "value": ["Windows", "Windows-10", "Windows-10-16299.309"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-nyc", "us-nyc-ellis", "us-nyc-ellis-a", "us-nyc-ellis-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-02T23:12:16.969860",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T12:43:59.436254",
            "quarantined": false,
            "state": "{\"audio\":[\"NVIDIA High Definition Audio\"],\"bot_group_cfg_version\":\"hash:04fbe8dc9d7c6b\",\"cost_usd_hour\":0.7578841254340277,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"C:\\\\b\\\\s\",\"cygwin\":[false],\"disks\":{\"c:\\\\\":{\"free_mb\":698305.5,\"size_mb\":762633.6}},\"env\":{\"PATH\":\"C:\\\\WINDOWS\\\\system32;C:\\\\WINDOWS;C:\\\\WINDOWS\\\\System32\\\\Wbem;C:\\\\WINDOWS\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\;C:\\\\Program Files (x86)\\\\Windows Kits\\\\8.1\\\\Windows Performance Toolkit\\\\;C:\\\\Tools;C:\\\\b\\\\depot_tools;C:\\\\CMake\\\\bin;C:\\\\Program Files\\\\Puppet Labs\\\\Puppet\\\\bin;C:\\\\Users\\\\chrome-bot\\\\AppData\\\\Local\\\\Microsoft\\\\WindowsApps;\"},\"files\":{\"C:/Users/chrome-bot\\\\ntuser.dat\":5767168},\"gpu\":[\"Nvidia Quadro P400 23.21.13.9103\"],\"hostname\":\"somebot17-a9.fleet.example.com\",\"ip\":\"192.168.216.27\",\"nb_files_in_temp\":2,\"pid\":8084,\"python\":{\"executable\":\"c:\\\\infra-system\\\\bin\\\\python.exe\",\"packages\":null,\"version\":\"2.7.13 (v2.7.13:a06454b1afa1, Dec 17 2016, 20:53:40) [MSC v.1500 64 bit (AMD64)]\"},\"ram\":32726,\"running_time\":874,\"sleep_streak\":5,\"ssd\":[],\"started_ts\":1528977403,\"top_windows\":[],\"uptime\":894,\"user\":\"chrome-bot\"}",
            "task_id": "3e170673db0c2511",
            "task_name": "Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Release-All-Vulkan_Skpbench_DDLTotal_9x9",
            "version": "abcdoeraymeyouandme"
        },
        {
            "authenticated_as": "bot:somebot18-a9.fleet.example.com",
            "bot_id": "somebot18-a9",
            "deleted": false,
            "lease_id": "22596363b3de40b06f981fb85d82312e8c0ed511",
            "lease_expiration_ts": "2019-10-24T16:01:02",
            "dimensions": [
                {
                    "key": "caches",
                    "value": ["vpython"]
                },
                {
                    "key": "cores",
                    "value": ["8"]
                },
                {
                    "key": "cpu",
                    "value": ["x86", "x86-64", "x86-64-E3-1230_v5"]
                },
                {
                    "key": "gpu",
                    "value": ["10de", "10de:1cb3", "10de:1cb3-23.21.13.9103"]
                },
                {
                    "key": "id",
                    "value": ["somebot18-a9"]
                },
                {
                    "key": "integrity",
                    "value": ["high"]
                },
                {
                    "key": "locale",
                    "value": ["en_US.cp1252"]
                },
                {
                    "key": "machine_type",
                    "value": ["n1-standard-8"]
                },
                {
                    "key": "os",
                    "value": ["Windows", "Windows-10", "Windows-10-16299.431"]
                },
                {
                    "key": "pool",
                    "value": ["Skia"]
                },
                {
                    "key": "python",
                    "value": ["2.7.13"]
                },
                {
                    "key": "server_version",
                    "value": ["9001-3541841"]
                },
                {
                    "key": "zone",
                    "value": ["us", "us-nyc", "us-nyc-ellis", "us-nyc-ellis-a", "us-nyc-ellis-a-9"]
                }
            ],
            "external_ip": "93.184.216.34",
            "first_seen_ts": "2017-08-02T23:12:16.539590",
            "is_dead": false,
            "last_seen_ts": "2018-06-14T12:44:13.291731",
            "quarantined": false,
            "state": "{\"audio\":[\"NVIDIA High Definition Audio\"],\"bot_group_cfg_version\":\"hash:04fbe8dc9d7c6b\",\"cost_usd_hour\":0.757346923828125,\"cpu_name\":\"Intel(R) Xeon(R) CPU E3-1230 v5 @ 3.40GHz\",\"cwd\":\"C:\\\\b\\\\s\",\"cygwin\":[false],\"disks\":{\"c:\\\\\":{\"free_mb\":695446.4,\"size_mb\":762633.6}},\"env\":{\"PATH\":\"C:\\\\WINDOWS\\\\system32;C:\\\\WINDOWS;C:\\\\WINDOWS\\\\System32\\\\Wbem;C:\\\\WINDOWS\\\\System32\\\\WindowsPowerShell\\\\v1.0\\\\;C:\\\\Program Files (x86)\\\\Windows Kits\\\\8.1\\\\Windows Performance Toolkit\\\\;C:\\\\Tools;C:\\\\b\\\\depot_tools;C:\\\\CMake\\\\bin;C:\\\\Program Files\\\\Puppet Labs\\\\Puppet\\\\bin;C:\\\\Users\\\\chrome-bot\\\\AppData\\\\Local\\\\Microsoft\\\\WindowsApps;\"},\"files\":{\"C:/Users/chrome-bot\\\\ntuser.dat\":5767168},\"gpu\":[\"Nvidia Quadro P400 23.21.13.9103\"],\"hostname\":\"somebot18-a9.fleet.example.com\",\"ip\":\"192.168.216.28\",\"nb_files_in_temp\":2,\"pid\":5464,\"python\":{\"executable\":\"c:\\\\infra-system\\\\bin\\\\python.exe\",\"packages\":null,\"version\":\"2.7.13 (v2.7.13:a06454b1afa1, Dec 17 2016, 20:53:40) [MSC v.1500 64 bit (AMD64)]\"},\"ram\":32726,\"running_time\":25281,\"sleep_streak\":447,\"ssd\":[],\"started_ts\":1528954972,\"top_windows\":[],\"uptime\":25308,\"user\":\"chrome-bot\"}",
            "task_id": "",
            "version": "abcdoeraymeyouandme"
        }
    ],
    "now": "2018-06-14T12:44:25.001011"
};
// end s10

export const fleetDimensions = {
 "ts": "2018-06-14T12:44:19.551385",
 "bots_dimensions": [
  {
   "value": ["1", "2", "3", "4", "5", "6", "7"],
   "key": "android_devices"
  },
  {
   "value": ["1", "16", "2", "24", "32", "4", "8"],
   "key": "cores"
  },
  {
   "value": ["arm", "arm-32", "armv7l", "armv7l-32", "x86", "x86-32", "x86-64", "x86-64-avx2"],
   "key": "cpu"
  },
  {
   "value": ["iPad4,1"],
   "key": "device"
  },
  {
   "value": ["J", "JSS15J", "K", "KOT49H", "KTU84P", "KTU84U", "L", "LMY48I", "LMY49K.LZC89", "LRW77", "LRX22H", "M", "MDB08O", "MMB29K", "MMB29Q", "MOB30K", "MRA58K", "MRA59F"],
   "key": "device_os"
  },
  {
   "value": ["bullhead", "flo", "flounder", "foster", "fugu", "gce_x86", "grouper", "hammerhead", "m0", "mako", "manta", "shamu", "sprout"],
   "key": "device_type"
  },
  {
   "value": ["[Errno 110] Connection timed out", "[Errno 111] Connection refused"],
   "key": "error"
  },
  {
   "value": ["1002", "1002:6779", "1002:6821", "1002:683d", "1002:9830", "102b", "102b:0522", "102b:0532", "102b:0534", "10de", "10de:08a4", "10de:0fe9", "10de:104a", "10de:11c0", "10de:1244", "10de:1401", "8086", "8086:0412", "8086:041a", "8086:0a2e", "8086:0d26", "8086:22b1", "MaliT86x", "none"],
   "key": "gpu"
  },
  {
   "value": ["0", "1"],
   "key": "hidpi"
  },
  {
   "value": ["n1-highcpu-16", "n1-highcpu-2", "n1-highcpu-4", "n1-highcpu-8", "n1-highmem-2", "n1-highmem-32", "n1-highmem-4", "n1-standard-1", "n1-standard-16", "n1-standard-2", "n1-standard-4", "n1-standard-8"],
   "key": "machine_type"
  },
  {
   "value": ["Android", "Linux", "Mac", "Mac-10.10", "Mac-10.11", "Mac-10.9", "Ubuntu", "Ubuntu-12.04", "Ubuntu-14.04", "Windows", "Windows-10-10240", "Windows-10-10586", "Windows-2008ServerR2-SP1", "Windows-2012ServerR2-SP0", "Windows-7-SP1", "Windows-8.1-SP0", "iOS-9.2"],
   "key": "os"
  },
  {
   "value": ["AndroidBuilder", "CT", "Chrome", "Chrome-perf", "LinuxBuilder", "QO", "Skia", "SkiaCT", "SkiaTriggers", "Swarmbucket-Testing", "V8-AVX2", "default"],
   "key": "pool"
  },
  {
   "value": ["1"],
   "key": "quarantined"
  },
  {
   "value": ["5.1.1", "6.3", "7.0", "7.3", "7.3.1"],
   "key": "xcode_version"
  },
  {
   "value": ["us", "us-central1", "us-central1-a", "us-central1-b", "us-central1-c"],
   "key": "zone"
  }
 ],
 "kind": "swarming#botsItem",
 "etag": "\"d6G6dOeYK-vHGD-PwSbeOsCnwa8/Bc6D9yOV3qWztuAExH_BbC_5oNM\""
};

let caches = new Array(400);
caches.fill('builder_1f179f0f1635560dacd58f0e40b3454e2bb4599d6850bdacacb50f08879998e7_v2');
caches = caches.map((b) => b += ('_' + Math.random() * 100));

// For testing https://crbug.com/927532
export const hardToSortDimensions = {
 "ts": "2018-06-14T12:44:19.551385",
 "bots_dimensions": [
  {
   "value": caches,
   "key": "caches"
  }],
};