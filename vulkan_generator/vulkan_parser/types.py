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
from typing import OrderedDict
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
class VulkanBaseType(VulkanType):
    """Base class for a Vulkan basetype"""
    # If there is no base type, then it is a forward declaration
    basetype: Optional[str]


@dataclass
class VulkanHandle(VulkanType):
    """The meta data defines a Vulkan Handle"""
    # Melih TODO: Vulkan Handles have object type in the XML that might be required in the future
    dispatchable: bool


@dataclass
class VulkanHandleAlias(VulkanType):
    """The meta data defines a Vulkan Handle alias"""
    aliased_typename: str


@dataclass
class VulkanStructMember:
    """The meta data defines a Vulkan Handle"""
    variable_type: str
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
    array_size_reference: Optional[str]

    # Is this field has to be set and/or not-null
    optional: Optional[bool]

    # Melih TODO: Currently put the pointer and const info directly
    # into the type name. If we need to extract it later, we extract from the
    # typename with helper functions


@dataclass
class VulkanStruct(VulkanType):
    """The meta data defines a Vulkan Struct"""

    # These give us both what are the members are and their order
    member_order: List[str] = field(default_factory=list)
    members: Dict[str, VulkanStructMember] = field(default_factory=dict)


@dataclass
class VulkanStructAlias(VulkanType):
    """The meta data defines a Vulkan Struct alias"""
    aliased_typename: str


@dataclass
class VulkanFunctionArgument:
    """The meta data defines a Function argument"""
    argument_type: str
    argument_name: str


@dataclass
class VulkanFunctionPtr(VulkanType):
    """The meta data defines a Function Pointer"""
    return_type: str

    # These give us both what are the arguments are and their order
    argument_order: List[str] = field(default_factory=list)
    arguments: Dict[str, VulkanFunctionArgument] = field(default_factory=dict)


@dataclass
class VulkanEnumField:
    """The meta data defines a value in an Enum"""
    name: str

    # Python3 int's are limitless
    value: int

    # Enum fields are sometimes represented hexadecimal especially in bitmasks.
    # It can be pretty irregular in the spec. Therefore, we can explicitly
    # state the representation when necessary.
    representation: str


@dataclass
class VulkanEnum(VulkanType):
    """The meta data defines an Enum"""
    field_order: List[str]
    fields: Dict[str, VulkanEnumField]

    # Enum fields can be aliases of other values especially around
    # the extensions. Moreover, these aliases can be chained.
    # In the spec and header they are pretty irregular in terms of where
    # they have been defined.
    # Therefore we keep them separate so that the field order does not
    # get mixed.
    aliases: Dict[str, str]

    # Is this enum have bitmask fields
    bitmask: bool

    # Is this enum is 64 bits
    bit64: bool


@dataclass
class VulkanEnumAlias(VulkanType):
    """The meta data defines an alias to an Enum"""
    aliased_typename: str


@dataclass
class VulkanBitmask(VulkanType):
    """The meta data defines a bitmask type"""
    # Type of the field which is an enum
    # if this field is empty, it means that it is reserved for the future
    field_type: Optional[str]

    # Base type of the field e.g. VkFlags, VkFlags64
    field_basetype: str


@dataclass
class VulkanBitmaskAlias(VulkanType):
    """The meta data defines an alias to a bitmask type"""
    aliased_typename: str


@dataclass
class VulkanDefine:
    variable_name: str
    value: str


@dataclass
class VulkanCommandParam:
    """The metadata defines a Vulkan Command's parameter"""
    parameter_type: str
    parameter_name: str

    # Is this field has to be set and/or not-null
    optional: Optional[bool]

    # Is this parameter must be externally synced
    externally_synced: Optional[bool]

    # If the parameter is an array, it's size is defined by another
    # parameter of the command. This is the name of the referring parameter.
    array_size_reference: Optional[str]


@dataclass
class VulkanCommand:
    """The metadata defines a Vulkan Command"""
    name: str
    return_type: str

    # Can this command be called inside or outside
    # of a renderpass, or both
    renderpass_allowance: Optional[str]

    success_codes: Optional[List[str]]
    error_codes: Optional[List[str]]

    # Which queues this command can be used
    queues: Optional[List[str]]

    # Which command
    command_buffer_levels: Optional[List[str]]

    # These give us both what are the parameters are and their order
    parameter_order: List[str] = field(default_factory=list)
    parameters: Dict[str, VulkanCommandParam] = field(default_factory=dict)


@dataclass
class VulkanCommandAlias:
    """The metadata defines a Vulkan Command alias"""
    command_name: str
    aliased_command_name: str


@dataclass
class AllVulkanTypes:
    """
    This class holds the information of parsed types from Vulkan XML
    This class should have all the information needed to generate code for types
    """
    # This class holds every Vulkan Type as [typename -> type]

    # Melih TODO: We probably need the map in two ways while generating code
    # both type -> alias and alias -> type
    # For now, lets store as the other types but when we do code generation,
    # We may have an extra step to convert the map to other direction.

    defines: OrderedDict[str, VulkanDefine] = field(default_factory=OrderedDict)

    basetypes: OrderedDict[str, VulkanBaseType] = field(default_factory=OrderedDict)

    handles: Dict[str, VulkanHandle] = field(default_factory=dict)
    handle_aliases: Dict[str, VulkanHandleAlias] = field(default_factory=dict)

    structs: Dict[str, VulkanStruct] = field(default_factory=dict)
    struct_aliases: Dict[str, VulkanStructAlias] = field(default_factory=dict)

    funcpointers: Dict[str, VulkanFunctionPtr] = field(default_factory=dict)

    bitmasks: Dict[str, VulkanBitmask] = field(default_factory=dict)
    bitmask_aliases: Dict[str, VulkanBitmaskAlias] = field(default_factory=dict)

    enums: Dict[str, VulkanEnum] = field(default_factory=dict)
    enum_aliases: Dict[str, VulkanEnumAlias] = field(default_factory=dict)


@dataclass
class AllVulkanCommands:
    """
    This class holds the information of parsed commands from Vulkan XML
    This class should have all the information needed to generate code for commands
    """
    commands: Dict[str, VulkanCommand] = field(default_factory=dict)
    command_aliases: Dict[str, VulkanCommandAlias] = field(default_factory=dict)


@dataclass
class SpirvExtension:
    """
    The metadata that defines a Spirv Extension
    """
    name: str

    # Set if this extension part of a Vulkan version
    version: Optional[str]

    # Vulkan extension enabled by this Spirv extension
    extension: str


@dataclass
class SpirvCapability:
    """
    The metadata that defines a Spirv Capability

    Attribures:
        feature     This capability enables 'Vk*Features::feature'
        property    This capability enables 'vk*Properties::property_group::property'
    """
    name: str

    # Set if this extension part of a Vulkan version
    version: Optional[str]

    # Which Vulkan feature this capabiliy enables
    feature: Optional[str]

    # Which Vulkan property this capabiliy enables
    property: Optional[str]

    # Vulkan extension enabled by this Spirv extension
    extension: str


@dataclass
class SpirvMetadata:
    """
    This class holds the information of Spirv features from Vulkan XML
    This class should have all the information needed to generate code related to Spirv
    """
    extensions: Dict[str, SpirvExtension] = field(default_factory=dict)
    capabilities: Dict[str, SpirvCapability] = field(default_factory=dict)


@dataclass
class VulkanMetadata:
    """
    This class holds the information parsed from Vulkan XML
    This class should have all the information needed to generate code
    """
    types: AllVulkanTypes
    commands: AllVulkanCommands
    spirv_metadata: SpirvMetadata
