/*
 * Copyright (C) 2018 Google Inc.
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

// Wrapper around MinGW's ar utility. ar cannot handle long file names, even
// with the //?/ prefix, so we copy the .o files to a temporary directory,
// archive them there and then move the result into place.

#include <string>

#include "file_collector.h"

int main(int argc, char** argv) {
  FileCollector files("ar");
  ArgumentCollector arguments("ar");

  for (int i = 1; i < argc; i++) {
    std::string arg(argv[i]);
    if (arg[0] == '@') {
      arguments.Push("@" + files.ProcessParamsFile(arg.substr(1), FileCollector::AR));
    } else {
      arguments.Push(arg);
    }
  }

  return arguments.Execute("%{AR}", &files);
}
