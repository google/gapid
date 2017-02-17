// Copyright (c) 2016 Google Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and/or associated documentation files (the
// "Materials"), to deal in the Materials without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Materials, and to
// permit persons to whom the Materials are furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Materials.
//
// MODIFICATIONS TO THIS FILE MAY MEAN IT NO LONGER ACCURATELY REFLECTS
// KHRONOS STANDARDS. THE UNMODIFIED, NORMATIVE VERSIONS OF KHRONOS
// SPECIFICATIONS AND HEADER INFORMATION ARE LOCATED AT
//    https://www.khronos.org/registry/
//
// THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.

#ifndef LIBSPIRV_TOOLS_IO_H_
#define LIBSPIRV_TOOLS_IO_H_

#include <cstdint>
#include <cstdio>
#include <vector>

// Appends the content from the file named as |filename| to |data|, assuming
// each element in the file is of type |T|. The file is opened with the given
// |mode|. If |filename| is nullptr or "-", reads from the standard input. If
// any error occurs, writes error messages to standard error and returns false.
template <typename T>
bool ReadFile(const char* filename, const char* mode, std::vector<T>* data) {
  const int buf_size = 1024;
  const bool use_file = filename && strcmp("-", filename);
  if (FILE* fp = (use_file ? fopen(filename, mode) : stdin)) {
    T buf[buf_size];
    while (size_t len = fread(buf, sizeof(T), buf_size, fp)) {
      data->insert(data->end(), buf, buf + len);
    }
    if (ftell(fp) == -1L) {
      if (ferror(fp)) {
        fprintf(stderr, "error: error reading file '%s'\n", filename);
        return false;
      }
    } else {
      if (sizeof(T) != 1 && (ftell(fp) % sizeof(T))) {
        fprintf(stderr, "error: corrupted word found in file '%s'\n", filename);
        return false;
      }
    }
    if (use_file) fclose(fp);
  } else {
    fprintf(stderr, "error: file does not exist '%s'\n", filename);
    return false;
  }
  return true;
}

// Writes the given |data| into the file named as |filename| using the given
// |mode|, assuming |data| is an array of |count| elements of type |T|. If
// |filename| is nullptr or "-", writes to standard output. If any error occurs,
// returns false and outputs error message to standard error.
template <typename T>
bool WriteFile(const char* filename, const char* mode, const T* data,
               size_t count) {
  const bool use_stdout =
      !filename || (filename[0] == '-' && filename[1] == '\0');
  if (FILE* fp = (use_stdout ? stdout : fopen(filename, mode))) {
    size_t written = fwrite(data, sizeof(T), count, fp);
    if (count != written) {
      fprintf(stderr, "error: could not write to file '%s'\n", filename);
      return false;
    }
    if (!use_stdout) fclose(fp);
  } else {
    fprintf(stderr, "error: could not open file '%s'\n", filename);
    return false;
  }
  return true;
}

#endif  // LIBSPIRV_TOOLS_IO_H_
