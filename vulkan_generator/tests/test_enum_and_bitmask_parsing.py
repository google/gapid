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

from vulkan_generator.vulkan_parser import bitmask_parser
from vulkan_generator.vulkan_parser import enums_parser
from vulkan_generator.vulkan_parser import enum_aliases_parser
from vulkan_generator.vulkan_parser import types


def test_vulkan_bitmask_with_require() -> None:
    """""Test the bitmask has "requires" field"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type requires="VkFramebufferCreateFlagBits" category="bitmask">typedef
        <type>VkFlags</type> <name>VkFramebufferCreateFlags</name>;</type>"""

    bitmask = bitmask_parser.parse(ET.fromstring(xml))

    assert isinstance(bitmask, types.VulkanBitmask)
    assert bitmask.typename == "VkFramebufferCreateFlags"
    assert bitmask.field_type == "VkFramebufferCreateFlagBits"
    assert bitmask.field_basetype == "VkFlags"


def test_vulkan_bitmask_without_require() -> None:
    """""Test the bitmask does not have a "requires" field"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="bitmask">typedef <type>VkFlags</type>
        <name>VkQueryPoolCreateFlags</name>;</type>"""

    bitmask = bitmask_parser.parse(ET.fromstring(xml))

    assert isinstance(bitmask, types.VulkanBitmask)
    assert bitmask.field_type is None
    assert bitmask.field_basetype == "VkFlags"


def test_vulkan_64_bit_bitmask() -> None:
    """""Test a 64 bit bitmask"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type bitvalues="VkFormatFeatureFlagBits2" category="bitmask">typedef <type>VkFlags64</type>
        <name>VkFormatFeatureFlags2</name>;</type>"""

    bitmask = bitmask_parser.parse(ET.fromstring(xml))

    assert isinstance(bitmask, types.VulkanBitmask)
    assert bitmask.field_basetype == "VkFlags64"


def test_vulkan_bitmask_alias() -> None:
    """""Test bitmask alias"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="bitmask" name="VkDescriptorBindingFlagsEXT" alias="VkDescriptorBindingFlags"/>"""

    bitmask = bitmask_parser.parse(ET.fromstring(xml))

    assert isinstance(bitmask, types.VulkanBitmaskAlias)
    assert bitmask.typename == "VkDescriptorBindingFlagsEXT"
    assert bitmask.aliased_typename == "VkDescriptorBindingFlags"


def test_enum_alias() -> None:
    """Test an enum with value fields"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="enum" name="VkResolveModeFlagBitsKHR" alias="VkResolveModeFlagBits"/>
    """

    enum_alias = enum_aliases_parser.parse(ET.fromstring(xml))
    assert isinstance(enum_alias, types.VulkanEnumAlias)
    assert enum_alias.typename == "VkResolveModeFlagBitsKHR"
    assert enum_alias.aliased_typename == "VkResolveModeFlagBits"


def test_enum_with_value_fields() -> None:
    """Test an enum with value fields"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <enums name="VkCommandBufferLevel" type="enum">
        <enum value="0"     name="VK_COMMAND_BUFFER_LEVEL_PRIMARY"/>
        <enum value="1"     name="VK_COMMAND_BUFFER_LEVEL_SECONDARY"/>
    </enums>
    """

    vulkan_enum = enums_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_enum, types.VulkanEnum)
    assert vulkan_enum.typename == "VkCommandBufferLevel"
    assert not vulkan_enum.bitmask
    assert not vulkan_enum.bit64

    assert vulkan_enum.field_order[0] == "VK_COMMAND_BUFFER_LEVEL_PRIMARY"
    assert vulkan_enum.fields["VK_COMMAND_BUFFER_LEVEL_PRIMARY"].value == 0
    assert vulkan_enum.fields["VK_COMMAND_BUFFER_LEVEL_PRIMARY"].representation == "0"

    assert vulkan_enum.field_order[1] == "VK_COMMAND_BUFFER_LEVEL_SECONDARY"
    assert vulkan_enum.fields["VK_COMMAND_BUFFER_LEVEL_SECONDARY"].value == 1
    assert vulkan_enum.fields["VK_COMMAND_BUFFER_LEVEL_SECONDARY"].representation == "1"


def test_enum_with_bitmask_fields() -> None:
    """Test an enum with bitmask fields"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <enums name="VkCommandPoolCreateFlagBits" type="bitmask">
        <enum bitpos="0" name="VK_COMMAND_POOL_CREATE_TRANSIENT_BIT" comment="Command buffers have a short lifetime"/>
        <enum bitpos="1" name="VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT"
            comment="Command buffers may release their memory individually"/>
    </enums>
    """

    vulkan_enum = enums_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_enum, types.VulkanEnum)
    assert vulkan_enum.typename == "VkCommandPoolCreateFlagBits"
    assert vulkan_enum.bitmask
    assert not vulkan_enum.bit64

    assert vulkan_enum.field_order[0] == "VK_COMMAND_POOL_CREATE_TRANSIENT_BIT"
    assert vulkan_enum.fields["VK_COMMAND_POOL_CREATE_TRANSIENT_BIT"].value == 1
    assert vulkan_enum.fields["VK_COMMAND_POOL_CREATE_TRANSIENT_BIT"].representation == "0x00000001"

    assert vulkan_enum.field_order[1] == "VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT"
    assert vulkan_enum.fields["VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT"].value == 2
    assert vulkan_enum.fields["VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT"].representation == "0x00000002"


def test_enum_with_64bit_bitmask_fields() -> None:
    """Test an enum with 64 bit bitmask fields"""

    # Skip the middle part of the XML as it is very long an unnecessary
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <enums name="VkAccessFlagBits2" type="bitmask" bitwidth="64">
        <enum bitpos="0" name="VK_ACCESS_2_INDIRECT_COMMAND_READ_BIT"/>
        <enum bitpos="32" name="VK_ACCESS_2_SHADER_SAMPLED_READ_BIT"/>
    </enums>
    """

    vulkan_enum = enums_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_enum, types.VulkanEnum)
    assert vulkan_enum.typename == "VkAccessFlagBits2"
    assert vulkan_enum.bitmask
    assert vulkan_enum.bit64

    assert vulkan_enum.field_order[0] == "VK_ACCESS_2_INDIRECT_COMMAND_READ_BIT"
    assert vulkan_enum.fields["VK_ACCESS_2_INDIRECT_COMMAND_READ_BIT"].value == 1
    assert vulkan_enum.fields["VK_ACCESS_2_INDIRECT_COMMAND_READ_BIT"].representation == "0x00000001ULL"

    assert vulkan_enum.field_order[1] == "VK_ACCESS_2_SHADER_SAMPLED_READ_BIT"
    assert vulkan_enum.fields["VK_ACCESS_2_SHADER_SAMPLED_READ_BIT"].value == 2 ** 32
    assert vulkan_enum.fields["VK_ACCESS_2_SHADER_SAMPLED_READ_BIT"].representation == "0x100000000ULL"


def test_enum_with_both_value_and_bitmask_fields() -> None:
    """Test an enum has both value and bitmask fields"""

    # Skip the middle part of the XML as it is very long an unnecessary
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <enums name="VkCullModeFlagBits" type="bitmask">
        <enum value="0"     name="VK_CULL_MODE_NONE"/>
        <enum bitpos="0"    name="VK_CULL_MODE_FRONT_BIT"/>
        <enum bitpos="1"    name="VK_CULL_MODE_BACK_BIT"/>
        <enum value="0x00000003" name="VK_CULL_MODE_FRONT_AND_BACK"/>
    </enums>
    """

    vulkan_enum = enums_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_enum, types.VulkanEnum)

    assert vulkan_enum.fields["VK_CULL_MODE_NONE"].value == 0
    assert vulkan_enum.fields["VK_CULL_MODE_NONE"].representation == "0"

    assert vulkan_enum.fields["VK_CULL_MODE_FRONT_BIT"].value == 1
    assert vulkan_enum.fields["VK_CULL_MODE_FRONT_BIT"].representation == "0x00000001"

    assert vulkan_enum.fields["VK_CULL_MODE_FRONT_AND_BACK"].value == 3
    assert vulkan_enum.fields["VK_CULL_MODE_FRONT_AND_BACK"].representation == "0x00000003"


def test_enum_with_aliased_fields() -> None:
    """Test an enum has both value and bitmask fields"""

    # Skip the middle part of the XML as it is very long an unnecessary
    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <enums name="VkColorSpaceKHR" type="enum">
        <enum value="0" name="VK_COLOR_SPACE_SRGB_NONLINEAR_KHR"/>
        <enum name="VK_COLORSPACE_SRGB_NONLINEAR_KHR" alias="VK_COLOR_SPACE_SRGB_NONLINEAR_KHR"
            comment="Backwards-compatible alias containing a typo"/>
    </enums>
    """

    vulkan_enum = enums_parser.parse(ET.fromstring(xml))
    assert isinstance(vulkan_enum, types.VulkanEnum)

    assert vulkan_enum.field_order[0] == "VK_COLOR_SPACE_SRGB_NONLINEAR_KHR"
    assert vulkan_enum.aliases["VK_COLORSPACE_SRGB_NONLINEAR_KHR"] == "VK_COLOR_SPACE_SRGB_NONLINEAR_KHR"
