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

#include "gapir/cc/context.h"
#include "gapir/cc/crash_uploader.h"
#include "gapir/cc/memory_manager.h"
#include "gapir/cc/replay_connection.h"
#include "gapir/cc/resource_disk_cache.h"
#include "gapir/cc/resource_in_memory_cache.h"
#include "gapir/cc/resource_requester.h"
#include "gapir/cc/server.h"

#include "core/cc/crash_handler.h"
#include "core/cc/debugger.h"
#include "core/cc/log.h"
#include "core/cc/socket_connection.h"
#include "core/cc/supported_abis.h"
#include "core/cc/target.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <memory>
#include <mutex>
#include <thread>

#if TARGET_OS == GAPID_OS_ANDROID
#include <sys/stat.h>
#include "android_native_app_glue.h"
#endif  // TARGET_OS == GAPID_OS_ANDROID

using namespace core;
using namespace gapir;

namespace {

std::vector<uint32_t> memorySizes{
    2 * 1024 * 1024 * 1024U,  // 2GB
    1 * 1024 * 1024 * 1024U,  // 1GB
    512 * 1024 * 1024U,       // 512MB
    256 * 1024 * 1024U,       // 256MB
    128 * 1024 * 1024U,       // 128MB
};

// createResourceProvider constructs and returns a ResourceInMemoryCache.
// If cachePath is non-null then the ResourceInMemoryCache will be backed by a
// disk-cache.
std::unique_ptr<ResourceInMemoryCache> createResourceProvider(
    const char* cachePath, MemoryManager* memoryManager) {
  if (cachePath != nullptr) {
    GAPID_FATAL("Disk cache is currently out of service. Got %s", cachePath);
    return std::unique_ptr<ResourceInMemoryCache>(ResourceInMemoryCache::create(
        ResourceDiskCache::create(ResourceRequester::create(), cachePath),
        memoryManager->getBaseAddress()));
  } else {
    return std::unique_ptr<ResourceInMemoryCache>(ResourceInMemoryCache::create(
        ResourceRequester::create(), memoryManager->getBaseAddress()));
  }
}

// Setup creates and starts a replay server at the given URI port. Returns the
// created and started server.
// Note the given memory manager and the crash handler, they may be used for
// multiple connections, so a mutex lock is passed in to make the accesses to
// to them exclusive to one connected client. All other replay requests from
// other clients will be blocked, until the current replay finishes.
std::unique_ptr<Server> Setup(const char* uri, const char* authToken,
                              const char* cachePath, int idleTimeoutSec,
                              core::CrashHandler* crashHandler,
                              MemoryManager* memMgr, std::mutex* lock) {
  // Return a replay server with the following replay ID handler. The first
  // package for a replay must be the ID of the replay.
  return Server::createAndStart(
      uri, authToken, idleTimeoutSec,
      [cachePath, memMgr, crashHandler, lock](ReplayConnection* replayConn,
                                              const std::string& replayId) {
        std::lock_guard<std::mutex> mem_mgr_crash_hdl_lock_guard(*lock);
        std::unique_ptr<ResourceInMemoryCache> resourceProvider(
            createResourceProvider(cachePath, memMgr));

        std::unique_ptr<CrashUploader> crash_uploader =
            std::unique_ptr<CrashUploader>(
                new CrashUploader(*crashHandler, replayConn));

        std::unique_ptr<Context> context = Context::create(
            replayConn, *crashHandler, resourceProvider.get(), memMgr);

        if (context == nullptr) {
          GAPID_WARNING("Loading Context failed!");
          return;
        }
        context->prefetch(resourceProvider.get());

        GAPID_INFO("Replay started");
        bool ok = context->interpret();
        GAPID_INFO("Replay %s", ok ? "finished successfully" : "failed");
      });
}

}  // anonymous namespace

#if TARGET_OS == GAPID_OS_ANDROID

const char* pipeName() {
#ifdef __x86_64
  return "gapir-x86-64";
#elif defined __i386
  return "gapir-x86";
#elif defined __ARM_ARCH_7A__
  return "gapir-arm";
#elif defined __aarch64__
  return "gapir-arm64";
#else
#error "Unrecognised target architecture"
#endif
}

// Main function for android
void android_main(struct android_app* app) {
  MemoryManager memoryManager(memorySizes);
  CrashHandler crashHandler;

  // Get the path of the file system socket.
  const char* pipe = pipeName();
  std::string internal_data_path = std::string(app->activity->internalDataPath);
  std::string socket_file_path = internal_data_path + "/" + std::string(pipe);
  std::string uri = std::string("unix://") + socket_file_path;

  __android_log_print(ANDROID_LOG_DEBUG, "GAPIR",
                      "Started Graphics API Replay daemon.\n"
                      "Listening on unix socket '%s'\n"
                      "Supported ABIs: %s\n",
                      uri.c_str(), core::supportedABIs());

  int idleTimeoutSec = 0;  // No timeout
  std::mutex lock;
  std::unique_ptr<Server> server =
      Setup(uri.c_str(), nullptr, nullptr, idleTimeoutSec, &crashHandler,
            &memoryManager, &lock);
  std::thread waiting_thread([&]() { server.get()->wait(); });
  if (chmod(socket_file_path.c_str(), S_IRUSR | S_IWUSR | S_IROTH | S_IWOTH)) {
    GAPID_ERROR("Chmod failed!");
  }

  bool alive = true;
  while (alive) {
    int ident;
    int fdesc;
    int events;
    struct android_poll_source* source;
    while ((ident = ALooper_pollAll(0, &fdesc, &events, (void**)&source)) >=
           0) {
      // process this event
      if (source) {
        source->process(app, source);
      }
      if (app->destroyRequested) {
        unlink(socket_file_path.c_str());
        server->shutdown();
        alive = false;
        break;
      }
    }
  }
  waiting_thread.join();
}

#else  // TARGET_OS == GAPID_OS_ANDROID

// Main function for PC
int main(int argc, const char* argv[]) {
  int logLevel = LOG_LEVEL;
  const char* logPath = "logs/gapir.log";

  bool wait_for_debugger = false;
  const char* cachePath = nullptr;
  const char* portArgStr = "0";
  const char* authTokenFile = nullptr;
  int idleTimeoutSec = 0;

  for (int i = 1; i < argc; i++) {
    if (strcmp(argv[i], "--auth-token-file") == 0) {
      if (i + 1 >= argc) {
        GAPID_FATAL("Usage: --auth-token-file <token-string>");
      }
      authTokenFile = argv[++i];
    } else if (strcmp(argv[i], "--cache") == 0) {
      if (i + 1 >= argc) {
        GAPID_FATAL("Usage: --cache <cache-directory>");
      }
      cachePath = argv[++i];
    } else if (strcmp(argv[i], "--port") == 0) {
      if (i + 1 >= argc) {
        GAPID_FATAL("Usage: --port <port_num>");
      }
      portArgStr = argv[++i];
    } else if (strcmp(argv[i], "--log-level") == 0) {
      if (i + 1 >= argc) {
        GAPID_FATAL("Usage: --log-level <F|E|W|I|D|V>");
      }
      switch (argv[++i][0]) {
        case 'F':
          logLevel = LOG_LEVEL_FATAL;
          break;
        case 'E':
          logLevel = LOG_LEVEL_ERROR;
          break;
        case 'W':
          logLevel = LOG_LEVEL_WARNING;
          break;
        case 'I':
          logLevel = LOG_LEVEL_INFO;
          break;
        case 'D':
          logLevel = LOG_LEVEL_DEBUG;
          break;
        case 'V':
          logLevel = LOG_LEVEL_VERBOSE;
          break;
        default:
          GAPID_FATAL("Usage: --log-level <F|E|W|I|D|V>");
      }
    } else if (strcmp(argv[i], "--log") == 0) {
      if (i + 1 >= argc) {
        GAPID_FATAL("Usage: --log <log-file-path>");
      }
      logPath = argv[++i];
    } else if (strcmp(argv[i], "--idle-timeout-sec") == 0) {
      if (i + 1 >= argc) {
        GAPID_FATAL("Usage: --idle-timeout-sec <timeout in seconds>");
      }
      idleTimeoutSec = atoi(argv[++i]);
    } else if (strcmp(argv[i], "--wait-for-debugger") == 0) {
      wait_for_debugger = true;
    } else if (strcmp(argv[i], "--version") == 0) {
      printf("GAPIR version " GAPID_VERSION_AND_BUILD "\n");
      return 0;
    } else {
      GAPID_FATAL("Unknown argument: %s", argv[i]);
    }
  }

  if (wait_for_debugger) {
    GAPID_INFO("Waiting for debugger to attach");
    core::Debugger::waitForAttach();
  }

  core::CrashHandler crashHandler;

  GAPID_LOGGER_INIT(logLevel, "gapir", logPath);

  // Read the auth-token.
  // Note: This must come before the socket is created as the auth token
  // file is deleted by GAPIS as soon as the port is written to stdout.
  std::vector<char> authToken;
  if (authTokenFile != nullptr) {
    FILE* file = fopen(authTokenFile, "rb");
    if (file == nullptr) {
      GAPID_FATAL("Unable to open auth-token file: %s", authTokenFile);
    }
    if (fseek(file, 0, SEEK_END) != 0) {
      GAPID_FATAL("Unable to get length of auth-token file: %s", authTokenFile);
    }
    size_t size = ftell(file);
    fseek(file, 0, SEEK_SET);
    authToken.resize(size + 1, 0);
    if (fread(&authToken[0], 1, size, file) != size) {
      GAPID_FATAL("Unable to read auth-token file: %s", authTokenFile);
    }
    fclose(file);
  }

  MemoryManager memoryManager(memorySizes);

  // If the user does not assign a port to use, get a free TCP port from OS.
  const char local_host_name[] = "127.0.0.1";
  std::string portStr(portArgStr);
  if (portStr == "0") {
    uint32_t port = SocketConnection::getFreePort(local_host_name);
    if (port == 0) {
      GAPID_FATAL("Failed to find a free port for hostname: '%s'",
                  local_host_name);
    }
    portStr = std::to_string(port);
  }
  std::string uri =
      std::string(local_host_name) + std::string(":") + std::string(portStr);

  std::mutex lock;
  std::unique_ptr<Server> server =
      Setup(uri.c_str(), (authToken.size() > 0) ? authToken.data() : nullptr,
            cachePath, idleTimeoutSec, &crashHandler, &memoryManager, &lock);
  // The following message is parsed by launchers to detect the selected port.
  // DO NOT CHANGE!
  printf("Bound on port '%s'\n", portStr.c_str());
  fflush(stdout);

  server->wait();
  return EXIT_SUCCESS;
}

#endif  // TARGET_OS == GAPID_OS_ANDROID
