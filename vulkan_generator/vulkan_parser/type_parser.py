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

"""This module is responsible for parsing Vulkan Types"""

import dataclasses
import xml.etree.ElementTree as ET

from dataclasses import dataclass
from typing import Dict

from vulkan_parser import types
from vulkan_parser import handle_parser


@dataclass
class AllVulkanTypes:
    """
    This class holds the information parsed from Vulkan XML
    This class should have all the information needed to generate code
    """
    # This class holds every Vulkan Type as [typename -> type]
    handles: Dict[str, types.VulkanHandle] = dataclasses.field(
        default_factory=dict)

    # Melih TODO: We probably need the map in two ways while generating code
    # both type -> alias and alias -> type
    # For now, lets store as the other types but when we do code generation,
    # We may have an extra step to convert the map to other direction.
    handle_aliases: Dict[str, types.VulkanHandleAlias] = dataclasses.field(
        default_factory=dict)


def process_handle(vulkan_types: AllVulkanTypes, handle_element: ET.Element) -> None:
    """ Parse the Vulkan type "Handle". This can be an handle or an alias to another handle """
    handle = handle_parser.parse(handle_element)

    if isinstance(handle, types.VulkanHandle):
        vulkan_types.handles[handle.typename] = handle
    elif isinstance(handle, types.VulkanHandleAlias):
        vulkan_types.handle_aliases[handle.typename] = handle


def parse(root: ET.Element) -> AllVulkanTypes:
    """ Parses all the Vulkan types and returns them in an object with dictionaries to each type """
    vulkan_types = AllVulkanTypes()

    for child in root:
        if 'category' in child.attrib:
            type_category = child.attrib['category']
            if type_category == "handle":
                process_handle(vulkan_types, child)

    return vulkan_types
