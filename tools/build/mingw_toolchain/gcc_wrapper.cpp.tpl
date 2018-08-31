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

// Wrapper around MinGW's gcc. gcc can handle long file names as long as they
// use the //?/ prefix. This wrapper processes the arguments, and changes
// any file paths to be absolute using the //?/ prefix. Linker param script
// files are also processed.

#include <string>
#include <windows.h>

#include "file_collector.h"

int main(int argc, char** argv) {
  FileCollector files("gcc");
  ArgumentCollector arguments("gcc");

  bool fixup = false, output = false, libstdc = false;
  for (int i = 1; i < argc; i++) {
    std::string arg(argv[i]);

    if (fixup) {
      fixup = false;
      arguments.Push(files.Fixup(arg, false));
      if (output) {
        // If the output allready exists, gcc munges the path when it invokes ld.
        DeleteFile(arg.c_str());
        output = false;
      }
    } else if (arg.compare(0, 5, "-Wl,@") == 0) {
      arguments.Push("-Wl,@" + files.ProcessParamsFile(arg.substr(5), FileCollector::GCC));
    } else if (arg == "-lstdc++") {
      // Move any -lstdc++ to the end. This is to work around how rules_go/golang calls us.
      // Ugly, dirty, and I hate it, but it works.
      libstdc = true;
    } else {
      arguments.Push(arg);
      output = arg == "-o";
      fixup = output || arg == "-MF";
    }
  }

  if (libstdc) {
    arguments.Push("-lstdc++");
  }

  return arguments.Execute("%{CC}", &files);
}
