/*
 * Copyright (C) 2019 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Perfetto build flags used by the GAPID build.

#ifndef GEN_BUILD_CONFIG_PERFETTO_BUILD_FLAGS_H_
#define GEN_BUILD_CONFIG_PERFETTO_BUILD_FLAGS_H_

#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_ANDROID_BUILD() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_CHROMIUM_BUILD() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_STANDALONE_BUILD() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_START_DAEMONS() (1)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_IPC() (1)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_WATCHDOG() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_COMPONENT_BUILD() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_FORCE_DLOG_ON() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_FORCE_DLOG_OFF() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_VERSION_GEN() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_TP_LINENOISE() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_TP_JSON() (0)
#define PERFETTO_BUILDFLAG_DEFINE_PERFETTO_TP_PERCENTILE() (1)

#endif  // GEN_BUILD_CONFIG_PERFETTO_BUILD_FLAGS_H_
