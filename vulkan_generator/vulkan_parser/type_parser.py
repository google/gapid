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


import xml.etree.ElementTree as ET

from dataclasses import dataclass
from dataclasses import field
from typing import Dict

from vulkan_generator.vulkan_parser import handle_parser
from vulkan_generator.vulkan_parser import struct_parser
from vulkan_generator.vulkan_parser import funcptr_parser
from vulkan_generator.vulkan_parser import types


@dataclass
class AllVulkanTypes:
    """
    This class holds the information parsed from Vulkan XML
    This class should have all the information needed to generate code
    """
    # This class holds every Vulkan Type as [typename -> type]
    handles: Dict[str, types.VulkanHandle] = field(
        default_factory=dict)

    # Melih TODO: We probably need the map in two ways while generating code
    # both type -> alias and alias -> type
    # For now, lets store as the other types but when we do code generation,
    # We may have an extra step to convert the map to other direction.
    handle_aliases: Dict[str, types.VulkanHandleAlias] = field(
        default_factory=dict)

    structs: Dict[str, types.VulkanStruct] = field(
        default_factory=dict)

    struct_aliases: Dict[str, types.VulkanStructAlias] = field(
        default_factory=dict)

    funcpointers: Dict[str, types.VulkanFunctionPtr] = field(
        default_factory=dict)


def process_handle(vulkan_types: AllVulkanTypes, handle_element: ET.Element) -> None:
    """ Parse the Vulkan type "Handle". This can be an handle or an alias to another handle"""
    handle = handle_parser.parse(handle_element)

    if isinstance(handle, types.VulkanHandle):
        vulkan_types.handles[handle.typename] = handle
    elif isinstance(handle, types.VulkanHandleAlias):
        vulkan_types.handle_aliases[handle.typename] = handle


def process_struct(vulkan_types: AllVulkanTypes, struct_element: ET.Element) -> None:
    """ Parse the Vulkan type "Struct". This can be a struct or an alias to another struct"""
    vulkan_struct = struct_parser.parse(struct_element)

    if isinstance(vulkan_struct, types.VulkanStruct):
        vulkan_types.structs[vulkan_struct.typename] = vulkan_struct
    elif isinstance(vulkan_struct, types.VulkanStructAlias):
        vulkan_types.struct_aliases[vulkan_struct.typename] = vulkan_struct


def process_funcpointer(vulkan_types: AllVulkanTypes, func_ptr_element: ET.Element) -> None:
    """ Parse the Vulkan type "Funcpointer"""
    vulkan_func_ptr = funcptr_parser.parse(func_ptr_element)

    if isinstance(vulkan_func_ptr, types.VulkanFunctionPtr):
        vulkan_types.funcpointers[vulkan_func_ptr.typename] = vulkan_func_ptr


def parse(types_root: ET.Element) -> AllVulkanTypes:
    """ Parses all the Vulkan types and returns them in an object with dictionaries to each type """
    vulkan_types = AllVulkanTypes()

    for type_element in types_root.iter():
        if "category" in type_element.attrib:
            type_category = type_element.attrib["category"]
            if type_category == "handle":
                process_handle(vulkan_types, type_element)
            elif type_category == "struct":
                process_struct(vulkan_types, type_element)
            elif type_category == "funcpointer":
                process_funcpointer(vulkan_types, type_element)

    return vulkan_types
