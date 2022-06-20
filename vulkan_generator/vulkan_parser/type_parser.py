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

from vulkan_generator.vulkan_parser import bitmask_parser
from vulkan_generator.vulkan_parser import enum_aliases_parser
from vulkan_generator.vulkan_parser import handle_parser
from vulkan_generator.vulkan_parser import struct_parser
from vulkan_generator.vulkan_parser import funcptr_parser
from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import defines_parser
from vulkan_generator.vulkan_parser import basetype_parser


def process_defines(vulkan_types: types.AllVulkanTypes, define_element: ET.Element) -> None:
    vulkan_define = defines_parser.parse(define_element)

    if isinstance(vulkan_define, types.VulkanDefine):
        vulkan_types.defines[vulkan_define.variable_name] = vulkan_define
        return

    raise SyntaxError(f"Unknown Define: {vulkan_define}")


def process_basetype(vulkan_types: types.AllVulkanTypes, define_element: ET.Element) -> None:
    vulkan_basetype = basetype_parser.parse(define_element)

    if isinstance(vulkan_basetype, types.VulkanBaseType):
        vulkan_types.basetypes[vulkan_basetype.typename] = vulkan_basetype
        return

    raise SyntaxError(f"Unknown Basetype: {vulkan_basetype}")


def process_bitmask_alises(vulkan_types: types.AllVulkanTypes, bitmask_element: ET.Element) -> None:
    """ Parse the Vulkan bitmaks and their aliases"""
    vulkan_bitmask = bitmask_parser.parse(bitmask_element)

    if isinstance(vulkan_bitmask, types.VulkanBitmask):
        vulkan_types.bitmasks[vulkan_bitmask.typename] = vulkan_bitmask
        return

    if isinstance(vulkan_bitmask, types.VulkanBitmaskAlias):
        vulkan_types.bitmask_aliases[vulkan_bitmask.typename] = vulkan_bitmask
        return

    raise SyntaxError(f"Unknown Bitmask: {vulkan_bitmask}")


def process_enum_alises(vulkan_types: types.AllVulkanTypes, enum_element: ET.Element) -> None:
    """ Parse the Vulkan enum aliases"""
    vulkan_enum_value_alias = enum_aliases_parser.parse(enum_element)

    if vulkan_enum_value_alias:
        vulkan_types.enum_aliases[vulkan_enum_value_alias.typename] = vulkan_enum_value_alias


def process_handle(vulkan_types: types.AllVulkanTypes, handle_element: ET.Element) -> None:
    """ Parse the Vulkan type "Handle". This can be an handle or an alias to another handle"""
    handle = handle_parser.parse(handle_element)

    if isinstance(handle, types.VulkanHandle):
        vulkan_types.handles[handle.typename] = handle
        return

    if isinstance(handle, types.VulkanHandleAlias):
        vulkan_types.handle_aliases[handle.typename] = handle
        return

    raise SyntaxError(f"Unknown VulkanType: {handle}")


def process_struct(vulkan_types: types.AllVulkanTypes, struct_element: ET.Element) -> None:
    """ Parse the Vulkan type "Struct". This can be a struct or an alias to another struct"""
    vulkan_struct = struct_parser.parse(struct_element)

    if isinstance(vulkan_struct, types.VulkanStruct):
        vulkan_types.structs[vulkan_struct.typename] = vulkan_struct
        return

    if isinstance(vulkan_struct, types.VulkanStructAlias):
        vulkan_types.struct_aliases[vulkan_struct.typename] = vulkan_struct
        return

    raise SyntaxError(f"Unknown VulkanType {vulkan_struct}")


def process_funcpointer(vulkan_types: types.AllVulkanTypes, func_ptr_element: ET.Element) -> None:
    """ Parse the Vulkan type "Funcpointer"""
    vulkan_func_ptr = funcptr_parser.parse(func_ptr_element)
    vulkan_types.funcpointers[vulkan_func_ptr.typename] = vulkan_func_ptr


def parse(types_root: ET.Element) -> types.AllVulkanTypes:
    """ Parses all the Vulkan types and returns them in an object with dictionaries to each type """
    vulkan_types = types.AllVulkanTypes()

    for type_element in types_root.iter():
        if "category" in type_element.attrib:
            type_category = type_element.attrib["category"]
            if type_category == "define":
                process_defines(vulkan_types, type_element)
            elif type_category == "basetype":
                process_basetype(vulkan_types, type_element)
            elif type_category == "bitmask":
                process_bitmask_alises(vulkan_types, type_element)
            elif type_category == "enum":
                process_enum_alises(vulkan_types, type_element)
            elif type_category == "handle":
                process_handle(vulkan_types, type_element)
            elif type_category == "struct":
                process_struct(vulkan_types, type_element)
            elif type_category == "funcpointer":
                process_funcpointer(vulkan_types, type_element)

    return vulkan_types
