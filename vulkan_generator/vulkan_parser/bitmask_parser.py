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

""" This module is responsible for parsing Vulkan Bitmasks and their aliases.

    Vulkan bitmasks are special types that they are basically typedefed integer values
    that their all possible values are coming from an enum.

    e.g. VkFramebufferCreateFlags is VkFlags(which is uint32_t)
    and its values are member of VkFramebufferCreateFlagBits

    Unfortunately the actual enums defined under a different subtree of `enums` tag
    in Vulkan XML. Therefore they will be parsed by a different module.
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import parser_utils


def parse_bitmask_by_attribute(bitmask_elem: ET.Element) -> types.VulkanBitmaskAlias:
    """Parses a Vulkan Bitmask if it has the attribute "name" and returns it.

    if any bitmask defined like this, they are always an alias of an existing type

    Example from Vk.xml
    <type category="bitmask" name="VkDescriptorUpdateTemplateCreateFlagsKHR"
        alias="VkDescriptorUpdateTemplateCreateFlags"/>
    """

    name = bitmask_elem.attrib["name"]
    alias = bitmask_elem.attrib["alias"]
    return types.VulkanBitmaskAlias(typename=name, aliased_typename=alias)


def parse_bitmask_by_tag(bitmask_elem: ET.Element) -> types.VulkanBitmask:
    bitmask_name = parser_utils.get_text_from_tag_in_children(bitmask_elem, "name")
    # This is optional because there are flags that are not used but reserved for the future.
    bitfield_type = parser_utils.try_get_attribute(bitmask_elem, "requires")
    bitfield_basetype = parser_utils.get_text_from_tag_in_children(bitmask_elem, "type")

    return types.VulkanBitmask(
        typename=bitmask_name,
        field_type=bitfield_type,
        field_basetype=bitfield_basetype,
    )


def parse(bitmask_elem: ET.Element) -> types.VulkanType:
    """Returns a Vulkan bitmask or it's alias from the XML element that defines it.

    A sample Vulkan Bitmask:
    <type requires="VkFramebufferCreateFlagBits" category="bitmask">typedef
        <type>VkFlags</type> <name>VkFramebufferCreateFlags</name>;</type>

    A sample Vulkan Bitmask Alias:
    <type category="bitmask" name="VkDescriptorUpdateTemplateCreateFlagsKHR"
        alias="VkDescriptorUpdateTemplateCreateFlags"/>
    """

    if "name" in bitmask_elem.attrib:
        return parse_bitmask_by_attribute(bitmask_elem)

    return parse_bitmask_by_tag(bitmask_elem)
