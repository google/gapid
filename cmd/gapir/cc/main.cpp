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

#include "gapir/cc/archive_replay_service.h"
#include "gapir/cc/cached_resource_loader.h"
#include "gapir/cc/context.h"
#include "gapir/cc/crash_uploader.h"
#include "gapir/cc/grpc_replay_service.h"
#include "gapir/cc/in_memory_resource_cache.h"
#include "gapir/cc/memory_manager.h"
#include "gapir/cc/on_disk_resource_cache.h"
#include "gapir/cc/server.h"
#include "gapir/cc/surface.h"

#include "core/cc/crash_handler.h"
#include "core/cc/debugger.h"
#include "core/cc/log.h"
#include "core/cc/socket_connection.h"
#include "core/cc/supported_abis.h"
#include "core/cc/target.h"

#include <signal.h>
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
#elif TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
#include <dirent.h>
#include <ftw.h>
#include <sys/types.h>
#endif  // TARGET_OS == GAPID_OS_ANDROID

using namespace core;
using namespace gapir;

namespace {

std::vector<uint32_t> memorySizes {
// If we are on desktop, we can try more memory
#if TARGET_OS != GAPID_OS_ANDROID
  3 * 1024 * 1024 * 1024U,  // 3GB
#endif
      2 * 1024 * 1024 * 1024U,  // 2GB
      1 * 1024 * 1024 * 1024U,  // 1GB
      512 * 1024 * 1024U,       // 512MB
      256 * 1024 * 1024U,       // 256MB
      128 * 1024 * 1024U,       // 128MB
};

#if TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
std::string getTempOnDiskCachePath() {
  const char* tmpDir = std::getenv("TMPDIR");
  if (!tmpDir) {
    struct stat sb;
    if (stat("/tmp", &sb) == 0 && S_ISDIR(sb.st_mode)) {
      tmpDir = "/tmp";
    } else {
      GAPID_WARNING("$TMPDIR is null and /tmp is not a directory");
      return "";
    }
  }

  auto t = std::string(tmpDir) + "/gapir-cache.XXXXXX";
  std::vector<char> v(t.begin(), t.end());
  v.push_back('\0');
  char* path = mkdtemp(v.data());
  if (path == nullptr) {
    GAPID_WARNING("Failed at creating temp dir");
    return "";
  }
  return path;
}
#endif

// Setup creates and starts a replay server at the given URI port. Returns the
// created and started server.
// Note the given memory manager and the crash handler, they may be used for
// multiple connections, so a mutex lock is passed in to make the accesses to
// to them exclusive to one connected client. All other replay requests from
// other clients will be blocked, until the current replay finishes.
std::unique_ptr<Server> Setup(const char* uri, const char* authToken,
                              ResourceCache* cache, int idleTimeoutSec,
                              core::CrashHandler* crashHandler,
                              MemoryManager* memMgr, std::mutex* lock) {
  // Return a replay server with the following replay ID handler. The first
  // package for a replay must be the ID of the replay.
  return Server::createAndStart(
      uri, authToken, idleTimeoutSec,
      [cache, memMgr, crashHandler, lock](GrpcReplayService* replayConn,
                                          const std::string& replayId) {
        std::lock_guard<std::mutex> mem_mgr_crash_hdl_lock_guard(*lock);

        std::unique_ptr<ResourceLoader> resLoader;
        if (cache == nullptr) {
          resLoader = PassThroughResourceLoader::create(replayConn);
        } else {
          resLoader = CachedResourceLoader::create(
              cache, PassThroughResourceLoader::create(replayConn));
        }

        std::unique_ptr<CrashUploader> crash_uploader =
            std::unique_ptr<CrashUploader>(
                new CrashUploader(*crashHandler, replayConn));

        std::unique_ptr<Context> context =
            Context::create(replayConn, *crashHandler, resLoader.get(), memMgr);

        if (context == nullptr) {
          GAPID_WARNING("Loading Context failed!");
          return;
        }
        if (cache != nullptr) {
          context->prefetch(cache);
        }

        GAPID_INFO("Replay started");
        bool ok = context->interpret();
        GAPID_INFO("Replay %s", ok ? "finished successfully" : "failed");
      });
}

}  // anonymous namespace

#if TARGET_OS == GAPID_OS_ANDROID

namespace {

template <typename... Args>
jobject jni_call_o(JNIEnv* env, jobject obj, const char* name, const char* sig,
                   Args&&... args) {
  jmethodID mid = env->GetMethodID(env->GetObjectClass(obj), name, sig);
  return env->CallObjectMethod(obj, mid, std::forward<Args>(args)...);
}

template <typename... Args>
int jni_call_i(JNIEnv* env, jobject obj, const char* name, const char* sig,
               Args&&... args) {
  jmethodID mid = env->GetMethodID(env->GetObjectClass(obj), name, sig);
  return env->CallIntMethod(obj, mid, std::forward<Args>(args)...);
}

struct Options {
  int idleTimeoutSec = 0;
  std::string authToken = "";

  static Options Parse(struct android_app* app) {
    Options opts;

    JNIEnv* env;
    app->activity->vm->AttachCurrentThread(&env, 0);

    jobject intent = jni_call_o(env, app->activity->clazz, "getIntent",
                                "()Landroid/content/Intent;");
    opts.idleTimeoutSec =
        jni_call_i(env, intent, "getIntExtra", "(Ljava/lang/String;I)I",
                   env->NewStringUTF("idle_timeout"), 0);
    jobject token = jni_call_o(env, intent, "getStringExtra",
                               "(Ljava/lang/String;)Ljava/lang/String;",
                               env->NewStringUTF("auth_token"));
    if (token != nullptr) {
      const char* tmp = env->GetStringUTFChars((jstring)token, nullptr);
      opts.authToken = tmp;
      env->ReleaseStringUTFChars((jstring)token, tmp);
    }

    app->activity->vm->DetachCurrentThread();
    return opts;
  }
};

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

void android_process(struct android_app* app, int32_t cmd) {
  switch (cmd) {
    case APP_CMD_INIT_WINDOW: {
      gapir::android_window = app->window;
      GAPID_DEBUG("Received window: %p\n", gapir::android_window);
      break;
    }
  }
}

}  // namespace

// Main function for android
void android_main(struct android_app* app) {
  MemoryManager memoryManager(memorySizes);
  CrashHandler crashHandler;

  // Get the path of the file system socket.
  const char* pipe = pipeName();
  std::string internal_data_path = std::string(app->activity->internalDataPath);
  std::string socket_file_path = internal_data_path + "/" + std::string(pipe);
  std::string uri = std::string("unix://") + socket_file_path;

  GAPID_INFO(
      "Started Graphics API Replay daemon.\n"
      "Listening on unix socket '%s'\n"
      "Supported ABIs: %s\n",
      uri.c_str(), core::supportedABIs());

  auto opts = Options::Parse(app);
  auto cache = InMemoryResourceCache::create(memoryManager.getTopAddress());
  std::mutex lock;
  std::unique_ptr<Server> server =
      Setup(uri.c_str(), opts.authToken.c_str(), cache.get(),
            opts.idleTimeoutSec, &crashHandler, &memoryManager, &lock);
  std::thread waiting_thread([&]() { server.get()->wait(); });
  if (chmod(socket_file_path.c_str(), S_IRUSR | S_IWUSR | S_IROTH | S_IWOTH)) {
    GAPID_ERROR("Chmod failed!");
  }

  app->onAppCmd = android_process;

  bool alive = true;
  while (alive) {
    int fdesc;
    int events;
    struct android_poll_source* source;
    while (ALooper_pollAll(1000, &fdesc, &events, (void**)&source) >= 0) {
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

namespace {

struct Options {
  struct OnDiskCache {
    bool enabled = false;
    bool cleanUp = false;
    const char* path = "";
  };

  int logLevel = LOG_LEVEL;
  const char* logPath = "logs/gapir.log";

  enum ReplayMode {
    kUnknown = 0,    // Can't determine replay type from arguments yet.
    kConflict,       // Impossible combination of command line arguments.
    kReplayServer,   // Run gapir as a server.
    kReplayArchive,  // Replay an exported archive.
  };
  ReplayMode mode = kUnknown;
  bool waitForDebugger = false;
  const char* cachePath = nullptr;
  const char* portArgStr = "0";
  const char* authTokenFile = nullptr;
  int idleTimeoutSec = 0;
  const char* replayArchive = nullptr;
  const char* postbackDirectory = "";
  bool version = false;

  OnDiskCache onDiskCacheOptions;

  static Options Parse(int argc, const char* argv[]) {
    Options opts;

    for (int i = 1; i < argc; i++) {
      if (strcmp(argv[i], "--replay-archive") == 0) {
        opts.SetMode(kReplayArchive);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --replay-archive <archive-directory>");
        }
        opts.replayArchive = argv[++i];
      } else if (strcmp(argv[i], "--postback-dir") == 0) {
        opts.SetMode(kReplayArchive);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --postback-dir <output-directory>");
        }
        opts.postbackDirectory = argv[++i];
      } else if (strcmp(argv[i], "--auth-token-file") == 0) {
        opts.SetMode(kReplayServer);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --auth-token-file <token-string>");
        }
        opts.authTokenFile = argv[++i];
      } else if (strcmp(argv[i], "--enable-disk-cache") == 0) {
        opts.SetMode(kReplayServer);
        opts.onDiskCacheOptions.enabled = true;
      } else if (strcmp(argv[i], "--disk-cache-path") == 0) {
        opts.SetMode(kReplayServer);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --disk-cache-path <cache-directory>");
        }
        opts.onDiskCacheOptions.path = argv[++i];
      } else if (strcmp(argv[i], "--cleanup-on-disk-cache") == 0) {
        opts.onDiskCacheOptions.cleanUp = true;
      } else if (strcmp(argv[i], "--port") == 0) {
        opts.SetMode(kReplayServer);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --port <port_num>");
        }
        opts.portArgStr = argv[++i];
      } else if (strcmp(argv[i], "--log-level") == 0) {
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --log-level <F|E|W|I|D|V>");
        }
        switch (argv[++i][0]) {
          case 'F':
            opts.logLevel = LOG_LEVEL_FATAL;
            break;
          case 'E':
            opts.logLevel = LOG_LEVEL_ERROR;
            break;
          case 'W':
            opts.logLevel = LOG_LEVEL_WARNING;
            break;
          case 'I':
            opts.logLevel = LOG_LEVEL_INFO;
            break;
          case 'D':
            opts.logLevel = LOG_LEVEL_DEBUG;
            break;
          case 'V':
            opts.logLevel = LOG_LEVEL_VERBOSE;
            break;
          default:
            GAPID_FATAL("Usage: --log-level <F|E|W|I|D|V>");
        }
      } else if (strcmp(argv[i], "--log") == 0) {
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --log <log-file-path>");
        }
        opts.logPath = argv[++i];
      } else if (strcmp(argv[i], "--idle-timeout-sec") == 0) {
        opts.SetMode(kReplayServer);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --idle-timeout-sec <timeout in seconds>");
        }
        opts.idleTimeoutSec = atoi(argv[++i]);
      } else if (strcmp(argv[i], "--wait-for-debugger") == 0) {
        opts.waitForDebugger = true;
      } else if (strcmp(argv[i], "--version") == 0) {
        opts.version = true;
      } else {
        GAPID_FATAL("Unknown argument: %s", argv[i]);
      }
    }
    return opts;
  }

 private:
  void SetMode(ReplayMode mode) {
    if (this->mode != kUnknown && this->mode != mode) {
      mode = kConflict;
    }
    this->mode = mode;
  }
};

// createCache constructs and returns a ResourceCache based on the given
// onDiskCacheOpts. If on-disk cache is not enabled or not possible to create,
// an in-memory cache will be built and returned. If on-disk cache is created
// in a temporary directory or onDiskCacheOpts is specified to clear cache
// files, a monitor process will be forked to delete the cache files when the
// main GAPIR VM process ends.
std::unique_ptr<ResourceCache> createCache(
    const Options::OnDiskCache& onDiskCacheOpts, MemoryManager* memoryManager) {
#if TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
  if (!onDiskCacheOpts.enabled) {
    return InMemoryResourceCache::create(memoryManager->getTopAddress());
  }
  auto onDiskCachePath = std::string(onDiskCacheOpts.path);
  bool cleanUpOnDiskCache = onDiskCacheOpts.cleanUp;
  bool useTempCacheFolder = false;
  if (onDiskCachePath.size() == 0) {
    useTempCacheFolder = true;
    cleanUpOnDiskCache = true;
    onDiskCachePath = getTempOnDiskCachePath();
  }
  if (onDiskCachePath.size() == 0) {
    GAPID_WARNING(
        "No disk cache path specified and no $TMPDIR environment variable "
        "defined for temporary on-disk cache, fallback to use in-memory "
        "cache.");
    return InMemoryResourceCache::create(memoryManager->getTopAddress());
  }
  auto onDiskCache =
      OnDiskResourceCache::create(onDiskCachePath, cleanUpOnDiskCache);
  if (onDiskCache == nullptr) {
    GAPID_WARNING(
        "On-disk cache creation failed, fallback to use in-memory cache");
    return InMemoryResourceCache::create(memoryManager->getTopAddress());
  }
  GAPID_INFO("On-disk cache created at %s", onDiskCachePath.c_str());
  if (cleanUpOnDiskCache || useTempCacheFolder) {
    GAPID_INFO("On-disk cache files will be cleaned up when GAPIR ends");
    if (fork() == 0) {
      pid_t ppid = getppid();
      while (!kill(ppid, 0)) {
        // check every 500ms
        usleep(500000);
      }
      DIR* dir = opendir(onDiskCachePath.c_str());
      if (dir != nullptr) {
        if (useTempCacheFolder) {
          // Using temporary folder for cache files, delete both the files and
          // the folder.
          nftw(onDiskCachePath.c_str(),
               [](const char* fpath, const struct stat* sb, int typeflag,
                  struct FTW* ftwbuf) -> int {
                 switch (typeflag) {
                   case FTW_D:
                     return rmdir(fpath);
                   default:
                     return unlink(fpath);
                 }
                 return 0;
               },
               64, FTW_DEPTH);
          rmdir(onDiskCachePath.c_str());
        } else {
          // The OnDiskResourceCache must have been created with "clean up"
          // enabled. Calling its destructor to delete the cache files.
          onDiskCache.reset(nullptr);
        }
      }
      exit(0);
    }
  }
  return std::move(onDiskCache);
#else   // TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
  if (onDiskCacheOpts.enabled) {
    GAPID_WARNING(
        "On-disk cache not supported, fallback to use in-memory cache");
  }
#endif  // TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
  // Just use the in-memory cache
  return InMemoryResourceCache::create(memoryManager->getTopAddress());
}
}  // namespace

static int replayArchive(Options opts) {
  // The directory consists an archive(resources.{index,data}) and payload.bin.
  core::CrashHandler crashHandler;
  GAPID_LOGGER_INIT(opts.logLevel, "gapir", opts.logPath);
  MemoryManager memoryManager(memorySizes);
  std::string payloadPath = std::string(opts.replayArchive) + "/payload.bin";
  gapir::ArchiveReplayService replayArchive(payloadPath,
                                            opts.postbackDirectory);
  // All the resource data must be in the archive file, no fallback resource
  // loader to fetch uncached resources data.
  auto onDiskCache = OnDiskResourceCache::create(opts.replayArchive, false);
  std::unique_ptr<ResourceLoader> resLoader =
      CachedResourceLoader::create(onDiskCache.get(), nullptr);
  std::unique_ptr<Context> context = Context::create(
      &replayArchive, crashHandler, resLoader.get(), &memoryManager);

  GAPID_INFO("Replay started");
  bool ok = context->interpret();
  GAPID_INFO("Replay %s", ok ? "finished successfully" : "failed");

  return ok ? EXIT_SUCCESS : EXIT_FAILURE;
}

static int startServer(Options opts) {
  core::CrashHandler crashHandler;

  GAPID_LOGGER_INIT(opts.logLevel, "gapir", opts.logPath);

  // Read the auth-token.
  // Note: This must come before the socket is created as the auth token
  // file is deleted by GAPIS as soon as the port is written to stdout.
  std::vector<char> authToken;
  if (opts.authTokenFile != nullptr) {
    FILE* file = fopen(opts.authTokenFile, "rb");
    if (file == nullptr) {
      GAPID_FATAL("Unable to open auth-token file: %s", opts.authTokenFile);
    }
    if (fseek(file, 0, SEEK_END) != 0) {
      GAPID_FATAL("Unable to get length of auth-token file: %s",
                  opts.authTokenFile);
    }
    size_t size = ftell(file);
    fseek(file, 0, SEEK_SET);
    authToken.resize(size + 1, 0);
    if (fread(&authToken[0], 1, size, file) != size) {
      GAPID_FATAL("Unable to read auth-token file: %s", opts.authTokenFile);
    }
    fclose(file);
  }

  MemoryManager memoryManager(memorySizes);

  // If the user does not assign a port to use, get a free TCP port from OS.
  const char local_host_name[] = "127.0.0.1";
  std::string portStr(opts.portArgStr);
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

  auto cache = createCache(opts.onDiskCacheOptions, &memoryManager);

  std::mutex lock;
  std::unique_ptr<Server> server = Setup(
      uri.c_str(), (authToken.size() > 0) ? authToken.data() : nullptr,
      cache.get(), opts.idleTimeoutSec, &crashHandler, &memoryManager, &lock);
  // The following message is parsed by launchers to detect the selected port.
  // DO NOT CHANGE!
  printf("Bound on port '%s'\n", portStr.c_str());
  fflush(stdout);

  server->wait();

  gapir::WaitForWindowClose();
  return EXIT_SUCCESS;
}

// Main function for PC
int main(int argc, const char* argv[]) {
  Options opts = Options::Parse(argc, argv);

#if TARGET_OS == GAPID_OS_LINUX
  // Ignore SIGPIPE so we can log after gapis closes.
  signal(SIGPIPE, SIG_IGN);
#endif

  if (opts.waitForDebugger) {
    GAPID_INFO("Waiting for debugger to attach");
    core::Debugger::waitForAttach();
  }
  if (opts.version) {
    printf("GAPIR version " GAPID_VERSION_AND_BUILD "\n");
    return EXIT_SUCCESS;
  } else if (opts.mode == Options::kConflict) {
    GAPID_ERROR("Argument conflicts.");
    return EXIT_FAILURE;
  } else if (opts.mode == Options::kReplayArchive) {
    return replayArchive(opts);
  } else {
    return startServer(opts);
  }
}

#endif  // TARGET_OS == GAPID_OS_ANDROID
