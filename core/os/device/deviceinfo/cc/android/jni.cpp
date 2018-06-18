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

#include "../instance.h"

#include <jni.h>

extern "C" {

jbyteArray Java_com_google_android_gapid_DeviceInfoService_getDeviceInfo(
    JNIEnv* env) {
  JavaVM* vm = nullptr;
  env->GetJavaVM(&vm);
  auto device_instance = get_device_instance(vm);
  auto out = env->NewByteArray(device_instance.size);
  auto data = reinterpret_cast<jbyte*>(device_instance.data);
  env->SetByteArrayRegion(out, 0, device_instance.size, data);
  free_device_instance(device_instance);
  return out;
}

}  // extern "C"
