cmake_minimum_required(VERSION 3.16.3)
project(gapid2)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include_directories(${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/include)
include_directories(${gapid2_SOURCE_DIR}/externals/digestpp)
add_subdirectory(externals/vk_callback_swapchain)

add_library(spirv-reflect STATIC 
    ${CMAKE_CURRENT_SOURCE_DIR}/externals/SPIRV-Reflect/spirv_reflect.h
    ${CMAKE_CURRENT_SOURCE_DIR}/externals/SPIRV-Reflect/spirv_reflect.c
)

find_package (Python3 COMPONENTS Interpreter REQUIRED)

#add_compile_options("/bigobj")

set(ALL_HEADERS
    buffer_view.h
    buffer.h
    command_buffer.h
    command_pool.h
    common.h
    creation_data_tracker.h
    decoder.h
    descriptor_pool.h
    descriptor_set_layout.h
    descriptor_set.h
    descriptor_update_template.h
    device_memory.h
    device.h
    encoder.h
    event.h
    fence.h
    framebuffer.h
    handle_runner.h
    handle_templates.h
    handles.h
    helpers.cpp
    helpers.h
    image_view.h
    image.h
    instance.h
    memory_tracker.cpp
    memory_tracker.h
    minimal_state_tracker.h
    null_cloner.h
    physical_device.h
    pipeline_cache.h
    pipeline_layout.h
    pipeline.h
    query_pool.h
    queue.h
    render_pass.h
    sampler_ycbcr_conversion.h
    sampler.h
    semaphore.h
    shader_module.h
    state_tracker.h
    surface.h
    swapchain.h
    temporary_allocator.h
)

set(ALL_GENERATED 
    ${gapid2_BINARY_DIR}/struct_deserialization.h
    ${gapid2_BINARY_DIR}/struct_serialization.h
    ${gapid2_BINARY_DIR}/struct_clone.h
    ${gapid2_BINARY_DIR}/return_serializer.h
    ${gapid2_BINARY_DIR}/commands.h
    ${gapid2_BINARY_DIR}/command_processor.h
    ${gapid2_BINARY_DIR}/command_caller.h
    ${gapid2_BINARY_DIR}/forwards.h
    ${gapid2_BINARY_DIR}/device_functions.h
    ${gapid2_BINARY_DIR}/instance_functions.h
    ${gapid2_BINARY_DIR}/call_forwards.h
    ${gapid2_BINARY_DIR}/handle_defines.inl
    ${gapid2_BINARY_DIR}/command_deserializer.h
    ${gapid2_BINARY_DIR}/layer_internal.h
    ${gapid2_BINARY_DIR}/layerer.h
    ${gapid2_BINARY_DIR}/struct_printer.h
    ${gapid2_BINARY_DIR}/command_printer.h
)

add_custom_command(OUTPUT
    ${ALL_GENERATED}
    ${gapid2_BINARY_DIR}/call_forwards.cpp
    COMMAND ${Python3_EXECUTABLE} 
        ${gapid2_SOURCE_DIR}/generator.py
        ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/registry/vk.xml
        ${gapid2_BINARY_DIR}
    DEPENDS 
        ${gapid2_SOURCE_DIR}/generator.py
        ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/registry/vk.xml
)

add_library(gapii SHARED
    helpers.cpp
    memory_tracker.cpp
    layer_setup.cpp 
    layer.def
    spy.h
    ${gapid2_BINARY_DIR}/call_forwards.cpp
    ${ALL_GENERATED}
    ${ALL_HEADERS}
)

target_compile_definitions(gapii PRIVATE VK_USE_PLATFORM_WIN32_KHR=1)
target_link_libraries(gapii spirv-reflect)
target_include_directories(gapii PRIVATE
    ${gapid2_BINARY_DIR}
    ${gapid2_SOURCE_DIR}
    ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/include/vulkan
)

add_executable(gapir
    replayer.cpp
    helpers.h
    helpers.cpp
    ${ALL_GENERATED}
    ${ALL_HEADERS}
)

target_compile_definitions(gapir PRIVATE VK_USE_PLATFORM_WIN32_KHR=1 NOMINMAX=1)

target_include_directories(gapir PRIVATE
    ${gapid2_BINARY_DIR}
    ${gapid2_SOURCE_DIR}
    ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/include/vulkan
)

add_executable(printer
    printer.cpp
    helpers.cpp
    ${ALL_GENERATED}
    ${ALL_HEADERS}
)

target_compile_definitions(printer PRIVATE VK_USE_PLATFORM_WIN32_KHR=1 NOMINMAX=1)

target_include_directories(printer PRIVATE
    ${gapid2_BINARY_DIR}
    ${gapid2_SOURCE_DIR}
    ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/include/vulkan
)

configure_file(
    manifest.json.in 
    ${gapid2_BINARY_DIR}/manifest.json
    @ONLY 
)
