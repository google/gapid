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

"""This module is the Vulkan parser that extracts information from Vulkan XML"""

from pathlib import Path
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import type_parser
from vulkan_generator.vulkan_parser import enums_parser
from vulkan_generator.vulkan_parser import commands_parser


def process_enums(vulkan_types: types.AllVulkanTypes, enum_element: ET.Element) -> None:
    """Process the parsing of Vulkan enums"""
    # Enums are not under the types tag in the XML.
    # Therefore, they have to be handled separately.
    vulkan_enums = enums_parser.parse(enum_element)

    if not vulkan_enums:
        raise SyntaxError(f"Enum could not be parsed {ET.tostring(enum_element, 'utf-8')}")

    if isinstance(vulkan_enums, types.VulkanEnum):
        vulkan_types.enums[vulkan_enums.typename] = vulkan_enums
        return

    # Some Vulkan defines are under enums tag. Therefore we need to parse them here.
    if isinstance(vulkan_enums, dict):
        for define in vulkan_enums.values():
            vulkan_types.defines[define.variable_name] = define
        return

    raise SyntaxError(f"Unknown define or enum {vulkan_enums}")


def parse(filename: Path) -> types.VulkanMetadata:
    """ Parse the Vulkan XML to extract every information that is needed for code generation"""
    tree = ET.parse(filename)
    all_types = types.AllVulkanTypes()
    all_commands = types.AllVulkanCommands()

    for child in tree.iter():
        if child.tag == "types":
            all_types = type_parser.parse(child)
        elif child.tag == "enums":
            process_enums(all_types, child)
        elif child.tag == "commands":
            all_commands = commands_parser.parse(child)

    return types.VulkanMetadata(types=all_types, commands=all_commands)
