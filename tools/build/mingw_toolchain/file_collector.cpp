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

#include "file_collector.h"

#include <direct.h>
#include <shlwapi.h>
#include <windows.h>

#include <algorithm>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

namespace {

std::string GetCWD() {
  char* cwd = _getcwd(nullptr, 0);
  if (cwd == nullptr) {
    std::cerr << "Failed to get workding dir" << std::endl;
  }
  std::string result = R"(\\?\)" + std::string(cwd);
  if (result.back() != '\\') {
    result.push_back('\\');
  }
  delete cwd;
  return result;
}

// Attempts to create a new folder in the temporary folder based off the PID.
std::string GetTempDir(const std::string& baseName) {
  char buffer[(MAX_PATH + 1) + /* max return value of GetTempPathA */
              baseName.length() + 6 /*_XXXX\ */ + 1 /* terminator */];
  int len = GetTempPathA(sizeof(buffer), buffer);

  for (int i = 0, suffix = GetCurrentProcessId(); i < 10000; i++, suffix++) {
    snprintf(buffer + len, sizeof(buffer) - len, "%s_%04x\\", baseName.c_str(),
             suffix % 0x10000);
    if (CreateDirectory(buffer, nullptr)) {
      return std::string(buffer);
    } else if (GetLastError() != ERROR_ALREADY_EXISTS) {
      std::cerr << "Failed to create temporary directory " << buffer
                << std::endl;
      abort();
    }
  }

  std::cerr << "Couldn't find a unique temp dir" << std::endl;
  abort();
  return "";
}

// Copies source to target, optinally munging the target file name if
// a file with the same name already exists at that location.
std::string Copy(const std::string& source, const std::string& target) {
  std::string result = target;
  if (PathFileExists(target.c_str())) {
    std::string::size_type p = target.rfind(".");
    std::string base;
    if (p == std::string::npos) {
      base = target;
    } else {
      base = target.substr(0, p);
    }

    int s = 1;
    while (true) {
      std::stringstream ss;
      ss << base << '_' << s++;
      if (p != std::string::npos) {
        ss << target.substr(p);
      }
      result = ss.str();
      if (!PathFileExists(result.c_str())) {
        break;
      }
    }
  }

  if (!CopyFile(source.c_str(), result.c_str(), true)) {
    std::cerr << "Failed to copy file " << source << " to " << result
              << std::endl;
    abort();
  }

  // Make the temporary files writtable, so they can be deleted later.
  int attr = GetFileAttributes(result.c_str());
  if (attr != INVALID_FILE_ATTRIBUTES && (attr & FILE_ATTRIBUTE_READONLY)) {
    SetFileAttributes(result.c_str(), attr & ~FILE_ATTRIBUTE_READONLY);
  }
  return result;
}

}  // namespace

FileCollector::FileCollector(const std::string& baseName)
    : baseName_(baseName), cwd_(GetCWD()), tmpDir_(""), counter_(0) {}

std::string FileCollector::ProcessParamsFile(const std::string& path,
                                             ParamsType type) {
  std::ifstream input(path);
  if (!input.is_open()) {
    std::cerr << "Failed to open params file " << path << std::endl;
    abort();
  }

  std::string params = NewParamsFile();
  std::ofstream output(params);
  if (!output.is_open()) {
    std::cerr << "Failed to open output file " << params << std::endl;
    abort();
  }

  std::string line;
  for (int i = 0; std::getline(input, line); i++) {
    std::string::size_type p = line.rfind("/");
    if (p == std::string::npos || line[0] == '-') {
      output << line << std::endl;
      continue;
    }

    if (type == AR) {
      output << HandleParam(line, p, i == 1) << std::endl;
    } else {
      output << Fixup(line, true) << std::endl;
    }
  }
  input.close();
  output.close();

  return params;
}

std::string FileCollector::Fixup(const std::string& path, bool escape) {
  std::string result = path;
  std::replace(result.begin(), result.end(), '/', '\\');
  if (result.length() > 3 && result[1] == ':' && result[2] == '\\') {
    result.insert(0, R"(\\?\)");
  } else {
    result.insert(0, cwd_);
  }

  if (escape) {
    for (auto p = result.find_last_of('\\'); p != std::string::npos;
         p = result.find_last_of('\\', p - 1)) {
      result.insert(p, "\\");
      if (p == 0) {
        break;
      }
    }
  }
  return result;
}

void FileCollector::CopyOutputs() {
  for (auto it = outputs_.begin(); it != outputs_.end(); it++) {
    if (!CopyFile(it->temp_.c_str(), it->final_.c_str(), false)) {
      std::cerr << "Failed to copy file " << it->temp_ << " to " << it->final_
                << std::endl;
      abort();
    }
  }
}

void FileCollector::Cleanup() {
  for (auto it = tmpFiles_.begin(); it != tmpFiles_.end(); it++) {
    DeleteFile(it->c_str());
  }
  tmpFiles_.clear();

  if (tmpDir_ != "") {
    RemoveDirectory(tmpDir_.c_str());
    tmpDir_.clear();
  }
}

const std::string FileCollector::NewParamsFile() {
  if (tmpDir_ == "") {
    tmpDir_ = GetTempDir(baseName_);
  }

  std::stringstream ss;
  ss << tmpDir_ << "params_" << ++counter_ << ".params";
  tmpFiles_.push_back(ss.str());
  return tmpFiles_.back();
}

std::string FileCollector::HandleParam(const std::string& path,
                                       std::string::size_type p,
                                       bool isOutput) {
  if (tmpDir_ == "") {
    tmpDir_ = GetTempDir(baseName_);
  }

  std::string source;
  if (path.length() > 3 && path[1] == ':' && path[2] == '/') {
    source = path;
  } else {
    source = cwd_ + path;
  }
  std::replace(source.begin(), source.end(), '/', '\\');

  std::string target;
  if (isOutput) {
    std::stringstream ss;
    ss << tmpDir_ << "output_" << ++counter_ << ".a";
    target = ss.str();
    outputs_.push_back({source, target});
  } else {
    target = Copy(source, tmpDir_ + path.substr(p + 1));
  }

  tmpFiles_.push_back(target);
  std::replace(target.begin(), target.end(), '\\', '/');
  return target;
}

ArgumentCollector::ArgumentCollector(const std::string& cmd) { Push(cmd); }

void ArgumentCollector::Push(const std::string& arg) {
  std::string el(arg);
  // When spawning the wrapped binary, Windows eats quotes. Escape them.
  for (auto p = el.find_last_of('"'); p != std::string::npos;
       p = el.find_last_of('"', p - 1)) {
    el.insert(p, "\\");
    if (p == 0) {
      break;
    }
  }
  arguments_.push_back(std::move(el));
}

int ArgumentCollector::Execute(const std::string exe, FileCollector* fc) {
  std::vector<const char*> argv;
  for (auto it = arguments_.begin(); it != arguments_.end(); ++it) {
    argv.push_back(it->c_str());
  }
  argv.push_back(nullptr);

  int r = spawnv(_P_WAIT, exe.c_str(), const_cast<char**>(argv.data()));
  if (r == 0 && fc != nullptr) {
    fc->CopyOutputs();
    fc->Cleanup();
  }
  return r;
}
