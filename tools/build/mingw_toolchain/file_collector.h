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

#include <string>
#include <vector>

// Keeps track of temporary files and processes @ params files
class FileCollector {
 public:
  FileCollector(const std::string& baseName);

  // Type of params files expected.
  enum ParamsType {
    GCC,  // params files only contain inputs and support \\?\ paths.
    AR,  // first file parameter is output and params file does not support \\?\
         // paths.
  };

  // Processes the @ param file specified by path and returns the path
  // to the substitute param file to use.
  std::string ProcessParamsFile(const std::string& path, ParamsType type);

  // Returns path as a //?/ prefixed absolute path, optionally escaping
  // '\' characters if the result is to be used in a @ param file.
  std::string Fixup(const std::string& path, bool escape);

  // Copies any output files from their temporary location to the final
  // expected location. Call this after invoking the wrapped binary.
  void CopyOutputs();

  // Deletes any created temporary files.
  void Cleanup();

 private:
  // Returns the path to a new temporary @ params file.
  const std::string NewParamsFile();

  // Handles a line/path read from a @ params file. p should contain the
  // index of the last '/' in path. If isOutput is true, the path is
  // processed as an output of the wrapped binary, otherwhise as an input.
  std::string HandleParam(const std::string& path, std::string::size_type p,
                          bool isOutput);

  struct Output {
    std::string final_;
    std::string temp_;
  };

  // Name of the wrapped binary, used for constructing temp paths.
  const std::string baseName_;
  // The current working dir, used to make relative paths absolute.
  const std::string cwd_;
  // Temporary directory to create any files in.
  std::string tmpDir_;
  // Used as a unique ID when constructing temp files.
  int counter_;
  // List of all temp files to be cleaned up.
  std::vector<std::string> tmpFiles_;
  // List of all outputs expected from the wrapped binary and their
  // final expected location.
  std::vector<Output> outputs_;
};

// Collects arguments to the wrapped binary.
class ArgumentCollector {
 public:
  // cmd is the name of the wrapped binary and will be arg[0].
  ArgumentCollector(const std::string& cmd);

  // Pushes the given argument to the argument list.
  void Push(const std::string& arg);

  // Executes the wrapped binary of the given path and returns its return
  // value. Invokes the cleanup functions of the optional file collector
  // only if the return code is 0.
  int Execute(const std::string exe, FileCollector* fc);

 private:
  std::vector<std::string> arguments_;
};
