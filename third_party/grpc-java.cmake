# Copyright (C) 2017 Google Inc.
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

set(grpc_java_dir "third_party/grpc-java")
set(protoc_gen_grpc_java_dir "${grpc_java_dir}/compiler/src/java_plugin/cpp")

set(protoc_gen_grpc_java_srcs
    "java_generator.cpp"
    "java_plugin.cpp"
)

if(NOT DISABLED_CXX)
    abs_list(protoc_gen_grpc_java_srcs ${protoc_gen_grpc_java_dir})
    add_executable(protoc-gen-grpc-java ${protoc_gen_grpc_java_srcs})
    target_link_libraries(protoc-gen-grpc-java protoc_lib)
endif()

