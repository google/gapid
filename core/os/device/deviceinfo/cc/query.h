/*
 * Copyright (C) 2017 Google Inc.
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

#ifndef DEVICEINFO_QUERY_H
#define DEVICEINFO_QUERY_H

#include "core/os/device/device.pb.h"

namespace query {

// getDeviceInstance returns the device::Instance proto message for the
// current device. It must be freed with delete.
device::Instance* getDeviceInstance(void* platform_data);

// The functions below are used by getDeviceInstance(), and are implemented
// in the target-dependent sub-directories.

bool createContext(void* platform_data);
const char* contextError();
void destroyContext();

// The functions below require a context to be created.

int numABIs();
void abi(int idx, device::ABI* abi);
device::ABI* currentABI();
device::MemoryLayout* currentMemoryLayout();

const char* hardwareName();

const char* cpuName();
const char* cpuVendor();
device::Architecture cpuArchitecture();
int cpuNumCores();

const char* gpuName();
const char* gpuVendor();

const char* instanceName();

void glDriver(device::OpenGLDriver*);

device::OSKind osKind();
const char* osName();
const char* osBuild();
int osMajor();
int osMinor();
int osPoint();

}  // namespace query

#endif  // DEVICEINFO_QUERY_H
