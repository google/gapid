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

#include <android/asset_manager.h>
#include <android/asset_manager_jni.h>
#include <android/native_window_jni.h>
#include <jni.h>
#include <pthread.h>

#include "cube.h"

static AndroidAppState state;
static Cube cube;
static pthread_t thread;

static void *startCubes(void* state) {
    AndroidAppState* appState = (AndroidAppState*)state;
    appState->running = true;
    cube.Run(appState);
    appState->running = false;
    appState->destroyRequested = false;
    return nullptr;
}

extern "C" JNIEXPORT void JNICALL
Java_com_google_android_gapid_VKCubeActivity_nStartCube(JNIEnv* env, jobject clazz, jobject surface,
                                                jobject assetManager) {
  if (!surface || state.running) {
    return;
  }
  state.window = ANativeWindow_fromSurface(env, surface);
  state.assetManager = AAssetManager_fromJava(env, assetManager);
  env->GetJavaVM(&state.vm);
  pthread_create(&thread, nullptr, startCubes, &state);
}

extern "C" JNIEXPORT void JNICALL
Java_com_google_android_gapid_VKCubeActivity_nStopCube(JNIEnv* env, jobject clazz) {
  if (state.running) {
    state.destroyRequested = true;
    pthread_join(thread, nullptr);
  }
}
