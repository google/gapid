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
#include "core/cc/version.h"

#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <memory>
#include <mutex>
#include <sstream>
#include <thread>

#if TARGET_OS == GAPID_OS_ANDROID
#include <android/window.h>
#include <sys/stat.h>
#include "android_native_app_glue.h"
#include "gapir/cc/android/asset_replay_service.h"
#include "gapir/cc/android/asset_resource_cache.h"
#elif TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
#include <dirent.h>
#include <ftw.h>
#include <sys/types.h>
#endif  // TARGET_OS == GAPID_OS_ANDROID

using namespace core;
using namespace gapir;

namespace {

// kSocketName must match "socketName" in gapir/client/device_connection.go
const std::string kSocketName("gapir-socket");

std::shared_ptr<MemoryAllocator> createAllocator() {
#if defined(__x86_64) || defined(__aarch64__)
  size_t size = 16ull * 1024ull * 1024ull * 1024ull;
#else
  size_t size = 2ull * 1024ull * 1024ull * 1024ull;
#endif

  return std::shared_ptr<MemoryAllocator>(new MemoryAllocator(size));
}

enum ReplayMode {
  kUnknown = 0,    // Can't determine replay type from arguments yet.
  kConflict,       // Impossible combination of command line arguments.
  kReplayServer,   // Run gapir as a server.
  kReplayArchive,  // Replay an exported archive.
};

struct Options {
  struct OnDiskCache {
    bool enabled = false;
    bool cleanUp = false;
    const char* path = "";
  };

  int logLevel = LOG_LEVEL;
  const char* logPath = "logs/gapir.log";
  ReplayMode mode = kUnknown;
  bool waitForDebugger = false;
  const char* cachePath = nullptr;
  const char* portArgStr = "0";
  const char* authTokenFile = nullptr;
  int idleTimeoutSec = 0;
  const char* replayArchive = nullptr;
  const char* postbackDirectory = "";
  bool version = false;
  bool help = false;

  OnDiskCache onDiskCacheOptions;

#if TARGET_OS == GAPID_OS_ANDROID
  std::string authToken;
#endif

  static void PrintHelp() {
    GAPID_WARNING(
        "gapir: gapir is a VM for the graphics api debugger system\n");
    GAPID_WARNING("Usage: gapir [args]\n");
    GAPID_WARNING("Args:\n");
    GAPID_WARNING("  --replay-archive string\n");
    GAPID_WARNING(
        "    Path to an archive directory to replay, and then exit\n");
    GAPID_WARNING("  --postback-dir string\n");
    GAPID_WARNING(
        "    Path to a directory to use for outputs of the replay-archive\n");
    GAPID_WARNING("  --auth-token-file string\n");
    GAPID_WARNING(
        "    Path to the a file containing the authentication token\n");
    GAPID_WARNING("  --enable-disk-cache\n");
    GAPID_WARNING(
        "    If set, then gapir will create and use a disk cache for "
        "resources.\n");
    GAPID_WARNING("  --disk-cache-path string\n");
    GAPID_WARNING(
        "    Path to a directory that will be used for the disk cache.\n");
    GAPID_WARNING("    If it contains an existing cache, that will be used\n");
    GAPID_WARNING(
        "    If unset, the disk cache will default to a temp directory\n");
    GAPID_WARNING("  --cleanup-disk-cache\n");
    GAPID_WARNING(
        "    If set, the disk cache will be deleted when gapir exits.\n");
    GAPID_WARNING("  --port int\n");
    GAPID_WARNING("    The port to use when listening for connections\n");
    GAPID_WARNING("  --log-level <F|E|W|I|D|V>\n");
    GAPID_WARNING("    Sets the log level for gapir.\n");
    GAPID_WARNING("  --log string\n");
    GAPID_WARNING("    Sets the path for the log file\n");
    GAPID_WARNING("  --idle-timeout-sec int\n");
    GAPID_WARNING(
        "    Timeout if gapir has not received communication from the server "
        "(default infinity)\n");
    GAPID_WARNING("  --wait-for-debugger\n");
    GAPID_WARNING(
        "    Causes gapir to pause on init, and wait for a debugger to "
        "connect\n");
    GAPID_WARNING("   -h,-help,--help\n");
    GAPID_WARNING("    Prints this help text and exits.\n");
  }

  static void warnAndroid(const char* flag) {
#if TARGET_OS == GAPID_OS_ANDROID
    GAPID_WARNING("Usage: %s is ignored on android devices.", flag);
#endif
  }

  static void ensureNotAndroid(const char* flag) {
#if TARGET_OS == GAPID_OS_ANDROID
    GAPID_FATAL("Usage: %s may not be used on android devices.", flag);
#endif
  }

  static void ensureAndroid(const char* flag) {
#if TARGET_OS != GAPID_OS_ANDROID
    GAPID_FATAL("Usage: %s may not be used on non-android devices.", flag);
#endif
  }

  static void Parse(const std::vector<std::string>& args, Options* opts) {
    std::vector<const char*> argv;
    for (unsigned int i = 0; i < args.size(); ++i) {
      argv.push_back(args[i].c_str());
    }

    Parse(args.size(), args.size() > 0 ? &argv[0] : nullptr, opts);
  }

  static void Parse(int argc, const char* argv[], Options* opts) {
    for (int i = 1; i < argc; i++) {
      if (strcmp(argv[i], "--replay-archive") == 0) {
        ensureNotAndroid("--replay-archive");
        opts->SetMode(kReplayArchive);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --replay-archive <archive-directory>");
        }
        opts->replayArchive = argv[++i];
      } else if (strcmp(argv[i], "--postback-dir") == 0) {
        ensureNotAndroid("--postback-dir");
        opts->SetMode(kReplayArchive);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --postback-dir <output-directory>");
        }
        opts->postbackDirectory = argv[++i];
      } else if (strcmp(argv[i], "--auth-token-file") == 0) {
        opts->SetMode(kReplayServer);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --auth-token-file <token-string>");
        }
        opts->authTokenFile = argv[++i];
      } else if (strcmp(argv[i], "--enable-disk-cache") == 0) {
        ensureNotAndroid("--enable-disk-cache");
        opts->SetMode(kReplayServer);
        opts->onDiskCacheOptions.enabled = true;
      } else if (strcmp(argv[i], "--disk-cache-path") == 0) {
        ensureNotAndroid("--disk-cache-path");
        opts->SetMode(kReplayServer);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --disk-cache-path <cache-directory>");
        }
        opts->onDiskCacheOptions.path = argv[++i];
      } else if (strcmp(argv[i], "--cleanup-on-disk-cache") == 0) {
        ensureNotAndroid("--cleanup-on-disk-cache");
        opts->onDiskCacheOptions.cleanUp = true;
      } else if (strcmp(argv[i], "--port") == 0) {
        opts->SetMode(kReplayServer);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --port <port_num>");
        }
        opts->portArgStr = argv[++i];
      } else if (strcmp(argv[i], "--log-level") == 0) {
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --log-level <F|E|W|I|D|V>");
        }
        switch (argv[++i][0]) {
          case 'F':
            opts->logLevel = LOG_LEVEL_FATAL;
            break;
          case 'E':
            opts->logLevel = LOG_LEVEL_ERROR;
            break;
          case 'W':
            opts->logLevel = LOG_LEVEL_WARNING;
            break;
          case 'I':
            opts->logLevel = LOG_LEVEL_INFO;
            break;
          case 'D':
            opts->logLevel = LOG_LEVEL_DEBUG;
            break;
          case 'V':
            opts->logLevel = LOG_LEVEL_VERBOSE;
            break;
          default:
            GAPID_FATAL("Usage: --log-level <F|E|W|I|D|V>");
        }
      } else if (strcmp(argv[i], "--log") == 0) {
        warnAndroid("--log");
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --log <log-file-path>");
        }
        opts->logPath = argv[++i];
      } else if (strcmp(argv[i], "--idle-timeout-sec") == 0) {
        opts->SetMode(kReplayServer);
        if (i + 1 >= argc) {
          GAPID_FATAL("Usage: --idle-timeout-sec <timeout in seconds>");
        }
        opts->idleTimeoutSec = atoi(argv[++i]);
      } else if (strcmp(argv[i], "--wait-for-debugger") == 0) {
        opts->waitForDebugger = true;
      } else if (strcmp(argv[i], "--version") == 0) {
        opts->version = true;
      } else if (strcmp(argv[i], "-h") == 0 || strcmp(argv[i], "-help") == 0 ||
                 strcmp(argv[i], "--help") == 0) {
        opts->help = true;
      } else {
        GAPID_FATAL("Unknown argument: %s", argv[i]);
      }
    }
  }

  void SetMode(ReplayMode mode) {
    if (this->mode != kUnknown && this->mode != mode) {
      mode = kConflict;
    }
    this->mode = mode;
  }
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

struct PrewarmData {
  GrpcReplayService* prewarm_service = nullptr;
  Context* prewarm_context = nullptr;
  std::string prewarm_id;
  std::string cleanup_id;
  std::string current_state;
};

// Setup creates and starts a replay server at the given URI port. Returns the
// created and started server.
// Note the given memory manager and the crash handler, they may be used for
// multiple connections, so a mutex lock is passed in to make the accesses to
// to them exclusive to one connected client. All other replay requests from
// other clients will be blocked, until the current replay finishes.
std::unique_ptr<Server> Setup(const char* uri, const char* authToken,
                              ResourceCache* cache, int idleTimeoutSec,
                              core::CrashHandler* crashHandler,
                              MemoryManager* memMgr, PrewarmData* prewarm,
                              std::mutex* lock) {
  // Return a replay server with the following ReplayHandler.
  return Server::createAndStart(
      uri, authToken, idleTimeoutSec,
      [cache, memMgr, crashHandler, lock,
       prewarm](GrpcReplayService* replayConn) {
        // This lambda implements the ReplayHandler. Any error should be
        // reported to GAPIS.  Benign errors (e.g. Vulkan errors collected
        // during a "report" replay) are sent back through replay
        // notifications. All other errors (e.g. failure during priming) should
        // be handled with GAPID_FATAL(): the crash handler will notify GAPIS,
        // which will be aware of the replay failure and will restart the
        // replayer. In any case, do NOT fail silently via an early return,
        // otherwise GAPIS may hang on waiting for a replay response. The only
        // clean termination for this ReplayHandler is to return when there is
        // no more replay requests to process, which reflects the fact that the
        // GAPIS-GAPIR connection has been terminated.

        std::unique_ptr<CrashUploader> crash_uploader =
            std::unique_ptr<CrashUploader>(
                new CrashUploader(*crashHandler, replayConn));

        std::unique_ptr<ResourceLoader> resLoader;
        if (cache == nullptr) {
          resLoader = PassThroughResourceLoader::create(replayConn);
        } else {
          resLoader = CachedResourceLoader::create(
              cache, PassThroughResourceLoader::create(replayConn));
        }

        std::unique_ptr<Context> context =
            Context::create(replayConn, *crashHandler, resLoader.get(), memMgr);
        if (context == nullptr) {
          GAPID_FATAL("Loading Context failed!");
        }

        auto cleanup_state = [&](bool isPrewarm) {
          if (!prewarm->prewarm_context->initialize(prewarm->cleanup_id)) {
            return false;
          }
          if (cache != nullptr) {
            prewarm->prewarm_context->prefetch(cache);
          }
          bool ok = prewarm->prewarm_context->interpret(true, isPrewarm);
          if (!ok) {
            return false;
          }
          if (!prewarm->prewarm_context->cleanup()) {
            return false;
          }
          prewarm->prewarm_id = "";
          prewarm->cleanup_id = "";
          prewarm->current_state = "";
          prewarm->prewarm_context = nullptr;
          prewarm->prewarm_service = nullptr;
          return true;
        };

        auto prime_state = [&](std::string state, std::string cleanup,
                               bool isPrewarm) {
          GAPID_INFO("Priming %s", state.c_str());
          if (context->initialize(state)) {
            GAPID_INFO("Replay context initialized successfully");
          } else {
            GAPID_ERROR("Replay context initialization failed");
            return false;
          }
          if (cache != nullptr) {
            context->prefetch(cache);
          }
          GAPID_INFO("Replay started");
          bool ok = context->interpret(false, isPrewarm);
          GAPID_INFO("Priming %s", ok ? "finished successfully" : "failed");
          if (!ok) {
            return false;
          }

          if (!cleanup.empty()) {
            prewarm->current_state = state;
            prewarm->cleanup_id = cleanup;
            prewarm->prewarm_id = state;
            prewarm->prewarm_service = replayConn;
            prewarm->prewarm_context = context.get();
          }
          return true;
        };

        // Loop on getting and processing replay requests
        do {
          auto req = replayConn->getReplayRequest();
          if (!req) {
            GAPID_INFO("No more requests!");
            break;
          }
          GAPID_INFO("Got request %d", req->req_case());
          switch (req->req_case()) {
            case replay_service::ReplayRequest::kReplay: {
              std::lock_guard<std::mutex> mem_mgr_crash_hdl_lock_guard(*lock);

              if (prewarm->current_state != req->replay().dependent_id()) {
                GAPID_INFO("Trying to get into the correct state");
                cleanup_state(false);
                if (req->replay().dependent_id() != "") {
                  prime_state(req->replay().dependent_id(), "", false);
                }
              } else {
                GAPID_INFO("Already in the correct state");
              }
              GAPID_INFO("Running %s", req->replay().replay_id().c_str());
              if (context->initialize(req->replay().replay_id())) {
                GAPID_INFO("Replay context initialized successfully");
              } else {
                GAPID_FATAL("Replay context initialization failed");
              }
              if (cache != nullptr) {
                context->prefetch(cache);
              }

              GAPID_INFO("Replay started");
              bool ok = context->interpret();
              GAPID_INFO("Replay %s", ok ? "finished successfully" : "failed");
              replayConn->sendReplayFinished();
              if (!context->cleanup()) {
                GAPID_FATAL("Context cleanup failed");
              }
              prewarm->current_state = "";
              if (prewarm->prewarm_service && !prewarm->prewarm_id.empty() &&
                  !prewarm->cleanup_id.empty()) {
                prewarm->prewarm_service->primeState(prewarm->prewarm_id,
                                                     prewarm->cleanup_id);
              }
              break;
            }
            case replay_service::ReplayRequest::kPrewarm: {
              std::lock_guard<std::mutex> mem_mgr_crash_hdl_lock_guard(*lock);
              // We want to pre-warm into the existing state, good deal.
              if (prewarm->current_state == req->prewarm().prerun_id()) {
                GAPID_INFO(
                    "Already primed in the correct state, no more work is "
                    "needed");
                prewarm->cleanup_id = req->prewarm().cleanup_id();
                break;
              }
              if (prewarm->current_state != "") {
                if (!cleanup_state(true)) {
                  GAPID_FATAL(
                      "Could not clean up after previous replay, in a bad "
                      "state now");
                }
              }
              if (!prime_state(std::move(req->prewarm().prerun_id()),
                               std::move(req->prewarm().cleanup_id()), true)) {
                GAPID_FATAL("Could not prime state: in a bad state now");
              }
              break;
            }
            default: { GAPID_FATAL("Unknown replay request type"); }
          }
        } while (true);
      });
}

static int replayArchive(core::CrashHandler* crashHandler,
                         std::unique_ptr<ResourceCache> resourceCache,
                         gapir::ReplayService* replayArchiveService) {
  std::shared_ptr<MemoryAllocator> allocator = createAllocator();

  // The directory consists an archive(resources.{index,data}) and payload.bin.
  MemoryManager memoryManager(allocator);

  std::unique_ptr<ResourceLoader> resLoader =
      CachedResourceLoader::create(resourceCache.get(), nullptr);

  std::unique_ptr<Context> context = Context::create(
      replayArchiveService, *crashHandler, resLoader.get(), &memoryManager);

  if (replayArchiveService->getPayload("payload") == NULL) {
    GAPID_ERROR("Replay payload could not be found.");
  }

  if (context->initialize("payload")) {
    GAPID_DEBUG("Replay context initialized successfully");
  } else {
    GAPID_ERROR("Replay context initialization failed");
    return EXIT_FAILURE;
  }

  GAPID_INFO("Replay started");
  bool ok = context->interpret();
  replayArchiveService->sendReplayFinished();
  if (!context->cleanup()) {
    GAPID_ERROR("Replay cleanup failed");
    return EXIT_FAILURE;
  }
  GAPID_INFO("Replay %s", ok ? "finished successfully" : "failed");

  return ok ? EXIT_SUCCESS : EXIT_FAILURE;
}

}  // anonymous namespace

#if TARGET_OS == GAPID_OS_ANDROID

namespace {

const char* kReplayAssetToDetect = "replay_export/resources.index";

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

void android_process(struct android_app* app, int32_t cmd) {
  switch (cmd) {
    case APP_CMD_INIT_WINDOW: {
      gapir::android_window = app->window;
      GAPID_DEBUG("Received window: %p\n", gapir::android_window);
      break;
    }
  }
}

// Extract command line arguments from the extra of Android intent:
//   adb shell am start -n <...> -e gapir-launch-args "'list of arguments to be
//   extracted'"
// Note the quoting: from host terminal adb command, we need to double-escape
// the extra args string, as it is first quoted by host terminal emulator
// (e.g. bash), then it must be quoted for the on-device shell.
std::vector<std::string> getArgsFromIntents(struct android_app* app,
                                            Options* opts) {
  assert(app != nullptr);
  assert(opts != nullptr);

  // The extra flag to indicate arguments
  const char* intent_flag = "gapir-intent-flag";

  JNIEnv* env;
  app->activity->vm->AttachCurrentThread(&env, 0);

  // Select replay archive mode if replay assets are detected
  jobject j_asset_manager = jni_call_o(env, app->activity->clazz, "getAssets",
                                       "()Landroid/content/res/AssetManager;");
  AAssetManager* asset_manager = AAssetManager_fromJava(env, j_asset_manager);

  AAsset* asset = AAssetManager_open(asset_manager, kReplayAssetToDetect,
                                     AASSET_MODE_UNKNOWN);

  if (asset != nullptr) {
    opts->SetMode(kReplayArchive);
    AAsset_close(asset);
  } else {
    opts->SetMode(kReplayServer);
  }

  jobject intent = jni_call_o(env, app->activity->clazz, "getIntent",
                              "()Landroid/content/Intent;");

  jmethodID get_string_extra_method_id =
      env->GetMethodID(env->GetObjectClass(intent), "getStringExtra",
                       "(Ljava/lang/String;)Ljava/lang/String;");

  jvalue get_string_extra_args;
  get_string_extra_args.l = env->NewStringUTF(intent_flag);

  jstring extra_jstring = static_cast<jstring>(env->CallObjectMethodA(
      intent, get_string_extra_method_id, &get_string_extra_args));

  std::string extra_string;
  if (extra_jstring) {
    const char* extra_cstr = env->GetStringUTFChars(extra_jstring, nullptr);
    extra_string = extra_cstr;
    env->ReleaseStringUTFChars(extra_jstring, extra_cstr);
    env->DeleteLocalRef(extra_jstring);
  }

  env->DeleteLocalRef(get_string_extra_args.l);
  env->DeleteLocalRef(intent);

  app->activity->vm->DetachCurrentThread();

  // Prepare arguments with a value for argv[0], as gflags expects
  std::vector<std::string> args;
  args.push_back(
      "gapir");  // POSIX says the first argv should be the program being run.
                 // Lets inject a placeholder here for compatibility.

  // Split extra_string
  std::stringstream ss(extra_string);
  std::string arg;

  while (std::getline(ss, arg, ' ')) {
    if (!arg.empty()) {
      args.push_back(arg);
    }
  }

  return args;
}

std::string getCacheDir(struct android_app* app) {
  JNIEnv* env;
  app->activity->vm->AttachCurrentThread(&env, 0);

  jobject cache_dir_jobject =
      jni_call_o(env, app->activity->clazz, "getCacheDir", "()Ljava/io/File;");
  jmethodID get_absolute_path_method_id =
      env->GetMethodID(env->GetObjectClass(cache_dir_jobject),
                       "getAbsolutePath", "()Ljava/lang/String;");

  jstring cache_dir_jstring = (jstring)env->CallObjectMethod(
      cache_dir_jobject, get_absolute_path_method_id);

  std::string cache_dir_string;
  if (cache_dir_jstring) {
    const char* cache_dir_cstr =
        env->GetStringUTFChars(cache_dir_jstring, nullptr);
    cache_dir_string = cache_dir_cstr;
    env->ReleaseStringUTFChars(cache_dir_jstring, cache_dir_cstr);
    env->DeleteLocalRef(cache_dir_jstring);
  }

  app->activity->vm->DetachCurrentThread();

  return cache_dir_string;
}

}  // namespace

// Main function for android
void android_main(struct android_app* app) {
  // Start up in verbose mode until we have parsed any flags passed.
  GAPID_LOGGER_INIT(LOG_LEVEL_VERBOSE, "gapir", "");

  Options opts;

  std::vector<std::string> args = getArgsFromIntents(app, &opts);
  Options::Parse(args, &opts);

  if (opts.waitForDebugger) {
    GAPID_INFO("Waiting for debugger to attach");
    core::Debugger::waitForAttach();
  }

  if (opts.help) {
    Options::PrintHelp();
    return;
  } else if (opts.version) {
    GAPID_INFO("GAPIR version " AGI_VERSION_AND_BUILD "\n");
    return;
  } else if (opts.mode == kConflict) {
    GAPID_ERROR("Argument conflicts.");
    return;
  }

  // Restart logging with the correct level now that we've parsed the args.
  GAPID_LOGGER_INIT(opts.logLevel, "gapir", opts.logPath);

  CrashHandler crashHandler(getCacheDir(app));

  ANativeActivity_setWindowFlags(app->activity, AWINDOW_FLAG_KEEP_SCREEN_ON, 0);

  std::thread waiting_thread;
  std::atomic<bool> thread_is_done(false);

  // Get the path of the file system socket.
  std::string internal_data_path = std::string(app->activity->internalDataPath);
  std::string socket_file_path = internal_data_path + "/" + kSocketName;
  std::string uri = std::string("unix://") + socket_file_path;
  std::unique_ptr<Server> server = nullptr;
  std::shared_ptr<MemoryAllocator> allocator = createAllocator();
  MemoryManager memoryManager(allocator);
  auto cache =
      InMemoryResourceCache::create(allocator, allocator->getTotalSize());
  std::mutex lock;
  PrewarmData data;

  if (opts.mode == kReplayArchive) {
    GAPID_INFO("Started Graphics API Replay from archive.");

    waiting_thread = std::thread([&]() {
      // It's important to use a different JNIEnv as it is a separate thread
      JNIEnv* env;
      app->activity->vm->AttachCurrentThread(&env, 0);

      // Keep a jobject reference in the main thread to prevent garbage
      // collection of the asset manager.
      // https://developer.android.com/ndk/reference/group/asset#aassetmanager_fromjava
      jobject j_asset_manager =
          jni_call_o(env, app->activity->clazz, "getAssets",
                     "()Landroid/content/res/AssetManager;");
      AAssetManager* asset_manager =
          AAssetManager_fromJava(env, j_asset_manager);

      std::unique_ptr<ResourceCache> assetResourceCache =
          AssetResourceCache::create(asset_manager);
      gapir::AssetReplayService assetReplayService(asset_manager);

      replayArchive(&crashHandler, std::move(assetResourceCache),
                    &assetReplayService);

      app->activity->vm->DetachCurrentThread();

      thread_is_done = true;
    });

  } else if (opts.mode == kReplayServer) {
    GAPID_INFO(
        "Started Graphics API Replay daemon.\n"
        "Listening on unix socket '%s'\n"
        "Supported ABIs: %s\n",
        uri.c_str(), core::supportedABIs());

    server =
        Setup(uri.c_str(), opts.authToken.c_str(), cache.get(),
              opts.idleTimeoutSec, &crashHandler, &memoryManager, &data, &lock);
    waiting_thread = std::thread([&]() {
      server.get()->wait();
      thread_is_done = true;
    });
    if (chmod(socket_file_path.c_str(),
              S_IRUSR | S_IWUSR | S_IROTH | S_IWOTH)) {
      GAPID_ERROR("Chmod failed!");
    }
  } else {
    GAPID_ERROR("Invalid replay mode");
  }

  app->onAppCmd = android_process;

  bool finishing = false;
  bool alive = true;
  while (alive) {
    int fdesc;
    int events;
    const int timeoutMilliseconds = 1000;
    struct android_poll_source* source;
    while (ALooper_pollAll(timeoutMilliseconds, &fdesc, &events,
                           (void**)&source) >= 0) {
      // process this event
      if (source) {
        source->process(app, source);
      }
      if (app->destroyRequested) {
        // Clean up and exit the main loop
        if (opts.mode == kReplayServer) {
          server->shutdown();
        }
        alive = false;
        break;
      }
    }

    if (thread_is_done && !finishing) {
      // Start termination of the app
      ANativeActivity_finish(app->activity);

      // Note that we need to keep on polling events, eventually APP_CMD_DESTROY
      // will pop-up after which app->destroyRequested will be true, enabling us
      // to properly exit the main loop.

      // Meanwhile, remember that we are finishing to avoid calling
      // ANativeActivity_finish() several times.
      finishing = true;
    }
  }

  // Final clean up
  waiting_thread.join();
  if (opts.mode == kReplayServer) {
    unlink(socket_file_path.c_str());
  }
  GAPID_INFO("End of Graphics API Replay");
  return;
}

#else  // TARGET_OS == GAPID_OS_ANDROID

namespace {

// createCache constructs and returns a ResourceCache based on the given
// onDiskCacheOpts. If on-disk cache is not enabled or not possible to create,
// an in-memory cache will be built and returned. If on-disk cache is created
// in a temporary directory or onDiskCacheOpts is specified to clear cache
// files, a monitor process will be forked to delete the cache files when the
// main GAPIR VM process ends.
std::unique_ptr<ResourceCache> createCache(
    const Options::OnDiskCache& onDiskCacheOpts,
    std::shared_ptr<MemoryAllocator> allocator) {
#if TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
  if (!onDiskCacheOpts.enabled) {
    return InMemoryResourceCache::create(allocator, allocator->getTotalSize());
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
    return InMemoryResourceCache::create(allocator, allocator->getTotalSize());
  }
  auto onDiskCache =
      OnDiskResourceCache::create(onDiskCachePath, cleanUpOnDiskCache);
  if (onDiskCache == nullptr) {
    GAPID_WARNING(
        "On-disk cache creation failed, fallback to use in-memory cache");
    return InMemoryResourceCache::create(allocator, allocator->getTotalSize());
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

  return onDiskCache;
#else   // TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
  if (onDiskCacheOpts.enabled) {
    GAPID_WARNING(
        "On-disk cache not supported, fallback to use in-memory cache");
  }
#endif  // TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
  // Just use the in-memory cache
  return InMemoryResourceCache::create(allocator, allocator->getTotalSize());
}
}  // namespace

static int startServer(core::CrashHandler* crashHandler, Options opts) {
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

  std::shared_ptr<MemoryAllocator> allocator = createAllocator();
  MemoryManager memoryManager(allocator);

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

  auto cache = createCache(opts.onDiskCacheOptions, allocator);

  std::mutex lock;
  PrewarmData data;
  std::unique_ptr<Server> server =
      Setup(uri.c_str(), (authToken.size() > 0) ? authToken.data() : nullptr,
            cache.get(), opts.idleTimeoutSec, crashHandler, &memoryManager,
            &data, &lock);
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
  Options opts;
  Options::Parse(argc, argv, &opts);

#if TARGET_OS == GAPID_OS_LINUX
  // Ignore SIGPIPE so we can log after gapis closes.
  signal(SIGPIPE, SIG_IGN);
#endif

  if (opts.waitForDebugger) {
    GAPID_INFO("Waiting for debugger to attach");
    core::Debugger::waitForAttach();
  }

  if (opts.help) {
    Options::PrintHelp();
    return EXIT_SUCCESS;
  } else if (opts.version) {
    printf("GAPIR version " AGI_VERSION_AND_BUILD "\n");
    return EXIT_SUCCESS;
  } else if (opts.mode == kConflict) {
    GAPID_ERROR("Argument conflicts.");
    return EXIT_FAILURE;
  }

  core::CrashHandler crashHandler;
  GAPID_LOGGER_INIT(opts.logLevel, "gapir", opts.logPath);

  if (opts.mode == kReplayArchive) {
    std::string payloadPath = std::string(opts.replayArchive) + "/payload.bin";
    gapir::ArchiveReplayService replayArchiveService(payloadPath,
                                                     opts.postbackDirectory);
    // All the resource data must be in the archive file, no fallback resource
    // loader to fetch uncached resources data.
    auto onDiskCache = OnDiskResourceCache::create(opts.replayArchive, false);
    return replayArchive(&crashHandler, std::move(onDiskCache),
                         &replayArchiveService);
  } else {
    return startServer(&crashHandler, opts);
  }
}

#endif  // TARGET_OS == GAPID_OS_ANDROID
