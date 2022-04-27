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
This package is responsible for testing Vulkan Parser

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_parser import struct_parser
from vulkan_parser import types


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

    assert isinstance(typ, types.VulkanStruct)
    assert typ.typename == "VkDevicePrivateDataCreateInfo"

    assert len(typ.members) == 3
    assert "sType" in typ.members
    assert typ.members["sType"].typename == "VkStructureType"

    assert "pNext" in typ.members
    assert typ.members["pNext"].typename == "const void*"

    assert "privateDataSlotRequestCount" in typ.members
    assert typ.members["privateDataSlotRequestCount"].typename == "uint32_t"


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

    assert isinstance(typ, types.VulkanStruct)
    assert typ.typename == "VkInstanceCreateInfo"

    assert "ppEnabledLayerNames" in typ.members
    assert typ.members["ppEnabledLayerNames"].typename == "const char* const*"


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
    assert isinstance(typ, types.VulkanStruct)
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

    assert isinstance(typ, types.VulkanStruct)
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

    assert isinstance(typ, types.VulkanStruct)
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
    assert isinstance(typ, types.VulkanStruct)

    reference = typ.members["pBinds"].array_reference
    assert reference in typ.members


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
    assert isinstance(typ, types.VulkanStruct)

    assert "deviceName" in typ.members
    assert typ.members["deviceName"].variable_size == "VK_MAX_PHYSICAL_DEVICE_NAME_SIZE"


def test_vulkan_struct_alias() -> None:
    """"Tests Vulkan struct alias"""
    xml = """<?xml version="1.0" encoding="UTF-8"?>
        <type category="struct" name="VkDevicePrivateDataCreateInfoEXT"
        alias="VkDevicePrivateDataCreateInfo"/>
    """

    typ = struct_parser.parse(ET.fromstring(xml))

    assert isinstance(typ, types.VulkanStructAlias)
    assert typ.typename == "VkDevicePrivateDataCreateInfoEXT"
    assert typ.aliased_typename == "VkDevicePrivateDataCreateInfo"
