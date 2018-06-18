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

#ifndef CORE_DL_LOADER_H
#define CORE_DL_LOADER_H

namespace core {

// Utility class for retrieving function pointers from dynamic libraries.
class DlLoader {
 public:
  // Loads the dynamic library specified by the given name and fallback names
  // (if any). Names will be used to try to find the library in order. If the
  // library cannot be loaded then this is a fatal error. For *nix systems,
  // a nullptr can be used to search the application's functions.
  template <typename... ConstCharPtrs>
  DlLoader(const char* name, ConstCharPtrs... fallback_names);

  // Unloads the library loaded in the constructor.
  ~DlLoader();

  // Looks up the function with the specified name from the library.
  // Returns nullptr if the function is not found.
  void* lookup(const char* name);

  // can_load checks if the dynamic library specified with the given name can
  // be loaded. Returns true if so, otherwise returns false.
  static bool can_load(const char* lib_name);

 private:
  DlLoader() = default;
  DlLoader(const DlLoader&) = delete;
  DlLoader& operator=(const DlLoader&) = delete;

  void* mLibrary;
};

}  // namespace core

#endif  // CORE_DL_LOADER_H
