cmake_minimum_required(VERSION 3.16.3)
project(gapid2)
set(CMAKE_CXX_STANDARD 23)
set(CMAKE_C_STANDARD 99)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

if(MSVC)
  add_compile_options("/ZI")
  add_link_options("/INCREMENTAL")
endif()

include_directories(${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/include)
include_directories(${gapid2_SOURCE_DIR}/externals/digestpp)
add_subdirectory(externals/vk_callback_swapchain)

## Glslang options, disable as much as we can
set(BUILD_EXTERNAL OFF)
set(SKIP_GLSLANG_INSTALL ON)
set(ENABLE_SPVREMAPPER OFF)
set(ENABLE_GLSLANG_BINARIES OFF)
set(ENABLE_OPT OFF)
set(ENABLE_CTEST OFF)
set(ENABLE_HLSL OFF)
add_subdirectory(externals/glslang)

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
    scripts/stype_header.py
    stype_header.h
)

generate_file(
    scripts/command_deserializer.py
    command_deserializer.h
)

generate_file(
    scripts/null_caller.py
    null_caller.h
)

generate_file(
    scripts/struct_print.py
    struct_print.h
)

generate_file(
    scripts/command_printer.py
    command_printer.h
)

generate_file(
    scripts/command_buffer_invalidator.py
    command_buffer_invalidator.h
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


add_library(HANDLE_TYPES STATIC
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
    ${gapid2_BINARY_DIR}/forwards.h
)

add_library(BASE_FILES STATIC
    base_caller.cpp
    helpers.cpp
    minimal_state_tracker.inl
    minimal_state_tracker.h
    custom.h
    custom.cpp
    creation_data_tracker.h
    creation_tracker.h
    state_block.cpp
    utils.h
    command_serializer_fns.cpp
    enums.h
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
    ${gapid2_BINARY_DIR}/command_deserializer.h
    ${gapid2_BINARY_DIR}/command_buffer_invalidator.h
    ${gapid2_BINARY_DIR}/null_caller.h
    ${gapid2_BINARY_DIR}/stype_header.h
)

add_library(LAYER_FILES OBJECT
    ${gapid2_BINARY_DIR}/call_forwards.h
    ${gapid2_BINARY_DIR}/call_forwards.cpp
)

target_link_libraries(LAYER_FILES PUBLIC 
    BASE_FILES)

add_library(test SHARED
    passthrough_layer.cpp
    layer.def
    layer_setup.cpp
)

target_link_libraries(test BASE_FILES HANDLE_TYPES LAYER_FILES)

add_library(gapii SHARED
    spy.h
    spy.cpp
    spy_serializer.h
    spy_serializer.cpp
    mec_controller.h
    mec_controller.cpp
    gapii.cpp
    gapii.h
    state_tracker.h
    memory_tracker.h
    memory_tracker.cpp
    layer.def
    layer_setup.cpp
    command_buffer_invalidator.cpp
    mec_capture/buffer_view.cpp
    mec_capture/buffer.cpp
    mec_capture/command_buffer.cpp
    mec_capture/command_pool.cpp
    mec_capture/descriptor_pool.cpp
    mec_capture/descriptor_set_layout.cpp
    mec_capture/descriptor_set.cpp
    mec_capture/descriptor_update_template.cpp
    mec_capture/device_memory.cpp
    mec_capture/device.cpp
    mec_capture/framebuffer.cpp
    mec_capture/image_view.cpp
    mec_capture/image.cpp
    mec_capture/instance.cpp
    mec_capture/mid_execution_generator.h
    mec_capture/physical_device.cpp
    mec_capture/pipeline_cache.cpp
    mec_capture/pipeline_layout.cpp
    mec_capture/pipeline.cpp
    mec_capture/query_pool.cpp
    mec_capture/queue.cpp
    mec_capture/render_pass.cpp
    mec_capture/sampler_ycbcr_conversion.cpp
    mec_capture/sampler.cpp
    mec_capture/shader_module.cpp
    mec_capture/surface.cpp
    mec_capture/swapchain.cpp
    mec_capture/synchronization.cpp
    mec_capture/utils.cpp
    mec_capture/staging_resource_manager.cpp
    mec_capture/staging_resource_manager.h
    mec_capture/image_copier.cpp
    mec_capture/image_copier.h
    mec_capture/shader_manager.h
    mec_capture/shader_manager.cpp
    ${glslang_SOURCE_DIR}/Standalone/ResourceLimits.cpp
    ${glslang_SOURCE_DIR}/Standalone/resource_limits_c.cpp
)

target_link_libraries(gapii BASE_FILES HANDLE_TYPES LAYER_FILES spirv-reflect glslang SPIRV)
target_compile_definitions(gapii PRIVATE MEC=1)

add_executable(gapir
    replayer.cpp
    ${gapid2_BINARY_DIR}/handle_replacer.h
    ${gapid2_BINARY_DIR}/handle_replacer.cpp
    ${gapid2_BINARY_DIR}/command_inline_fixer.h
    ${gapid2_BINARY_DIR}/command_inline_fixer.cpp
    ${gapid2_BINARY_DIR}/fix_handles.h
)
target_link_libraries(gapir BASE_FILES HANDLE_TYPES )

add_executable(printer
    printer.cpp
    ${gapid2_BINARY_DIR}/struct_print.h
    ${gapid2_BINARY_DIR}/command_printer.h
)
target_link_libraries(printer BASE_FILES HANDLE_TYPES)

foreach(L test gapii gapir printer BASE_FILES HANDLE_TYPES LAYER_FILES)
    target_include_directories(${L} PRIVATE
        ${gapid2_SOURCE_DIR}
        ${gapid2_SOURCE_DIR}/handles
        ${gapid2_BINARY_DIR}
        ${gapid2_SOURCE_DIR}/externals/Vulkan-Headers/include
    )

    target_compile_definitions(${L} PRIVATE ${ADDITIONAL_DEFINES} NOMINMAX=1)
endforeach()


function(set_directories target)
    get_target_property(srcs ${target} SOURCES)
    foreach(X ${srcs})
        get_filename_component(comp ${X} EXT)
        get_filename_component(abs ${X} ABSOLUTE)
        file(RELATIVE_PATH rel_bin ${CMAKE_CURRENT_BINARY_DIR} ${abs})
        file(RELATIVE_PATH rel_src ${CMAKE_CURRENT_SOURCE_DIR} ${abs})
        string(FIND ${rel_bin} ".." fout)
        if (fout EQUAL -1)
            set(rel ${rel_bin})
        else()
            set(rel ${rel_src})
        endif()

        get_filename_component(reldir ${rel} DIRECTORY)
        if (("${comp}" STREQUAL ".h") OR ("${comp}" STREQUAL ".hpp"))
            message("SETTING ${X} to inc/${rel}")
            source_group("inc/${reldir}" FILES ${X})
        else()
            message("SETTING ${X} to src/${rel}")
            source_group("src/${reldir}" FILES ${X})
        endif()
        
    endforeach()
endfunction()

set_directories(test)
set_directories(gapii)
set_directories(gapir)
set_directories(printer)
set_directories(BASE_FILES)
set_directories(HANDLE_TYPES)
set_directories(LAYER_FILES)