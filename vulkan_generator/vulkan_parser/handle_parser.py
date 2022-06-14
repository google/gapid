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

""" This module is responsible for parsing Vulkan Handles and aliases of them"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_utils import parsing_utils
from vulkan_generator.vulkan_parser import types


def parse_handle_by_attribute(root: ET.Element) -> types.VulkanHandleAlias:
    """Parses a Vulkan Handle if it has the attribute "name" and returns it.

    if any handle defined like this, they are always an alias of an existing type

    Example from Vk.xml
    <type category="handle" name="VkDescriptorUpdateTemplateKHR"
    alias="VkDescriptorUpdateTemplate"/>
    """
    name = root.attrib["name"]
    alias = root.attrib["alias"]
    vulkan_handle = types.VulkanHandleAlias(typename=name, aliased_typename=alias)
    return vulkan_handle


def parse_handle_by_tag(root: ET.Element) -> types.VulkanHandle:
    """Parses a Vulkan Handle if it has the tag "name" and returns it.

    Example from Vk.xml
    <type category="handle" parent="VkDevice" objtypeenum="VK_OBJECT_TYPE_QUEUE">
    <type>VK_DEFINE_HANDLE</type>(<name>VkQueue</name>)</type>
    """

    name = parsing_utils.get_text_from_tag_in_children(root, "name")
    handle_definer = parsing_utils.get_text_from_tag_in_children(root, "type")

    if handle_definer not in ["VK_DEFINE_HANDLE", "VK_DEFINE_NON_DISPATCHABLE_HANDLE"]:
        raise SyntaxError(f"Unknown Handle definer {ET.tostring(root, 'utf-8')}")

    dispatchable = handle_definer == "VK_DEFINE_HANDLE"

    vulkan_handle = types.VulkanHandle(typename=name, dispatchable=dispatchable)
    return vulkan_handle


def parse(root: ET.Element) -> types.VulkanType:
    """Returns a Vulkan handle or alias from the XML element that defines it"""
    if "name" in root.attrib:
        return parse_handle_by_attribute(root)

    return parse_handle_by_tag(root)
