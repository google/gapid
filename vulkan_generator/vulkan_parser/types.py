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

"""This module contains the Vulkan meta data definitions to define Vulkan types and functions"""

from dataclasses import dataclass
from dataclasses import field
from typing import Dict
from typing import List
from typing import Optional


@dataclass
class VulkanType:
    """Base class for a Vulkan Type. All the other Types should inherit from this class"""
    typename: str
    # Some vulkan types have comments, which are complicated to parse with regards to
    # which comment belongs to which element. We are skipping them now, if they would be
    # useful, we can consider parsing them


@dataclass
class VulkanHandle(VulkanType):
    """The meta data defines a Vulkan Handle"""
    # Melih TODO: Vulkan Handles have object type in the XML that might be required in the future


@dataclass
class VulkanHandleAlias(VulkanType):
    """The meta data defines a Vulkan Handle alias"""
    aliased_typename: str


@dataclass
class VulkanStructMember(VulkanType):
    """The meta data defines a Vulkan Handle"""
    variable_name: str

    # Some member variables are static arrays with a default size
    variable_size: Optional[str]

    # Some members have this property which states if that particular
    # member has to be valid if they are not null
    no_auto_validity: Optional[bool]

    # Melih TODO: In the future we probably need to change
    # this from str to VulkanEnum.
    # Does member has an expected value e.g. sType
    expected_value: Optional[str]

    # If the member is an array, it's size is defined by another
    # member in the struct. This is the name of the referring member
    array_reference: Optional[str]

    # Is this field has to be set and/or not-null
    optional: Optional[bool]

    # Melih TODO: Currently put the pointer and const info directly
    # into the type name. If we need to extract it later, we extract from the
    # typename with helper functions


@dataclass
class VulkanStruct(VulkanType):
    """The meta data defines a Vulkan Handle"""
    members: Dict[str, VulkanStructMember] = field(
        default_factory=dict)


@dataclass
class VulkanStructAlias(VulkanType):
    """The meta data defines a Vulkan Handle alias"""
    aliased_typename: str


@dataclass
class VulkanFunctionArgument(VulkanType):
    """The meta data defines a Function argument"""
    argument_name: str


@dataclass
class VulkanFunctionPtr(VulkanType):
    """The meta data defines a Function Pointer"""
    return_type: str
    arguments: List[VulkanFunctionArgument] = field(
        default_factory=list)
