cmake_minimum_required(VERSION 3.16.3)
project(gapid2)
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_C_STANDARD 99)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include_directories(${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/include)
include_directories(${gapid2_SOURCE_DIR}/externals/digestpp)
add_subdirectory(externals/vk_callback_swapchain)

add_library(spirv-reflect STATIC 
    ${CMAKE_CURRENT_SOURCE_DIR}/externals/SPIRV-Reflect/spirv_reflect.h
    ${CMAKE_CURRENT_SOURCE_DIR}/externals/SPIRV-Reflect/spirv_reflect.c
)

find_package (Python3 COMPONENTS Interpreter REQUIRED)

set(ADDITIONAL_DEFINES)

if (WIN32)
    set(ADDITIONAL_DEFINES VK_USE_PLATFORM_WIN32_KHR=1)
    add_compile_options("/bigobj")
    add_compile_options("/MP")
elseif (UNIX)
    set(ADDITIONAL_DEFINES VK_USE_PLATFORM_XCB_KHR=1)
endif()

configure_file(
    scripts/layer_internal.py
    ${gapid2_BINARY_DIR}/scripts/layer_internal.py
    COPYONLY
)

configure_file(
    scripts/build_file.bat
    ${gapid2_BINARY_DIR}/scripts/build_file.bat
    COPYONLY
)

configure_file(
    layer.h
    ${gapid2_BINARY_DIR}/scripts/layer.h
    COPYONLY
)

configure_file(
    layer_internal_setup.h
    ${gapid2_BINARY_DIR}/scripts/layer_internal_setup.h
    COPYONLY
)

configure_file(
    scripts/CMakeLists.txt
    ${gapid2_BINARY_DIR}/scripts/CMakeLists.txt
    COPYONLY
)

configure_file(
    scripts/generator/args.py
    ${gapid2_BINARY_DIR}/scripts/generator/args.py
    COPYONLY
)

configure_file(
    scripts/generator/standard.py
    ${gapid2_BINARY_DIR}/scripts/generator/standard.py
    COPYONLY
)

configure_file(
    scripts/generator/vulkan.py
    ${gapid2_BINARY_DIR}/scripts/generator/vulkan.py
    COPYONLY
)

configure_file(
    scripts/generator/generator.py
    ${gapid2_BINARY_DIR}/scripts/generator/generator.py
    COPYONLY
)

configure_file(
    scripts/generator/arg_serialization.py
    ${gapid2_BINARY_DIR}/scripts/generator/arg_serialization.py
    COPYONLY
)


file(COPY externals/Vulkan-Headers DESTINATION ${gapid2_BINARY_DIR}/scripts/)

function(generate_file SOURCE)
    set(of)
    foreach(X ${ARGN})
        list(APPEND of ${gapid2_BINARY_DIR}/${X})
    endforeach()
    add_custom_command(OUTPUT
        ${of}
        COMMAND ${Python3_EXECUTABLE} 
            ${gapid2_SOURCE_DIR}/${SOURCE}
            ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/registry/vk.xml
            ${gapid2_BINARY_DIR}
        DEPENDS
            ${gapid2_SOURCE_DIR}/${SOURCE}
            scripts/generator/args.py
            scripts/generator/standard.py
            scripts/generator/vulkan.py
            scripts/generator/generator.py
            scripts/generator/arg_serialization.py
            ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/registry/vk.xml
    )
endfunction()

generate_file(
    scripts/transform_base.py
    transform_base.h
)

generate_file(
    scripts/transform.py
    transform.h
)

generate_file(
    scripts/struct_serializer.py
    struct_serialization.h
)

generate_file(
    scripts/struct_deserializer.py
    struct_deserialization.h
)

generate_file(
    scripts/struct_clone.py
    struct_clone.h
)

generate_file(
    scripts/forwards.py
    forwards.h
)

generate_file(
    scripts/handle_defines.py
    handle_defines.inl
)

generate_file(
    scripts/base_caller.py
    base_caller.h
)

generate_file(
    scripts/instance_functions.py
    instance_functions.h
)

generate_file(
    scripts/device_functions.py
    device_functions.h
)

generate_file(
    scripts/layerer.py
    layerer.h
)

generate_file(
    scripts/call_forwards.py
    call_forwards.h
    call_forwards.cpp
)

generate_file(
    scripts/indirect_functions.py
    indirect_functions.h
)

generate_file(
    scripts/command_buffer_recorder.py
    command_buffer_recorder.inl
    command_buffer_recorder.cpp
)

generate_file(
    scripts/command_buffer_deserializer.py
    command_buffer_deserializer.h
)

generate_file(
    scripts/command_serializer.py
    command_serializer.h
    command_serializer.cpp
)

generate_file(
    scripts/handle_replacer.py
    handle_replacer.h
    handle_replacer.cpp
)

generate_file(
    scripts/fix_handles.py
    fix_handles.h
)

generate_file(
    scripts/command_inline_fixer.py
    command_inline_fixer.h
    command_inline_fixer.cpp
)

generate_file(
    scripts/command_deserializer.py
    command_deserializer.h
)

generate_file(
    scripts/null_caller.py
    null_caller.h
)

configure_file(
    manifest.json.in 
    ${gapid2_BINARY_DIR}/manifest.json
    @ONLY 
)

configure_file(
    manifest2.json.in 
    ${gapid2_BINARY_DIR}/manifest2.json
    @ONLY 
)


set(HANDLE_TYPES
    handles/buffer_view.cpp
    handles/buffer_view.h
    handles/buffer.cpp
    handles/buffer.h
    handles/command_buffer.cpp
    handles/command_buffer.h
    handles/command_pool.cpp
    handles/command_pool.h
    handles/descriptor_pool.cpp
    handles/descriptor_pool.h
    handles/descriptor_update_template.cpp
    handles/descriptor_update_template.h
    handles/descriptor_set_layout.cpp
    handles/descriptor_set_layout.h
    handles/descriptor_set.cpp
    handles/descriptor_set.h
    handles/device_memory.cpp
    handles/device_memory.h
    handles/device.cpp
    handles/device.h
    handles/event.cpp
    handles/event.h
    handles/fence.cpp
    handles/fence.h
    handles/framebuffer.cpp
    handles/framebuffer.h
    handles/image_view.cpp
    handles/image_view.h
    handles/image.cpp
    handles/image.h
    handles/instance.cpp
    handles/instance.h
    handles/physical_device.cpp
    handles/physical_device.h
    handles/pipeline_cache.cpp
    handles/pipeline_cache.h
    handles/pipeline_layout.cpp
    handles/pipeline_layout.h
    handles/pipeline.cpp
    handles/pipeline.h
    handles/query_pool.cpp
    handles/query_pool.h
    handles/queue.cpp
    handles/queue.h
    handles/render_pass.cpp
    handles/render_pass.h
    handles/sampler_ycbcr_conversion.cpp
    handles/sampler_ycbcr_conversion.h
    handles/sampler.cpp
    handles/sampler.h
    handles/semaphore.cpp
    handles/semaphore.h
    handles/shader_module.cpp
    handles/shader_module.h
    handles/surface.cpp
    handles/surface.h
    handles/swapchain.cpp
    handles/swapchain.h
)

set(BASE_FILES 
    base_caller.cpp
    helpers.cpp
    minimal_state_tracker.cpp
    state_block.cpp
    ${HANDLE_TYPES}
    ${gapid2_BINARY_DIR}/forwards.h
    ${gapid2_BINARY_DIR}/struct_clone.h
    ${gapid2_BINARY_DIR}/device_functions.h
    ${gapid2_BINARY_DIR}/instance_functions.h
    ${gapid2_BINARY_DIR}/indirect_functions.h
    ${gapid2_BINARY_DIR}/struct_deserialization.h
    ${gapid2_BINARY_DIR}/struct_serialization.h
    ${gapid2_BINARY_DIR}/transform_base.h
    ${gapid2_BINARY_DIR}/handle_defines.inl
    ${gapid2_BINARY_DIR}/transform.h
    ${gapid2_BINARY_DIR}/base_caller.h
    ${gapid2_BINARY_DIR}/layerer.h
    ${gapid2_BINARY_DIR}/command_buffer_recorder.inl
    ${gapid2_BINARY_DIR}/command_buffer_recorder.cpp
    ${gapid2_BINARY_DIR}/command_buffer_deserializer.h
    ${gapid2_BINARY_DIR}/command_serializer.h
    ${gapid2_BINARY_DIR}/command_serializer.cpp
    ${gapid2_BINARY_DIR}/command_inline_fixer.h
    ${gapid2_BINARY_DIR}/command_inline_fixer.cpp
    ${gapid2_BINARY_DIR}/command_deserializer.h
    ${gapid2_BINARY_DIR}/null_caller.h
)

set(LAYER_FILES 
    layer_setup.cpp
    layer.def
    ${gapid2_BINARY_DIR}/call_forwards.h
    ${gapid2_BINARY_DIR}/call_forwards.cpp
)

add_library(test SHARED
    passthrough_layer.cpp
    ${BASE_FILES}
    ${LAYER_FILES}
)

add_library(gapii SHARED
    ${BASE_FILES}
    ${LAYER_FILES}
    spy.h
    spy.cpp
    gapii.cpp
    memory_tracker.h
    memory_tracker.cpp
)

add_executable(gapir
    ${BASE_FILES}
    replayer.cpp
    ${gapid2_BINARY_DIR}/handle_replacer.h
    ${gapid2_BINARY_DIR}/handle_replacer.cpp
    ${gapid2_BINARY_DIR}/fix_handles.h
)

foreach(L test gapii gapir)
    target_include_directories(${L} PRIVATE
        ${gapid2_SOURCE_DIR}
        ${gapid2_SOURCE_DIR}/handles
        ${gapid2_BINARY_DIR}
        ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/include
    )

    target_compile_definitions(${L} PRIVATE ${ADDITIONAL_DEFINES} NOMINMAX=1)
endforeach()
