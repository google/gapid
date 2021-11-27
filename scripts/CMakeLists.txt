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

include_directories(${CMAKE_CURRENT_SOURCE_DIR}/../externals/Vulkan-Headers/include)
include_directories(${CMAKE_CURRENT_BINARY_DIR})

add_custom_command(OUTPUT
    ${CMAKE_CURRENT_BINARY_DIR}/layer.h
    COMMAND ${Python3_EXECUTABLE} 
        ${CMAKE_CURRENT_SOURCE_DIR}/../generator.py
        ${CMAKE_CURRENT_SOURCE_DIR}/../externals/Vulkan-Headers/registry/vk.xml
        ${CMAKE_CURRENT_BINARY_DIR}
    DEPENDS 
        ${CMAKE_CURRENT_SOURCE_DIR}/../generator.py
        ${CMAKE_CURRENT_SOURCE_DIR}/../externals/Vulkan-Headers/registry/vk.xml
)

add_library(${LIB_NAME} SHARED
    "${LIB_SRC}"
    ${CMAKE_CURRENT_BINARY_DIR}/layer.h
)
target_compile_definitions(${LIB_NAME} PRIVATE VK_USE_PLATFORM_WIN32_KHR=1)