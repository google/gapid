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


def process_enums(vulkan_types: types.AllVulkanTypes, enum_element: ET.Element) -> None:
    """Process the parsing of Vulkan enums"""
    # Enums are not under the types tag in the XML.
    # Therefore, they have to be handled separately.
    vulkan_enum = enums_parser.parse(enum_element)

    if not vulkan_enum:
        # API constants
        return

    vulkan_types.enums[vulkan_enum.typename] = vulkan_enum


def parse(filename: Path) -> types.AllVulkanTypes:
    """ Parse the Vulkan XML to extract every information that is needed for code generation"""
    tree = ET.parse(filename)
    all_types = types.AllVulkanTypes()

    for child in tree.iter():
        if child.tag == "types":
            all_types = type_parser.parse(child)
        if child.tag == "enums":
            process_enums(all_types, child)

    # Melih TODO: In the future this should return not only types
    # but other information that is needed as well. e.g. functions, comments etc.
    return all_types
