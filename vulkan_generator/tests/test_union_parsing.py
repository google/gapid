# Copyright (C) 2022 Google Inc.
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

"""
This module is responsible for testing Vulkan structs and aliases

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import union_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def test_vulkan_union() -> None:
    """"tests a Vulkan union with members"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
       <type category="union" name="VkPerformanceCounterResultKHR"
        comment="// Union of all the possible return types a counter result could return">
            <member><type>int32_t</type>  <name>int32</name></member>
            <member><type>int64_t</type>  <name>int64</name></member>
            <member><type>uint32_t</type> <name>uint32</name></member>
            <member><type>uint64_t</type> <name>uint64</name></member>
            <member><type>float</type>    <name>float32</name></member>
            <member><type>double</type>   <name>float64</name></member>
        </type>
    """

    typ = union_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanUnion)
    assert typ.typename == "VkPerformanceCounterResultKHR"

    assert len(typ.members) == 6

    member_names = list(typ.members)

    assert member_names[0] == "int32"
    assert typ.members["int32"].member_type == "int32_t"

    assert member_names[1] == "int64"
    assert typ.members["int64"].member_type == "int64_t"

    assert member_names[2] == "uint32"
    assert typ.members["uint32"].member_type == "uint32_t"

    assert member_names[3] == "uint64"
    assert typ.members["uint64"].member_type == "uint64_t"

    assert member_names[4] == "float32"
    assert typ.members["float32"].member_type == "float"

    assert member_names[5] == "float64"
    assert typ.members["float64"].member_type == "double"


def test_vulkan_union_with_returned_only() -> None:
    """"tests a Vulkan union with members"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
       <type category="union" name="VkPipelineExecutableStatisticValueKHR" returnedonly="true">
            <member selection="VK_PIPELINE_EXECUTABLE_STATISTIC_FORMAT_BOOL32_KHR">
                <type>VkBool32</type>           <name>b32</name></member>
            <member selection="VK_PIPELINE_EXECUTABLE_STATISTIC_FORMAT_INT64_KHR">
                <type>int64_t</type>            <name>i64</name></member>
            <member selection="VK_PIPELINE_EXECUTABLE_STATISTIC_FORMAT_UINT64_KHR">
                <type>uint64_t</type>           <name>u64</name></member>
            <member selection="VK_PIPELINE_EXECUTABLE_STATISTIC_FORMAT_FLOAT64_KHR">
                <type>double</type>             <name>f64</name></member>
        </type>
    """

    typ = union_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanUnion)
    assert typ.returned_only


def test_vulkan_union_with_no_auto_validity() -> None:
    """"tests a Vulkan union with no auto validity member"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
       <type category="union" name="VkClearValue" comment="// DELETED ON PURPOSE">
            <member noautovalidity="true"><type>VkClearColorValue</type>      <name>color</name></member>
            <member><type>VkClearDepthStencilValue</type> <name>depthStencil</name></member>
        </type>
    """

    typ = union_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanUnion)
    assert typ.members["color"].no_auto_validity
    assert not typ.members["depthStencil"].no_auto_validity


def test_vulkan_union_with_selection() -> None:
    """"tests a Vulkan union with selection"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
      <type category="union" name="VkAccelerationStructureMotionInstanceDataNV">
            <member selection="VK_ACCELERATION_STRUCTURE_MOTION_INSTANCE_TYPE_STATIC_NV">
                <type>VkAccelerationStructureInstanceKHR</type>            <name>staticInstance</name></member>
            <member selection="VK_ACCELERATION_STRUCTURE_MOTION_INSTANCE_TYPE_MATRIX_MOTION_NV">
                <type>VkAccelerationStructureMatrixMotionInstanceNV</type> <name>matrixMotionInstance</name></member>
            <member selection="VK_ACCELERATION_STRUCTURE_MOTION_INSTANCE_TYPE_SRT_MOTION_NV">
                <type>VkAccelerationStructureSRTMotionInstanceNV</type>    <name>srtMotionInstance</name></member>
        </type>
    """

    typ = union_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanUnion)
    assert typ.members["staticInstance"].selection == "VK_ACCELERATION_STRUCTURE_MOTION_INSTANCE_TYPE_STATIC_NV"

    value = "VK_ACCELERATION_STRUCTURE_MOTION_INSTANCE_TYPE_MATRIX_MOTION_NV"
    assert typ.members["matrixMotionInstance"].selection == value
    assert typ.members["srtMotionInstance"].selection == "VK_ACCELERATION_STRUCTURE_MOTION_INSTANCE_TYPE_SRT_MOTION_NV"


def test_vulkan_union_with_array() -> None:
    """"tests a Vulkan union with array"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
      <type category="union" name="VkClearColorValue" comment="//DELETED ON PURPOSE.">
            <member><type>float</type>                  <name>float32</name>[4]</member>
            <member><type>int32_t</type>                <name>int32</name>[4]</member>
            <member><type>uint32_t</type>               <name>uint32</name>[4]</member>
        </type>
    """

    typ = union_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanUnion)

    size_float32 = typ.members["float32"].size
    assert size_float32
    assert len(size_float32) == 1
    assert size_float32[0] == "4"

    size_int32 = typ.members["int32"].size
    assert size_int32
    assert len(size_int32) == 1
    assert size_int32[0] == "4"

    size_uint32 = typ.members["float32"].size
    assert size_uint32
    assert len(size_uint32) == 1
    assert size_uint32[0] == "4"
