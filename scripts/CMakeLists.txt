cmake_minimum_required(VERSION 3.16.3)
project(gapid2_layer)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

set(LIB_NAME ${LAYER_LIB} CACHE STRING "Name of the layer library")
set(LIB_SRC ${LAYER_SRC} CACHE STRING "Layer source file")

if (NOT LIB_NAME)
    message(FATAL_ERROR "Please specify a library name")
endif()

if (NOT LIB_SRC)
    message(FATAL_ERROR "Please set LIB_SRC")
endif()

if (NOT (EXISTS "${LIB_SRC}"))
    message(FATAL_ERROR "Invalid file ${LIB_SRC}")
endif()
file(TO_CMAKE_PATH "${LIB_SRC}" LIB_SRC)

include_directories(${CMAKE_CURRENT_SOURCE_DIR}/Vulkan-Headers/include)
include_directories(${CMAKE_CURRENT_SOURCE_DIR})
include_directories(${CMAKE_CURRENT_BINARY_DIR})


function(generate_file SOURCE)
    set(of)
    foreach(X ${ARGN})
        list(APPEND of ${CMAKE_CURRENT_BINARY_DIR}/${X})
    endforeach()
    add_custom_command(OUTPUT
        ${of}
        COMMAND ${Python3_EXECUTABLE} 
            ${CMAKE_CURRENT_SOURCE_DIR}/${SOURCE}
            ${CMAKE_CURRENT_SOURCE_DIR}/Vulkan-Headers/registry/vk.xml
            ${CMAKE_CURRENT_BINARY_DIR}
        DEPENDS
            ${CMAKE_CURRENT_SOURCE_DIR}/${SOURCE}
            ${CMAKE_CURRENT_SOURCE_DIR}/generator/args.py
            ${CMAKE_CURRENT_SOURCE_DIR}/generator/standard.py
            ${CMAKE_CURRENT_SOURCE_DIR}/generator/vulkan.py
            ${CMAKE_CURRENT_SOURCE_DIR}/generator/generator.py
            ${CMAKE_CURRENT_SOURCE_DIR}/Vulkan-Headers/registry/vk.xml
    )
endfunction()

generate_file(
    "layer_internal.py"
    "layer_internal.inl"
)

add_library(${LIB_NAME} SHARED
    "${LIB_SRC}"
    ${CMAKE_CURRENT_BINARY_DIR}/layer_internal.inl
)
target_compile_definitions(${LIB_NAME} PRIVATE VK_USE_PLATFORM_WIN32_KHR=1)