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

"""This module contains the utilities to query information on Vulkan Types"""

from typing import Optional
from vulkan_generator.vulkan_parser.api import types


def try_get_type_or_deducted_type(typename: str, vulkan_types: types.VulkanTypeInfo) -> Optional[types.VulkanType]:
    """This function will return the type object of given typename.
    If type cannot found, then it will look for the type alias with given typename and return the aliased type of it
    If no type alias can be found with given name, then it will return None"""
    if typename in vulkan_types.all_types:
        return vulkan_types.all_types[typename]
    elif typename in vulkan_types.all_aliases:
        return vulkan_types.all_aliases[typename].aliased_type
    else:
        return None


def get_type_or_deducted_type(typename: str, vulkan_types: types.VulkanTypeInfo) -> types.VulkanType:
    """This function will return the type object of given typename.
    If type cannot found, then it will look for the type alias with given typename and return the aliased type of it"""
    if typename in vulkan_types.all_types:
        return vulkan_types.all_types[typename]

    return vulkan_types.all_aliases[typename].aliased_type


def get_struct_or_deducted_struct(typename: str, vulkan_types: types.VulkanTypeInfo) -> types.VulkanStruct:
    """This function will return the type object of given typename.
    If type cannot found, then it will look for the type alias with given typename and return the aliased type of it"""
    if typename in vulkan_types.structs:
        return vulkan_types.structs[typename]

    aliased_struct = vulkan_types.struct_aliases[typename].aliased_type

    if not isinstance(aliased_struct, types.VulkanStruct):
        raise ValueError(f"Unexpected aliased type: {aliased_struct.typename}")

    return aliased_struct


def get_enum_field_or_deducted_field(enum: types.VulkanEnum, field_name: str) -> types.VulkanEnumField:
    """
    This will return the field for the field name.
    If no field cannot be found, then it will search the alias and deduct the field from alias and return it.
    """
    if field_name in enum.fields:
        return enum.fields[field_name]

    return enum.aliases[field_name].aliased_field
