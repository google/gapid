# SPIR-V Headers

This repository contains machine-readable files from the
[SPIR-V Registry](https://www.khronos.org/registry/spir-v/).
This includes:

* Header files for various languages.
* JSON files describing the grammar for the SPIR-V core instruction set,
  and for the GLSL.std.450 extended instruction set.
* The XML registry file.

Under the [include](include) directory, header files are provided according to
their own version.  Only major and minor version numbers count.
For example, the headers for SPIR-V 1.1 are in
[include/spirv/1.1](include/spirv/1.1).  Also, the headers for the 1.0 versions
of the GLSL.std.450 and OpenCL extended instruction sets are in
[include/spirv/1.0](include/spirv/1.0).

In contrast, the XML registry file has a linear history, so it is
not tied to SPIR-V specification versions.

## How is this repository updated?

When a new version or revision of the SPIR-V header files are published,
the SPIR Working Group will push new commits onto master, updating
the files under [include](include).
A newer revision of a header file always replaces an older revision of
the same version.  For example, verison 1.0 Rev 4 of `spirv.h` is placed in
`include/spirv/1.0/spirv.h` and if there is a Rev 5, then it will be placed
in the same location.

The SPIR-V XML registry file is updated by the Khronos registrar whenever
a new enum range is allocated.

In particular, pull requests that update header files will not be accepted.
Issues with the header files should be filed in the [issue
tracker](https://github.com/KhronosGroup/SPIRV-Headers/issues).

## How to install the headers

```
mkdir build
cd build
cmake ..
# Linux
cmake --build . --target install-headers
# Windows
cmake --build . --config Debug --target install-headers
```

Then, for example, you will have `/usr/local/include/spirv/1.0/spirv.h`

If you want to install them somewhere else, then use
`-DCMAKE_INSTALL_PREFIX=/other/path` on the first `cmake` command.

## Using the headers without installing

A CMake-based project can use the headers without installing, as follows:

1. Add an `add_subdirectory` directive to include this source tree.
2. Use `${SPIRV-Headers_SOURCE_DIR}/include}` in a `target_include_directories`
   directive.
3. In your C or C++ source code use `#include` directives that explicitly mention
   the `spirv` path component.  For example the following uses SPIR-V 1.1
   core instructions, and the 1.0 versions of the GLSL.std.450 and OpenCL
   extended instructions.
```
#include "spirv/1.0/GLSL.std.450.h"
#include "spirv/1.0/OpenCL.std.h"
#include "spirv/1.1/spirv.hpp"
```

See also the [example](example/) subdirectory.  But since that example is
*inside* this repostory, it doesn't use and `add_subdirectory` directive.

## FAQ

* *How are different versions published?*

  All versions are present in the master branch of the repository.
  They are located in different subdirectories under
  [include/spirv](include/spirv), where the subdirectory at that
  level encodes the major and minor version number of the relevant spec.

  An application should consciously select the targeted spec version
  number, by naming the specific version in its `#include` directives,
  as above and in the examples.

* *How do you handle the evolution of extended instruction sets?*

  Extended instruction sets evolve asynchronously from the core spec.
  Right now there is only a single version of both the GLSL and OpenCL
  headers.  So we don't yet have a problematic example to resolve.

## License
<a name="license"></a>
```
Copyright (c) 2015-2016 The Khronos Group Inc.

Permission is hereby granted, free of charge, to any person obtaining a
copy of this software and/or associated documentation files (the
"Materials"), to deal in the Materials without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Materials, and to
permit persons to whom the Materials are furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be included
in all copies or substantial portions of the Materials.

MODIFICATIONS TO THIS FILE MAY MEAN IT NO LONGER ACCURATELY REFLECTS
KHRONOS STANDARDS. THE UNMODIFIED, NORMATIVE VERSIONS OF KHRONOS
SPECIFICATIONS AND HEADER INFORMATION ARE LOCATED AT
   https://www.khronos.org/registry/

THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.
```
