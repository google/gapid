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

from vulkan_generator.vulkan_parser.internal import struct_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def test_vulkan_struct_with_members() -> None:
    """"tests a Vulkan struct with members"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkDevicePrivateDataCreateInfo"
        allowduplicate="true" structextends="VkDeviceCreateInfo">

        <member values="VK_STRUCTURE_TYPE_DEVICE_PRIVATE_DATA_CREATE_INFO">
            <type>VkStructureType</type> <name>sType</name>
        </member>
        <member optional="true">const <type>void</type>*<name>pNext</name></member>
        <member><type>uint32_t</type> <name>privateDataSlotRequestCount</name></member>
    </type>
    """

    typ = struct_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanStruct)
    assert typ.typename == "VkDevicePrivateDataCreateInfo"

    assert len(typ.members) == 3

    member_names = list(typ.members.keys())

    assert member_names[0] == "sType"
    assert typ.members["sType"].member_type == "VkStructureType"
    assert typ.members["sType"].parent == "VkDevicePrivateDataCreateInfo"

    assert member_names[1] == "pNext"
    assert typ.members["pNext"].member_type == "const void*"
    assert typ.members["pNext"].parent == "VkDevicePrivateDataCreateInfo"

    assert member_names[2] == "privateDataSlotRequestCount"
    assert typ.members["privateDataSlotRequestCount"].member_type == "uint32_t"
    assert typ.members["privateDataSlotRequestCount"].parent == "VkDevicePrivateDataCreateInfo"


def test_vulkan_struct_with_const_and_pointer() -> None:
    """"tests a Vulkan struct with a double const pointer member"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkInstanceCreateInfo">
            <member values="VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO"><type>VkStructureType</type>
            <name>sType</name></member>
            <member optional="true">const <type>void</type>*     <name>pNext</name></member>
            <member optional="true"><type>VkInstanceCreateFlags</type>  <name>flags</name></member>
            <member optional="true">const <type>VkApplicationInfo</type>*
            <name>pApplicationInfo</name></member>
            <member optional="true"><type>uint32_t</type>
                           <name>enabledLayerCount</name></member>
            <member len="enabledLayerCount,null-terminated">const <type>char</type>
            * const*      <name>ppEnabledLayerNames</name>
            <comment>Ordered list of layer names to be enabled</comment></member>
            <member optional="true"><type>uint32_t</type>
                           <name>enabledExtensionCount</name></member>
            <member len="enabledExtensionCount,null-terminated">
            const <type>char</type>* const*      <name>ppEnabledExtensionNames</name
            ><comment>Extension names to be enabled</comment></member>
        </type>
    """
    typ = struct_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanStruct)
    assert typ.typename == "VkInstanceCreateInfo"

    assert typ.members["ppEnabledLayerNames"].member_type == "const char* const*"


def test_vulkan_struct_with_expected_value() -> None:
    """"tests a Vulkan struct with a member that has an expected value"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkDevicePrivateDataCreateInfo"
        allowduplicate="true" structextends="VkDeviceCreateInfo">

        <member values="VK_STRUCTURE_TYPE_DEVICE_PRIVATE_DATA_CREATE_INFO">
            <type>VkStructureType</type> <name>sType</name>
        </member>
        <member optional="true">const <type>void</type>*<name>pNext</name></member>
        <member><type>uint32_t</type> <name>privateDataSlotRequestCount</name></member>
    </type>
    """

    typ = struct_parser.parse(ET.fromstring(xml))
    assert isinstance(typ, internal_types.VulkanStruct)

    expected_value = "VK_STRUCTURE_TYPE_DEVICE_PRIVATE_DATA_CREATE_INFO"
    assert typ.members["sType"].expected_value == expected_value


def test_vulkan_struct_with_optional() -> None:
    """"tests a Vulkan struct with an optional member"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkDevicePrivateDataCreateInfo"
        allowduplicate="true" structextends="VkDeviceCreateInfo">

        <member values="VK_STRUCTURE_TYPE_DEVICE_PRIVATE_DATA_CREATE_INFO">
            <type>VkStructureType</type> <name>sType</name>
        </member>
        <member optional="true">const <type>void</type>*<name>pNext</name></member>
        <member><type>uint32_t</type> <name>privateDataSlotRequestCount</name></member>
    </type>
    """
    typ = struct_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanStruct)
    assert typ.members["pNext"].optional


def test_vulkan_struct_with_no_auto_validity() -> None:
    """"tests a Vulkan struct with a member has no auto validity"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkSpecializationMapEntry">
            <member><type>uint32_t</type>
                                 <name>constantID</name>
            <comment>The SpecConstant ID specified in the BIL</comment></member>
            <member><type>uint32_t</type>
                                 <name>offset</name>
            <comment>Offset of the value in the data block</comment></member>
            <member noautovalidity="true"><type>size_t</type> <name>size</name>
            <comment>Size in bytes of the SpecConstant</comment></member>
        </type>
    """
    typ = struct_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanStruct)
    assert typ.members["size"].no_auto_validity


def test_vulkan_struct_with_dynamic_array() -> None:
    """"tests a Vulkan struct with a dynamic array as a member"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkSparseBufferMemoryBindInfo">
            <member><type>VkBuffer</type> <name>buffer</name></member>
            <member><type>uint32_t</type>               <name>bindCount</name></member>
            <member len="bindCount">const <type>VkSparseMemoryBind</type>
            * <name>pBinds</name></member>
        </type>
    """
    typ = struct_parser.parse(ET.fromstring(xml))
    assert isinstance(typ, internal_types.VulkanStruct)

    reference = typ.members["pBinds"].size
    assert reference
    assert len(reference) == 1

    member_names = list(typ.members.keys())
    assert reference[0] == member_names[1]


def test_vulkan_struct_with_dynamic_array_alternative_length() -> None:
    """"tests a Vulkan struct with a dynamic array as a member who has an alternative length"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkAccelerationStructureVersionInfoKHR">
            <member values="VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_VERSION_INFO_KHR">
                <type>VkStructureType</type> <name>sType</name></member>
            <member optional="true">const <type>void</type>*
                <name>pNext</name></member>
            <member len="latexmath:[2 \times \\mathtt{VK\\_UUID\\_SIZE}]" altlen="2*VK_UUID_SIZE">const
                <type>uint8_t</type>*                    <name>pVersionData</name></member>
        </type>
    """
    typ = struct_parser.parse(ET.fromstring(xml))
    assert isinstance(typ, internal_types.VulkanStruct)

    reference = typ.members["pVersionData"].size
    assert reference
    assert len(reference) == 1
    assert reference[0] == "2*VK_UUID_SIZE"


def test_vulkan_struct_with_static_array() -> None:
    """"Tests a Vulkan struct with a static array as a member"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="struct" name="VkPhysicalDeviceProperties" returnedonly="true">
            <member limittype="noauto"><type>uint32_t</type>
                   <name>apiVersion</name></member>
            <member limittype="noauto"><type>uint32_t</type>
                   <name>driverVersion</name></member>
            <member limittype="noauto"><type>uint32_t</type>
                   <name>vendorID</name></member>
            <member limittype="noauto"><type>uint32_t</type>
                   <name>deviceID</name></member>
            <member limittype="noauto"><type>VkPhysicalDeviceType</type>
             <name>deviceType</name></member>
            <member limittype="noauto"><type>char</type>           <name>deviceName</name>
            [<enum>VK_MAX_PHYSICAL_DEVICE_NAME_SIZE</enum>]</member>
            <member limittype="noauto"><type>uint8_t</type>
                    <name>pipelineCacheUUID</name>[<enum>VK_UUID_SIZE</enum>]</member>
            <member limittype="struct"><type>VkPhysicalDeviceLimits</type>
             <name>limits</name></member>
            <member limittype="struct"><type>VkPhysicalDeviceSparseProperties</type>
             <name>sparseProperties</name></member>
        </type>
    """

    typ = struct_parser.parse(ET.fromstring(xml))
    assert isinstance(typ, internal_types.VulkanStruct)

    reference = typ.members["deviceName"].size
    assert reference
    assert len(reference) == 1
    assert reference[0] == "VK_MAX_PHYSICAL_DEVICE_NAME_SIZE"


def test_vulkan_struct_with_multidimensional_array() -> None:
    """"Tests a Vulkan struct with a static array as a member"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="struct" name="VkTransformMatrixKHR">
            <member><type>float</type><name>matrix</name>[3][4]</member>
        </type>
    """

    typ = struct_parser.parse(ET.fromstring(xml))
    assert isinstance(typ, internal_types.VulkanStruct)

    size = typ.members["matrix"].size
    assert size
    assert len(size) == 2

    assert size[0] == "3"
    assert size[1] == "4"


def test_vulkan_struct_with_bitfield_size() -> None:
    """"Tests a Vulkan struct with a static array as a member"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="struct" name="VkAccelerationStructureInstanceKHR">
        <comment>The bitfields in this structure are non-normative since bitfield ordering is implementation-defined in C.
            The specification defines the normative layout.</comment>
        <member><type>VkTransformMatrixKHR</type>                                    <name>transform</name></member>
        <member><type>uint32_t</type>                                                <name>instanceCustomIndex</name>:24</member>
        <member><type>uint32_t</type>                                                <name>mask</name>:8</member>
        <member><type>uint32_t</type>                                                <name>instanceShaderBindingTableRecordOffset</name>:24</member>
        <member optional="true"><type>VkGeometryInstanceFlagsKHR</type>              <name>flags</name>:8</member>
        <member><type>uint64_t</type>                                                <name>accelerationStructureReference</name></member>
    </type>
    """

    typ = struct_parser.parse(ET.fromstring(xml))
    assert isinstance(typ, internal_types.VulkanStruct)

    assert typ.members["mask"].c_bitfield_size == 8


def test_vulkan_struct_with_multiple_base_struct() -> None:
    """"Tests a Vulkan struct with multiple base struct"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="struct" name="VkPhysicalDevicePrivateDataFeatures"
        structextends="VkPhysicalDeviceFeatures2,VkDeviceCreateInfo">
            <member values="VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PRIVATE_DATA_FEATURES">
                <type>VkStructureType</type> <name>sType</name></member>
            <member optional="true"><type>void</type>*
                <name>pNext</name></member>
            <member><type>VkBool32</type>
                <name>privateData</name></member>
        </type>
    """

    typ = struct_parser.parse(ET.fromstring(xml))
    assert isinstance(typ, internal_types.VulkanStruct)
    assert typ.base_structs
    assert len(typ.base_structs) == 2
    assert "VkPhysicalDeviceFeatures2" in typ.base_structs
    assert "VkDeviceCreateInfo" in typ.base_structs


def test_vulkan_struct_alias() -> None:
    """"Tests Vulkan struct alias"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkDevicePrivateDataCreateInfoEXT"
        alias="VkDevicePrivateDataCreateInfo"/>
    """

    typ = struct_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, internal_types.VulkanStructAlias)
    assert typ.typename == "VkDevicePrivateDataCreateInfoEXT"
    assert typ.aliased_typename == "VkDevicePrivateDataCreateInfo"
