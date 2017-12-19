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
#include "gapir/cc/resource_disk_cache.h"
#include "gapir/cc/resource_in_memory_cache.h"
#include "gapir/cc/resource_requester.h"
#include "gapir/cc/server_connection.h"
#include "gapir/cc/server_listener.h"

#include "core/cc/connection.h"
#include "core/cc/crash_handler.h"
#include "core/cc/debugger.h"
#include "core/cc/log.h"
#include "core/cc/socket_connection.h"
#include "core/cc/supported_abis.h"
#include "core/cc/target.h"

#include <memory>
#include <stdlib.h>
#include <string.h>
#include <thread>

#if TARGET_OS == GAPID_OS_ANDROID
#include <android_native_app_glue.h>
#endif  // TARGET_OS == GAPID_OS_ANDROID

using namespace core;
using namespace gapir;

namespace {

std::vector<uint32_t> memorySizes{
    2 * 1024 * 1024 * 1024U,  // 2GB
    1 * 1024 * 1024 * 1024U,  // 1GB
         512 * 1024 * 1024U,  // 512MB
         256 * 1024 * 1024U,  // 256MB
         128 * 1024 * 1024U,  // 128MB
};

// createResourceProvider constructs and returns a ResourceInMemoryCache.
// If cachePath is non-null then the ResourceInMemoryCache will be backed by a
// disk-cache.
std::unique_ptr<ResourceInMemoryCache> createResourceProvider(
        const char* cachePath, MemoryManager* memoryManager) {
    if (cachePath != nullptr) {
        GAPID_FATAL("Disk cache is currently out of service. Got %s", cachePath);
        return std::unique_ptr<ResourceInMemoryCache>(
            ResourceInMemoryCache::create(
                ResourceDiskCache::create(ResourceRequester::create(), cachePath),
                memoryManager->getBaseAddress()));
    } else {
        return std::unique_ptr<ResourceInMemoryCache>(
            ResourceInMemoryCache::create(
                ResourceRequester::create(), memoryManager->getBaseAddress()));
    }
}

void listenConnections(Connection* conn,
                       const char* authToken,
                       const char* cachePath,
                       int idleTimeoutMs,
                       MemoryManager* memoryManager,
                       core::CrashHandler& crashHandler) {
    ServerListener listener(conn, memoryManager->getSize());

    std::unique_ptr<ResourceInMemoryCache> resourceProvider(
            createResourceProvider(cachePath, memoryManager));

    while (true) {
        std::unique_ptr<ServerConnection> acceptedConn(
                listener.acceptConnection(idleTimeoutMs, authToken));
        if (!acceptedConn) {
            GAPID_INFO("Shutting down");
            break;
        }
        std::unique_ptr<CrashUploader> crash_uploader = std::unique_ptr<CrashUploader>(new CrashUploader(crashHandler, *acceptedConn));

        std::unique_ptr<Context> context =
                Context::create(*acceptedConn, resourceProvider.get(), memoryManager);
        if (context == nullptr) {
            GAPID_WARNING("Loading Context failed!");
            continue;
        }

        context->prefetch(resourceProvider.get());

        GAPID_INFO("Replay started");
        bool ok = context->interpret();
        GAPID_INFO("Replay %s", ok ? "finished successfully" : "failed");
    }
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
#   error "Unrecognised target architecture"
#endif
}

// Main function for android
void android_main(struct android_app* app) {
    app_dummy();
    MemoryManager memoryManager(memorySizes);
    CrashHandler crashHandler;

    const char* pipe = pipeName();
    auto conn = SocketConnection::createPipe(pipe, "");
    if (conn == nullptr) {
        GAPID_FATAL("Failed to create abstract local port: %s", pipe);
    }

    __android_log_print(ANDROID_LOG_DEBUG, "GAPIR",
            "Started Graphics API Replay daemon.\n"
            "Listening on localabstract port '%s'\n"
            "Supported ABIs: %s\n",
            pipe, core::supportedABIs());

    // Note if you want to create a disk cache create it under:
    // app->activity->internalDataPath
    std::thread listening_thread([&]() {
      listenConnections(conn.get(), nullptr, nullptr, Connection::NO_TIMEOUT,
                        &memoryManager, crashHandler);
    });
    while (true) {
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
          conn->close();
          break;
        }
      }
    }
    listening_thread.join();
}

#else  // TARGET_OS == GAPID_OS_ANDROID

// Main function for PC
int main(int argc, const char* argv[]) {
    int logLevel = LOG_LEVEL;
    const char* logPath = "logs/gapir.log";

    bool wait_for_debugger = false;
    bool enable_crash_reporting = false;
    const char* cachePath = nullptr;
    const char* portStr = "0";
    const char* authTokenFile = nullptr;
    int idleTimeoutMs = Connection::NO_TIMEOUT;

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
            portStr = argv[++i];
        } else if (strcmp(argv[i], "--log-level") == 0) {
            if (i + 1 >= argc) {
                GAPID_FATAL("Usage: --log-level <F|E|W|I|D|V>");
            }
            switch (argv[++i][0]) {
              case 'F': logLevel = LOG_LEVEL_FATAL; break;
              case 'E': logLevel = LOG_LEVEL_ERROR; break;
              case 'W': logLevel = LOG_LEVEL_WARNING; break;
              case 'I': logLevel = LOG_LEVEL_INFO; break;
              case 'D': logLevel = LOG_LEVEL_DEBUG; break;
              case 'V': logLevel = LOG_LEVEL_VERBOSE; break;
              default:
                GAPID_FATAL("Usage: --log-level <F|E|W|I|D|V>");
            }
        } else if (strcmp(argv[i], "--log") == 0) {
            if (i + 1 >= argc) {
                GAPID_FATAL("Usage: --log <log-file-path>");
            }
            logPath = argv[++i];
        } else if (strcmp(argv[i], "--idle-timeout-ms") == 0) {
            if (i + 1 >= argc) {
                GAPID_FATAL("Usage: --idle-timeout-ms <timeout in milliseconds>");
            }
            idleTimeoutMs = atoi(argv[++i]);
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
    auto conn = SocketConnection::createSocket("127.0.0.1", portStr);
    if (conn == nullptr) {
        GAPID_FATAL("Failed to create listening socket on port: %s", portStr);
    }

    listenConnections(conn.get(),
                      (authToken.size() > 0) ? authToken.data() : nullptr,
                      cachePath, idleTimeoutMs, &memoryManager, crashHandler);
    return EXIT_SUCCESS;
}

#endif  // TARGET_OS == GAPID_OS_ANDROID
