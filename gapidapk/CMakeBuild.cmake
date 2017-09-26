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

set(android_sources
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/build.gradle
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/java/com/google/android/gapid/Cache.java
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/java/com/google/android/gapid/Counter.java
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/java/com/google/android/gapid/DeviceInfoService.java
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/java/com/google/android/gapid/FileCache.java
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/java/com/google/android/gapid/IOUtils.java
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/java/com/google/android/gapid/PackageInfoService.java
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/java/com/google/android/gapid/SocketWriter.java
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/res/drawable-xxxhdpi/logo.png
    ${CMAKE_CURRENT_SOURCE_DIR}/android/build.gradle
    ${CMAKE_CURRENT_SOURCE_DIR}/android/gradle.properties
    ${CMAKE_CURRENT_SOURCE_DIR}/android/gradle/wrapper/gradle-wrapper.jar
    ${CMAKE_CURRENT_SOURCE_DIR}/android/gradle/wrapper/gradle-wrapper.properties
    ${CMAKE_CURRENT_SOURCE_DIR}/android/gradlew
    ${CMAKE_CURRENT_SOURCE_DIR}/android/gradlew.bat
    ${CMAKE_CURRENT_SOURCE_DIR}/android/settings.gradle
)
set(android_configure_sources
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/res/values/strings.xml
    ${CMAKE_CURRENT_SOURCE_DIR}/android/app/src/main/AndroidManifest.xml
)


set(EXTRA_ANDROID_PARAM "")
if (CMAKE_BUILD_TYPE STREQUAL "Debug")
    set(EXTRA_ANDROID_PARAM "android:debuggable=\"true\"")
endif()

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    set(abi_bin "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${ANDROID_BUILD_PATH_${abi}}/")
    set(apk_dir "${CMAKE_CURRENT_BINARY_DIR}/${abi}/gapid-apk")
    set(jni_libs "${apk_dir}/app/src/main/jniLibs/${ANDROID_ABI_NAME_${abi}}/")

    set(TARGET_SOURCES)
    set(libgapir_dest ${jni_libs}/libgapir.so)
    add_custom_command(
        OUTPUT ${libgapir_dest}
        COMMAND "${CMAKE_COMMAND}" -E copy
            ${abi_bin}/libgapir.so
            ${libgapir_dest}
        DEPENDS
            ${abi_bin}/libgapir.so
    )
    list(APPEND TARGET_SOURCES ${libgapir_dest})

    set(libgapii_dest ${jni_libs}/libgapii.so)
    add_custom_command(
        OUTPUT ${libgapii_dest}
        COMMAND "${CMAKE_COMMAND}" -E copy
            ${abi_bin}/libgapii.so
            ${libgapii_dest}
        DEPENDS
            ${abi_bin}/libgapii.so
    )
    list(APPEND TARGET_SOURCES ${libgapii_dest})

    set(libinterceptor_dest ${jni_libs}/libinterceptor.so)
    add_custom_command(
        OUTPUT ${libinterceptor_dest}
        COMMAND "${CMAKE_COMMAND}" -E copy
            ${abi_bin}/libinterceptor.so
            ${libinterceptor_dest}
        DEPENDS
            ${abi_bin}/libinterceptor.so
    )
    list(APPEND TARGET_SOURCES ${libinterceptor_dest})

    set(libdeviceinfo_dest ${jni_libs}/libdeviceinfo.so)
    add_custom_command(
        OUTPUT ${libdeviceinfo_dest}
        COMMAND "${CMAKE_COMMAND}" -E copy
            ${abi_bin}/libdeviceinfo.so
            ${libdeviceinfo_dest}
        DEPENDS
            ${abi_bin}/libdeviceinfo.so
    )
    list(APPEND TARGET_SOURCES ${libdeviceinfo_dest})

    set(vk_graphics_spy_dest ${jni_libs}/libVkLayerGraphicsSpy.so)
    add_custom_command(
        OUTPUT ${vk_graphics_spy_dest}
        COMMAND "${CMAKE_COMMAND}" -E copy
            ${abi_bin}/libVkLayerGraphicsSpy.so
            ${vk_graphics_spy_dest}
        DEPENDS
            ${abi_bin}/libVkLayerGraphicsSpy.so
    )
    list(APPEND TARGET_SOURCES ${vk_graphics_spy_dest})

    set(virtual_swapchain_dest ${jni_libs}/libVkLayer_VirtualSwapchain.so)
    add_custom_command(
        OUTPUT ${virtual_swapchain_dest}
        COMMAND "${CMAKE_COMMAND}" -E copy
            ${abi_bin}/libVkLayer_VirtualSwapchain.so
            ${virtual_swapchain_dest}
        DEPENDS
            ${abi_bin}/libVkLayer_VirtualSwapchain.so
    )
    list(APPEND TARGET_SOURCES ${virtual_swapchain_dest})

    set(GAPID_APK_ABI "${ANDROID_APK_NAME_${abi}}")

    foreach (source ${android_sources})
        file(RELATIVE_PATH rooted_source ${CMAKE_CURRENT_SOURCE_DIR}/android ${source})
        configure_file(${source} ${apk_dir}/${rooted_source} COPYONLY)
        list(APPEND TARGET_SOURCES ${apk_dir}/${rooted_source})
    endforeach()
    foreach (source ${android_configure_sources})
        file(RELATIVE_PATH rooted_source ${CMAKE_CURRENT_SOURCE_DIR}/android ${source})
        configure_file(${source}.in ${apk_dir}/${rooted_source} @ONLY)
        list(APPEND TARGET_SOURCES ${apk_dir}/${rooted_source})
    endforeach()

    string(REPLACE ";" "," all_inputs "${TARGET_SOURCES}")

    set(gradle_out "${apk_dir}/app/build/outputs/apk/app-debug.apk")
    gradle(${abi}-gradle-gapid-apk
        OUTPUT "${gradle_out}"
        DIRECTORY "${apk_dir}"
        ARGS
            "-Pfilehash=$<TARGET_FILE:filehash>"
            "-PgapidVersionAndBuild=${GAPID_VERSION_AND_BUILD}"
            "-Pinputs=${all_inputs}"
        DEPENDS
            ${TARGET_SOURCES}
            ${android_sources}
            filehash
    )

    set(abi_gapid_apk "${abi_bin}/gapid-${GAPID_APK_ABI}.apk")
    add_custom_command(
        OUTPUT "${abi_gapid_apk}"
        COMMAND "${CMAKE_COMMAND}" -E copy
            "${gradle_out}"
            "${abi_gapid_apk}"
        DEPENDS
            "${gradle_out}"
    )
    add_custom_target("${abi}-gapid-apk" ALL DEPENDS ${abi_gapid_apk})

    install(FILES "${abi_gapid_apk}" DESTINATION ${TARGET_INSTALL_PATH})
endforeach()
