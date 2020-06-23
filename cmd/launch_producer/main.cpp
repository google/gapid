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
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/system_properties.h>
#include <unistd.h>

#include <string>

// TODO(b/148950543): Figure out why stdout is not captured.
#define _LOG(lvl, name, msg, ...)                        \
  do {                                                   \
    fprintf(stderr, name ": " msg "\n", ##__VA_ARGS__);  \
    __android_log_print(lvl, "AGI", msg, ##__VA_ARGS__); \
  } while (false)

#define LOG_ERR(msg, ...) _LOG(ANDROID_LOG_ERROR, "E", msg, ##__VA_ARGS__)
#define LOG_WARN(msg, ...) _LOG(ANDROID_LOG_WARN, "W", msg, ##__VA_ARGS__)
#define LOG_INFO(msg, ...) _LOG(ANDROID_LOG_INFO, "I", msg, ##__VA_ARGS__)

#define NELEM(x) (sizeof(x) / sizeof(x[0]))

typedef void (*FN_PTR)(void);

const char* kProducerPaths[] = {
    "libgpudataproducer.so",
};
const char* kPidFileName = "/data/local/tmp/agi_launch_producer.pid";

FN_PTR loadLibrary(const char* lib) {
  char* error;

  LOG_INFO("Trying %s", lib);
  void* handle = dlopen(lib, RTLD_GLOBAL);
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

// If a previous producer has died without cleaning up its pidfile,
// here we kill a PID that may be related to another process.
// This is a risk we take, it would be rare for a previous PID to be reused,
// and in the worst case we kill a non-critical application as core services
// are not killable that easily.
void killExistingProcess() {
  int fd = open(kPidFileName, O_RDONLY);
  if (fd == -1) {
    return;
  }
  char pidString[10];
  if (read(fd, pidString, 10) > 0) {
    int pid = -1;
    sscanf(pidString, "%d", &pid);
    if (pid > 0) {
      kill(pid, SIGINT);
    }
  }
  close(fd);
}

bool writeToPidFile() {
  killExistingProcess();
  int fd = open(kPidFileName, O_CREAT | O_WRONLY | O_TRUNC,
                S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IWOTH);
  if (fd == -1) {
    return false;
  }
  pid_t pid = getpid();
  char pidString[10];
  sprintf(pidString, "%d", pid);
  write(fd, pidString, strlen(pidString));
  close(fd);
  return true;
}

// Program to load the GPU Perfetto producer .so and call start().
int main(int argc, char** argv) {
  if (!writeToPidFile()) {
    LOG_ERR("Could not open %s", kPidFileName);
    std::abort();
  }

  dlerror();
  FN_PTR startFunc = nullptr;
  for (int i = 0; startFunc == nullptr && i < NELEM(kProducerPaths); i++) {
    startFunc = loadLibrary(kProducerPaths[i]);
  }

  if (startFunc == nullptr) {
    LOG_ERR("Did not find the producer library");
    LOG_ERR("LD_LIBRARY_PATH=%s", getenv("LD_LIBRARY_PATH"));
    std::abort();
  }

  LOG_INFO("Calling start at %p", startFunc);
  (*startFunc)();
  LOG_WARN("Producer has exited.");

  return 0;
}
