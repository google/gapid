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

#ifndef DEVICEINFO_ANDROID_JNI_HELPERS_H
#define DEVICEINFO_ANDROID_JNI_HELPERS_H

#include <jni.h>

#include <string>
#include <vector>

// Class is a wrapper around a JNIEnv and class name and offers methods for
// getting fields.
class Class {
 public:
  inline Class(JNIEnv* env, const char* name)
      : mEnv(env), mClass(env->FindClass(name)) {}
  template <typename T>
  inline bool get_field(const char* name, T& out);

 private:
  inline void convString(jstring str, std::string&);

  JNIEnv* mEnv;
  jclass mClass;
};

template <>
inline bool Class::get_field(const char* name, std::string& out) {
  if (mClass == nullptr) {
    return false;
  }
  auto id = mEnv->GetStaticFieldID(mClass, name, "Ljava/lang/String;");
  jboolean flag = mEnv->ExceptionCheck();
  if (flag) {
    mEnv->ExceptionClear();
    return false;
  }
  if (id == nullptr) {
    return false;
  }
  auto str = reinterpret_cast<jstring>(mEnv->GetStaticObjectField(mClass, id));
  convString(str, out);
  return true;
}

template <>
inline bool Class::get_field(const char* name, std::vector<std::string>& out) {
  if (mClass == nullptr) {
    return false;
  }
  auto id = mEnv->GetStaticFieldID(mClass, name, "[Ljava/lang/String;");
  jboolean flag = mEnv->ExceptionCheck();
  if (flag) {
    mEnv->ExceptionClear();
    return false;
  }
  if (id == nullptr) {
    return false;
  }
  auto arr =
      reinterpret_cast<jobjectArray>(mEnv->GetStaticObjectField(mClass, id));
  auto len = mEnv->GetArrayLength(arr);
  out.resize(len);
  for (int i = 0; i < len; i++) {
    auto str = reinterpret_cast<jstring>(mEnv->GetObjectArrayElement(arr, i));
    if (str == nullptr) {
      return false;
    }
    convString(str, out[i]);
  }
  return true;
}

template <>
inline bool Class::get_field(const char* name, int& out) {
  if (mClass == nullptr) {
    return false;
  }
  auto id = mEnv->GetStaticFieldID(mClass, name, "I");
  jboolean flag = mEnv->ExceptionCheck();
  if (flag) {
    mEnv->ExceptionClear();
    return false;
  }
  if (id == nullptr) {
    return false;
  }
  out = mEnv->GetStaticIntField(mClass, id);
  return true;
}

void Class::convString(jstring str, std::string& out) {
  auto chars = mEnv->GetStringUTFChars(str, nullptr);
  out = chars;
  mEnv->ReleaseStringUTFChars(str, chars);
}

#endif  // DEVICEINFO_ANDROID_JNI_HELPERS_H
