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

#include <android/log.h>
#include <dlfcn.h>
#include <stdio.h>
#include <string.h>
#include <sys/system_properties.h>
#include <string>

#define _LOG(lvl, name, msg, ...)                          \
  do {                                                     \
    printf(name ": " msg "\n", ##__VA_ARGS__);             \
    __android_log_print(lvl, "GAPID", msg, ##__VA_ARGS__); \
  } while (false)

#define LOG_ERR(msg, ...) _LOG(ANDROID_LOG_ERROR, "E", msg, ##__VA_ARGS__)
#define LOG_WARN(msg, ...) _LOG(ANDROID_LOG_WARN, "W", msg, ##__VA_ARGS__)
#define LOG_INFO(msg, ...) _LOG(ANDROID_LOG_INFO, "I", msg, ##__VA_ARGS__)

#define GET_PROP(name, trans)                      \
  do {                                             \
    char _v[PROP_VALUE_MAX] = {0};                 \
    if (__system_property_get(name, _v) == 0) {    \
      LOG_ERR("Failed reading property %s", name); \
      std::abort();                                \
    }                                              \
    trans;                                         \
  } while (0)
#define GET_STRING_PROP(n, t) GET_PROP(n, t = _v)

#if defined(__arm__)
#define ABI "armeabi-v7a"
#elif defined(__aarch64__)
#define ABI "arm64-v8a"
#elif defined(__i686__)
#define ABI "x86"
#elif defined(__x86_64__)
#define ABI "x86_64"
#else
#error "Unsupported ABI"
#endif

#define NELEM(x) (sizeof(x) / sizeof(x[0]))

typedef void (*FN_PTR)(void);

const char* kDriverProperty = "ro.gfx.driver.1";
const char* kProducerPaths[] = {
    "/assets/libgpudataproducer.so",
    "/lib/" ABI "/libgpudataproducer.so",
};

std::string getDriverPackageOverride() {
  std::string driver;
  // Read gapid.driver_package_override setting to override default property.
  FILE* fp = popen("settings get global gapid.driver_package_override", "r");
  if (fp == nullptr) {
    return driver;
  }
  char buffer[1024] = {0};
  if (fgets(buffer, sizeof(buffer) - 1, fp) != nullptr) {
    driver = buffer;
    if (!driver.empty()) {
      driver.pop_back();  // chop '\n'.
    }
    if (driver == "null") {
      driver = "";
    }
    if (!driver.empty()) {
      LOG_INFO("Driver package override: %s", driver.c_str());
    }
  }
  pclose(fp);
  return driver;
}

std::string getDriver() {
  std::string driver = getDriverPackageOverride();
  if (driver.empty()) {
    GET_STRING_PROP(kDriverProperty, driver);
    LOG_INFO("Driver package: %s", driver.c_str());
  }

  if (driver.empty()) {
    LOG_ERR("No driver package set");
    std::abort();
  }

  // Check that the driver is a valid Java package.
  for (char const& c : driver) {
    if ((c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') &&
        (c != '.' && c != '_' && c != '$')) {
      LOG_ERR("Invalid driver package: %s", driver.c_str());
      std::abort();
    }
  }
  return driver;
}

std::string getApkPath(const std::string& driver) {
  std::string cmd = "pm path '" + driver + "'";
  FILE* fp = popen(cmd.c_str(), "r");
  if (fp == nullptr) {
    LOG_ERR("Failed to run '%s'", cmd.c_str());
    std::abort();
  }

  char buffer[1024] = {0};
  if (fgets(buffer, sizeof(buffer) - 1, fp) == nullptr) {
    LOG_ERR("Failed to read from pm path");
    std::abort();
  }
  pclose(fp);

  if (!strcmp("package:", buffer)) {
    LOG_ERR("Unrecognized pm path output: %s", buffer);
    std::abort();
  }

  // Remove the trailing '\n'.
  buffer[strcspn(buffer, "\n")] = 0;

  std::string path(&buffer[8]);
  LOG_INFO("Driver package path: %s", path.c_str());
  return path;
}

FN_PTR loadLibrary(const std::string& path, const char* lib) {
  char* error;

  std::string so = path + "!" + lib;
  LOG_INFO("Trying %s", so.c_str());
  void* handle = dlopen(so.c_str(), RTLD_GLOBAL);
  if ((error = dlerror()) != nullptr || handle == nullptr) {
    LOG_WARN("Error loading lib: %s", error);
    return nullptr;
  }

  FN_PTR startFunc = (FN_PTR)dlsym(handle, "start");
  if ((error = dlerror()) != nullptr) {
    LOG_ERR("Error looking for start symbol: %s", error);
    dlclose(handle);
    return nullptr;
  }
  return startFunc;
}

// Program to load the GPU Perfetto producer .so and call start().
int main(int argc, char** argv) {
  std::string driver = getDriver();
  std::string path = getApkPath(driver);

  dlerror();
  FN_PTR startFunc = nullptr;
  for (int i = 0; startFunc == nullptr && i < NELEM(kProducerPaths); i++) {
    startFunc = loadLibrary(path, kProducerPaths[i]);
  }

  if (startFunc == nullptr) {
    LOG_ERR("Did not find the producer library");
    std::abort();
  }

  LOG_INFO("Calling start at %p", startFunc);
  (*startFunc)();
  LOG_WARN("Producer has exited.");

  return 0;
}
