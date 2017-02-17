# Copyright (C) 2016 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

include_directories(${CMAKE_CURRENT_SOURCE_DIR}/SPIRV-Tools/external/spirv-headers/include)
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/SPIRV-Tools/include)
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/SPIRV-Tools/source)
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/glslang/OGLCompilersDLL)

glob_all_dirs()

glob(sources
  PATH
    SPIRV-Cross
    SPIRV-Tools/source
    SPIRV-Tools/source/opt
    glslang/OGLCompilersDLL
    glslang/SPIRV
    glslang/glslang/GenericCodeGen
    glslang/glslang/MachineIndependent
    glslang/glslang/MachineIndependent/preprocessor
    glslang/hlsl
  INCLUDE ".cpp$"
  EXCLUDE "enum_set.cpp$"
)

if(WIN32)
  glob(os_sources
    PATH glslang/glslang/OSDependent/Windows
    INCLUDE ".cpp$"
  )
elseif(UNIX)
  glob(os_sources
    PATH glslang/glslang/OSDependent/Unix
    INCLUDE ".cpp$"
  )
else(WIN32)
  message("unknown platform")
endif(WIN32)

add_library(khronos STATIC ${sources} ${os_sources})
