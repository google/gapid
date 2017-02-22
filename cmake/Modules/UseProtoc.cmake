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

set(PROTO_PATH
    "${ROOT_DIR}/src"
    "${CMAKE_SOURCE_DIR}/third_party/src"
    "${PROTOBUF_DIR}/src"
)

set(PROTO_HEADER "# Copyright (C) 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the \"License\")\;
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an \"AS IS\" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#The set of auto generated protoc rules
")
set(PROTO_FOOTER "
")

function(scan_for_protos)
    set(proto_cmake "${CMAKE_SOURCE_DIR}/CMakeProto.cmake")
    if(FORCING_GLOB OR NOT EXISTS ${proto_cmake})
        file(GLOB_RECURSE all_protos RELATIVE ${CMAKE_SOURCE_DIR}
            "${CMAKE_SOURCE_DIR}/core/*.proto"
            "${CMAKE_SOURCE_DIR}/gapic/*.proto"
            "${CMAKE_SOURCE_DIR}/gapis/*.proto"
            "${CMAKE_SOURCE_DIR}/test/*.proto"
        )
        list(SORT all_protos)
        file(WRITE ${proto_cmake} ${PROTO_HEADER})
        set(proto_set)
        set(proto_dir)
        foreach(proto ${all_protos})
            # group by path
            get_filename_component(proto_path ${proto} DIRECTORY)
            get_filename_component(proto_name ${proto} NAME_WE)
            if(NOT proto_dir STREQUAL proto_path)
                add_proto_rules(${proto_cmake} "${proto_dir}" "${proto_set}")
                set(proto_dir ${proto_path})
                set(proto_set)
            endif()
            list(APPEND proto_set ${proto_name})
        endforeach()
        add_proto_rules(${proto_cmake} "${proto_dir}" "${proto_set}")
        file(APPEND ${proto_cmake} ${PROTO_FOOTER})
    endif()
    if(NOT DISABLE_PROTOC)
        include(${proto_cmake})
    endif()
endfunction()

function(add_proto_rules rules proto_dir proto_set)
    set(go_protos)
    set(java_protos)
    set(java_classes)
    set(cc_protos)
    foreach(proto_name ${proto_set})
        set(proto "${proto_dir}/${proto_name}.proto")
        scan_proto(${proto})
        if(NOT go_package STREQUAL "")
            list(APPEND go_protos ${proto_name})
        endif()
        if(NOT java_package STREQUAL "")
            list(APPEND java_protos ${proto_name})
            if(NOT java_class STREQUAL "")
                string(REPLACE "." "/" class "${java_package}.${java_class}")
                list(APPEND java_classes ${class})
            endif()
            foreach(service ${services})
                string(REPLACE "." "/" class "${java_package}.${service}Grpc")
                list(APPEND java_classes ${class})
            endforeach()
        endif()
        if(NOT cc_package STREQUAL "")
            list(APPEND cc_protos ${proto_name})
        endif()
    endforeach()
    if(go_protos)
        set(go_inputs)
        foreach(proto_name ${go_protos})
            list(APPEND go_inputs "${proto_name}.proto")
        endforeach()
        file(APPEND ${proto_cmake} "
protoc_go(\"${go_package}\" \"${proto_dir}\" \"${go_inputs}\")")
    endif()
    if(java_protos)
        set(java_inputs)
        foreach(proto_name ${java_protos})
            list(APPEND java_inputs "${proto_name}.proto")
        endforeach()
        file(APPEND ${proto_cmake} "
protoc_java(\"${proto_dir}\" \"${java_inputs}\" \"${java_classes}\")")
    endif()
    if(cc_protos)
        set(cc_inputs)
        foreach(proto_name ${go_protos})
            list(APPEND cc_inputs "${proto_name}.proto")
        endforeach()
        file(RELATIVE_PATH cc_package "${ROOT_DIR}/src" "${CMAKE_SOURCE_DIR}/${proto_dir}")
        file(APPEND ${proto_cmake} "
protoc_cc(\"${cc_package}\" \"${proto_dir}\" \"${cc_inputs}\")")
    endif()
endfunction()

function(scan_proto proto)
    set(full_proto "${CMAKE_SOURCE_DIR}/${proto}")
    set(go_package "")
    set(java_package "")
    set(java_class "")
    set(cc_package "")
    set(messages)
    set(enums)
    file(STRINGS "${full_proto}" lines)
    foreach(line ${lines})
        if(line MATCHES "option go_package = \"(.*)\";")
            set(go_package "${CMAKE_MATCH_1}")
        endif()
        if(line MATCHES "option java_package = \"(.*)\";")
            set(java_package "${CMAKE_MATCH_1}")
        endif()
        if(line MATCHES "option java_outer_classname = \"(.*)\";")
            set(java_class "${CMAKE_MATCH_1}")
        endif()
        if(line MATCHES "cc_package")
            set(cc_package true)
        endif()
        if(line MATCHES "message +(.*) +{")
            list(APPEND messages "${CMAKE_MATCH_1}")
        endif()
        if(line MATCHES "enum +(.*) +{")
            list(APPEND enums "${CMAKE_MATCH_1}")
        endif()
        if(line MATCHES "service +(.*) +{")
            list(APPEND services "${CMAKE_MATCH_1}")
        endif()
    endforeach()
    set(go_package "${go_package}" PARENT_SCOPE)
    set(java_package "${java_package}" PARENT_SCOPE)
    set(java_class "${java_class}" PARENT_SCOPE)
    set(cc_package "${cc_package}" PARENT_SCOPE)
    set(messages "${messages}" PARENT_SCOPE)
    set(enums "${enums}" PARENT_SCOPE)
    set(services "${services}" PARENT_SCOPE)
endfunction()

function(protoc_go go_package src_dir protos)
    set(outputs)
    foreach(proto ${protos})
        get_filename_component(proto_name ${proto} NAME_WE)
        file(TO_NATIVE_PATH "${GO_SRC}/${go_package}/${proto_name}.pb.go" os_proto)
        list(APPEND outputs "${os_proto}")
    endforeach()
    file(TO_NATIVE_PATH "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/protoc-gen-go" os_plugin)
    file(TO_NATIVE_PATH "${ROOT_DIR}/src" os_go_out)
   
    _do_protoc("go" "${src_dir}" "${protos}" "${outputs}"
        "--plugin=${os_plugin}"
        "--go_out=plugins=grpc:${os_go_out}")
endfunction()

function(protoc_cc cc_package src_dir protos)
    set(outputs)
    set(cc_out "${CMAKE_BINARY_DIR}/proto_cc")
    set(dest_dir "${cc_out}/${cc_package}")
    foreach(proto ${protos})
        get_filename_component(proto_name ${proto} NAME_WE)
        file(TO_NATIVE_PATH "${dest_dir}/${proto_name}.pb.cc" os_pb_cc)
        file(TO_NATIVE_PATH "${dest_dir}/${proto_name}.pb.h" os_pb_h)
        list(APPEND outputs ${os_pb_cc} ${os_pb_h})
    endforeach()
    file(TO_NATIVE_PATH ${cc_out} os_cc_out)
    _do_protoc("cc" "${src_dir}" "${protos}" "${outputs}" "--cpp_out=${os_cc_out}")
endfunction()

function(protoc_java src_dir protos classes)
    set(outputs)
    foreach(class ${classes})
        file(TO_NATIVE_PATH "${JAVA_SERVICE}/${class}.java" os_java_output)
        list(APPEND outputs ${os_java_output})
    endforeach()
    
    file(TO_NATIVE_PATH "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/protoc-gen-grpc-java${CMAKE_HOST_EXECUTABLE_SUFFIX}" os_protoc_gen_grpc_java)
    file(TO_NATIVE_PATH "${JAVA_SERVICE}" os_java_service)

    _do_protoc("java" "${src_dir}" "${protos}" "${outputs}" 
        "--plugin=protoc-gen-grpc-java=${os_protoc_gen_grpc_java}"
        "--java_out=${os_java_service}"
        "--grpc-java_out=${os_java_service}"
    )
endfunction()

function(_do_protoc type src_dir protos outputs)
    abs_list(protos "${CMAKE_SOURCE_DIR}/${src_dir}")
    set(os_protos)
    foreach(proto ${protos})
        file(TO_NATIVE_PATH "${proto}" os_proto)
        list(APPEND os_protos ${os_proto})
    endforeach()
    
    set(os_proto_path "${PROTO_PATH}")
    file(TO_NATIVE_PATH "${os_proto_path}" os_proto_path)
    if (NOT WIN32)
        string(REPLACE ";" ":" os_proto_path "${os_proto_path}")
    endif()

    add_custom_command(
        OUTPUT ${outputs}
        COMMAND protoc
            "--proto_path=${os_proto_path}"
            ${ARGN}
            ${os_protos}
        DEPENDS ${protos} protoc-gen-go protoc-gen-grpc-java
        COMMENT "protoc ${proto}"
        WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
    )
    set_source_files_properties(${outputs} PROPERTIES GENERATED TRUE)
    string(REPLACE "/" "-" target_name "${src_dir}-proto-${type}")
    all_target(protoc ${target_name} ${outputs})
endfunction()
