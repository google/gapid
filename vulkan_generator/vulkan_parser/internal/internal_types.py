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
class ExternalPlatform:
    name: str

    # This is the preprocesser guard for the platform
    protect: str


@dataclass
class ExternalInclude:
    """The metadata defines a file included by Vulkan"""
    header: str

    # C directive to include the header if given
    directive: Optional[str]


@dataclass
class ExternalType:
    """The metadata defines a C type"""
    typename: str

    # If any header is required to use this type
    source_header: Optional[str]

    # Is this type a standard C type
    ctype: bool


@dataclass
class VulkanDefine:
    # This is used by Vulkan XML when it's referred from somewhere else
    key: str
    variable_name: str
    value: str

    # Is this define added after Vulkan 1.0 via extension or a later core version
    extension: bool


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
    member_type: str
    member_name: str

    parent: str

    # Some members have this property which states if that particular
    # member has to be valid if they are not null
    no_auto_validity: Optional[bool]

    # Melih TODO: In the future we probably need to change
    # this from str to VulkanEnum.
    # Does member has an expected value e.g. sType
    expected_value: Optional[str]

    # If the member is an array, it's size is defined by another
    # member in the struct or a define if it's a static array
    # If the array is multidimensional, sizes of each dimention will be
    # listed in order.
    size: Optional[List[str]]

    # Some structs uses C bitfield sizes
    c_bitfield_size: Optional[int]

    # Is this field has to be set and/or not-null
    optional: Optional[bool]

    # Melih TODO: Currently put the pointer and const info directly
    # into the type name. If we need to extract it later, we extract from the
    # typename with helper functions


@dataclass
class VulkanStruct(VulkanType):
    """The meta data defines a Vulkan Struct"""
    # if this struct is an extension via pNext, it will be noted here
    base_structs: Optional[List[str]]

    # These give us both what are the members are and their order
    members: Dict[str, VulkanStructMember] = field(default_factory=dict)


@dataclass
class VulkanStructAlias(VulkanType):
    """The meta data defines a Vulkan Struct alias"""
    aliased_typename: str


@dataclass
class VulkanUnionMember:
    """The meta data defines a Vulkan Union Member"""
    member_type: str
    member_name: str

    # Some members have this property which states if that particular
    # member has to be valid if they are not null
    no_auto_validity: Optional[bool]

    # If this member is a decided by an enum field, then its stated here
    selection: Optional[str]

    # If this member is a static array, what is the length
    size: Optional[List[str]]


@dataclass
class VulkanUnion(VulkanType):
    """The meta data defines a Vulkan Union"""
    returned_only: Optional[bool]

    # What types this union can be
    members: Dict[str, VulkanUnionMember] = field(default_factory=dict)


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

    # Which enum this field belongs to
    parent: str

    # Is this define added after Vulkan 1.0 via extension or a later core version
    extension: bool


@dataclass
class VulkanEnumFieldAlias:
    typename: str
    aliased_typename: str

    # Which enum this field belongs to
    parent: str

    # Is this define added after Vulkan 1.0 via extension or a later core version
    extension: bool


@dataclass
class VulkanEnum(VulkanType):
    """The meta data defines an Enum"""
    # Is this enum have bitmask fields
    bitmask: bool

    # Is this enum is 64 bits
    bit64: bool

    # Enum fields can be aliases of other values especially around
    # the extensions. Moreover, these aliases can be chained.
    # In the spec and header they are pretty irregular in terms of where
    # they have been defined.
    # Therefore we keep them separate so that the field order does not
    # get mixed.
    aliases: Dict[str, VulkanEnumFieldAlias] = field(default_factory=dict)
    fields: Dict[str, VulkanEnumField] = field(default_factory=dict)


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
class VulkanCommandParam:
    """The metadata defines a Vulkan Command's parameter"""
    parameter_type: str
    parameter_name: str

    # Is this field has to be set and/or not-null
    optional: Optional[bool]

    # Is this parameter must be externally synced
    externally_synced: Optional[bool]

    # if this is None then the entire field is externally synced
    # Otherwise the specific field represented here is externally synced
    externally_synced_field: Optional[str]

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

    # Includes and defines are actually not Vulkan types but they are under
    # type tag
    includes: Dict[str, ExternalInclude] = field(default_factory=dict)
    defines: Dict[str, VulkanDefine] = field(default_factory=dict)

    external_types: Dict[str, ExternalType] = field(default_factory=dict)
    basetypes: Dict[str, VulkanBaseType] = field(default_factory=dict)

    handles: Dict[str, VulkanHandle] = field(default_factory=dict)
    handle_aliases: Dict[str, VulkanHandleAlias] = field(default_factory=dict)

    structs: Dict[str, VulkanStruct] = field(default_factory=dict)
    struct_aliases: Dict[str, VulkanStructAlias] = field(default_factory=dict)

    unions: Dict[str, VulkanUnion] = field(default_factory=dict)

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
class VulkanFeature:
    """The structure to hold a required feature for a Vulkan Core Version"""
    name: str
    feature_type: str


@dataclass
class VulkanFeatureExtension(VulkanFeature):
    """The base structure to extend a Vulkan feature"""


@dataclass
class VulkanFeatureExtensionEnum(VulkanFeatureExtension):
    """
    The structure to hold field extension to an existing enum

    When a vulkan enum is a required feature, unlike other types, the extending fields are defined
    on the feature rather than the original enum
    """
    # Which type this enum is extending
    basetype: str

    # Extension number is used when calculating enum field value
    extnumber: Optional[str]

    # Offset is used when calculating enum field value
    offset: Optional[str]

    # Some enum fields are negative
    sign: Optional[str]

    # Bitpos is used when calcualting bitfield value
    bitpos: Optional[str]

    # Some features explicitly gives the new value
    value: Optional[str]


@dataclass
class VulkanFeatureExtensionEnumAlias(VulkanFeatureExtension):
    """
    The structure to hold aliased field extension to an existing enum

    When a vulkan enum is a required feature, unlike other types, the extending fields are defined
    on the feature rather than the original enum
    """
    # Which type this enum is extending
    basetype: str

    # Extending enum field is an alias to another enum fields
    alias: str


@dataclass
class VulkanFeatureExtensionDefine(VulkanFeatureExtension):
    """
    The structure to hold a new define added by an extension

    When a vulkan defined is a required feature, unlike other types, the extending fields are defined
    on the feature rather than the original enum
    """
    value: str


@dataclass
class VulkanExtensionRequirement:
    """The structure to holds a subgroup features required or added by a Vulkan extension"""
    # Sometimes a group of features depends on a core version
    required_version: Optional[str]
    # Sometimes a group of features depends on another extension
    required_extension: Optional[str]
    features: Dict[str, VulkanFeature] = field(default_factory=dict)


@dataclass
class VulkanCoreVersion:
    """The structure to hold all the features required or added by a Vulkan core version"""
    name: str
    number: str
    features: Dict[str, VulkanFeature] = field(default_factory=dict)


@dataclass
class VulkanExtension:
    """The structure to hold all the features required or added by a Vulkan extension"""
    name: str
    number: int

    # This extension is core in the promoted version or promoted to another extension
    # Usually from vendor to KHR
    promotedto: Optional[str]

    # Some extensions are disabled but still referenced from other Vulkan types
    disabled: bool

    # This extension is deprecated by another extension or core version
    deprecatedby: Optional[str]

    # Whether this extension a device or instance extension
    extension_type: Optional[str]

    # Sometimes an extension requires another extension
    required_extensions: Optional[List[str]]

    # Is this extension limited to a particular platform
    platform: Optional[str]

    requirements: List[VulkanExtensionRequirement] = field(default_factory=list)


@dataclass
class ImageFormatComponent:
    """
    The metadata that defines a component of a Vulkan image format
    """
    name: str
    numeric_format: str

    # bitsize for non-compressed images
    bits: Optional[int]

    # Is this image format is compressed
    compressed: bool


@dataclass
class ImageFormatPlane:
    """
    The metadata that defines a plane of a multi-planar Vulkan image format
    """
    index: int
    width_divisor: int
    height_divisor: int
    compatible: str


@dataclass
class ImageFormat:
    """
    The metadata that defines a Vulkan image format
    """
    name: str
    format_class: str
    block_size: int
    texels_per_block: int

    # Pack size for the packed formats
    packed: Optional[int]

    # if this format has a corresponding Spir-V format
    spirv_format: Optional[str]

    components: Dict[str, ImageFormatComponent] = field(default_factory=dict)

    # Plane information for multi-planar images
    planes: List[ImageFormatPlane] = field(default_factory=list)


@dataclass
class SpirvExtension:
    """
    The metadata that defines a Spirv Extension
    """
    name: str

    # Set if this extension part of a Vulkan version
    version: Optional[str]

    # Vulkan extension enabled by this Spirv extension
    vulkan_extension: str


@dataclass
class SpirvCapabilityFeature:
    struct: str
    feature: str


@dataclass
class SpirvCapabilityProperty:
    struct: str
    group: str
    value: str


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
    feature: Optional[SpirvCapabilityFeature]

    # Which Vulkan property this capabiliy enables
    property: Optional[SpirvCapabilityProperty]

    # Vulkan extension enabled by this Spirv extension
    vulkan_extension: Optional[str]


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
    platforms: Dict[str, ExternalPlatform]
    types: AllVulkanTypes
    commands: AllVulkanCommands
    core_versions: Dict[str, VulkanCoreVersion]
    extensions: Dict[str, VulkanExtension]
    image_formats: Dict[str, ImageFormat]
    spirv_metadata: SpirvMetadata
