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

# This is required for self referencing
from __future__ import annotations

from dataclasses import dataclass
from dataclasses import field
from enum import Enum

from typing import Dict
from typing import Union
from typing import List
from typing import Optional


@dataclass
class ExternalPlatform:
    name: str
    protect: str


@dataclass
class ExternalInclude:
    header: str

    # C directive to include the header, if given
    directive: Optional[str]


@dataclass
class VulkanDefine:
    # This is the key used by other Vulkan types to refer a define
    key: str
    name: str
    value: str

    # Is this define added after Vulkan 1.0 via extension or a later core version
    extension: bool


@dataclass
class VulkanType:
    """Base class for a Vulkan Type. All the other Types should inherit from this class"""
    typename: str


@dataclass
class VulkanTypeAlias(VulkanType):
    """Base class for a Vulkan Alias. All the other aliases should inherit from this class"""
    aliased_type: VulkanType


@dataclass
class ExternalType(VulkanType):
    """
    The metadata defines a C type

    This is actually not a Vulkan type but because its referenced from Vulkan types frequently
    it's easier to pass it as a Vulkan type
    """
    # If any header is required to use this type
    source_header: Optional[str]

    # Is this type a standard C type
    ctype: bool


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
class VulkanHandleAlias(VulkanTypeAlias):
    """The meta data defines a Vulkan Handle alias"""


@dataclass
class VulkanEnumField:
    """The meta data defines a value in an enum field"""
    name: str

    # Python3 int's are limitless
    value: int

    # Enum fields are sometimes represented hexadecimal especially in bitmasks.
    # It can be pretty irregular in the spec. Therefore, we can explicitly
    # state the representation when necessary.
    representation: str

    parent: VulkanEnum

    # Print parent's typename only to prevent circular dependency
    def to_str(self) -> str:
        return f"""
        {self.name}
            Value: {self.value}
            Representation: {self.representation}
            Parent: {self.parent.typename}
        """

    def __repr__(self) -> str:
        return self.to_str()

    def __str__(self) -> str:
        return self.to_str()


@dataclass
class VulkanEnumFieldAlias:
    """The meta data defines a value in an alias to an enum field"""
    name: str
    aliased_field: VulkanEnumField

    # Is this define added after Vulkan 1.0 via extension or a later core version
    extension: bool

    parent: VulkanEnum

    # Print parent's typename only to prevent circular dependency
    def to_str(self) -> str:
        return f"""
        {self.name}
            Alias: {self.aliased_field}
            Extension: {self.extension}
            Parent: {self.parent.typename}
        """

    def __repr__(self) -> str:
        return self.to_str()

    def __str__(self) -> str:
        return self.to_str()


@dataclass
class VulkanEnum(VulkanType):
    """The meta data defines an Enum"""
    # Is this enum have bitmask fields
    bitmask: bool

    # Is this enum is 64 bits
    bit64: bool

    fields: Dict[str, VulkanEnumField] = field(default_factory=dict)

    # Enum fields can be aliases of other values especially around
    # the extensions. Moreover, these aliases can be chained.
    # In the spec and header they are pretty irregular in terms of where
    # they have been defined.
    # Therefore we keep them separate so that the field order does not
    # get mixed.
    aliases: Dict[str, VulkanEnumFieldAlias] = field(default_factory=dict)


@dataclass
class VulkanEnumAlias(VulkanTypeAlias):
    """The meta data defines an alias to an Enum"""


@dataclass
class VulkanBitmask(VulkanType):
    """The meta data defines a bitmask type"""
    # Type of the field which is an enum
    # if this field is empty, it means that it is reserved for the future
    field_type: Optional[VulkanEnum]

    # Base type of the field e.g. VkFlags, VkFlags64
    field_basetype: VulkanBaseType


@dataclass
class VulkanBitmaskAlias(VulkanTypeAlias):
    """The meta data defines an alias to a bitmask type"""


@dataclass
class VulkanFunctionArgument:
    """The meta data defines a Function argument"""
    type: VulkanType
    name: str

    # Full typename of the function argument. This would include modifiers e.g. const and *
    full_typename: str


@dataclass
class VulkanFunctionPtr(VulkanType):
    """The meta data defines a Function Pointer"""
    return_type: VulkanType

    # These give us both what are the arguments are and their order
    arguments: Dict[str, VulkanFunctionArgument] = field(default_factory=dict)


@dataclass
class VulkanStructMember:
    """The meta data defines a Vulkan Handle"""
    type: VulkanType
    name: str

    parent: VulkanStruct

    # Full typename of the member. This would include modifiers e.g. const and *
    full_typename: str

    # Full field name of the member. This would include modifiers e.g. arrays
    full_member_name: str

    # Some members have this property which states if that particular
    # member has to be valid if they are not null
    no_auto_validity: Optional[bool]

    # Does member has an expected value e.g. sType
    expected_value: Optional[VulkanEnumField]

    # Some member variables are static arrays with a default size
    size: Optional[Union[VulkanDefine, VulkanStructMember, List[int]]]

    # Sometimes size is a calculation based on another member or a define
    # in that case we need the calculation text as well
    size_calculation: Optional[str]

    # If the field is a C sytle bitfield, this will be the size of it
    c_bitfield_size: Optional[int]

    # Is this field has to be set and/or not-null
    optional: Optional[bool]

    # Print only the name of type, parent and size to prevent circular dependency
    def to_str(self) -> str:
        return f"""
        {self.full_typename} {self.name}
            Parent: {self.parent.typename}
            No auto validity: {self.no_auto_validity}
            Expected Value: {self.expected_value}
            Size: {self.size}
            Size Calculation: {self.size_calculation}
            Optional: {self.optional}
        """

    def __repr__(self) -> str:
        return self.to_str()

    def __str__(self) -> str:
        return self.to_str()


@dataclass
class VulkanStruct(VulkanType):
    """The meta data defines a Vulkan Struct"""
    # if this struct is an extension via pNext, it will be noted here
    # Because VulkanStruct cannot be referenced from itself, it will be a VulkanType
    base_structs: Dict[str, VulkanStruct] = field(default_factory=dict)

    # if this struct is extendible all the possible pNext values will be in this list
    pnexts: Dict[str, VulkanStruct] = field(default_factory=dict)

    # These give us both what are the members are and their order
    members: Dict[str, VulkanStructMember] = field(default_factory=dict)

    # Trying print struct info causes circular dependency between base struct
    # and pnexts. To prevent this print only the pnexts' typename
    def to_str(self) -> str:
        return f"""
        Base Structures: [{", ".join(self.base_structs)}]
        Possible Pnexts: [{", ".join(self.pnexts)}]
        {self.members}
        """

    def __repr__(self) -> str:
        return self.to_str()

    def __str__(self) -> str:
        return self.to_str()


@dataclass
class VulkanStructAlias(VulkanTypeAlias):
    """The meta data defines a Vulkan Struct alias"""


@dataclass
class VulkanUnionMember:
    """The meta data defines a Vulkan Union Member"""
    type: VulkanType
    name: str

    # Full member name of the member required for declaration.
    # This would include modifiers e.g. arrays
    full_member_name: str

    # Some members have this property which states if that particular
    # member has to be valid if they are not null
    no_auto_validity: Optional[bool]

    # If this member is a decided by an enum field, then its stated here
    # Melih TODO: We may need to convert to a VulkanEnumField
    selection: Optional[str]

    # If this member is a static array, what is the length
    size: Optional[List[int]]


@dataclass
class VulkanUnion(VulkanType):
    """The meta data defines a Vulkan Union"""
    returned_only: Optional[bool]

    # What types this union can be
    members: Dict[str, VulkanUnionMember]


@dataclass
class VulkanCommandParam:
    """The metadata defines a Vulkan Command's parameter"""
    type: VulkanType
    name: str

    # Full typename of the parameter. This would include modifiers e.g. const and *
    full_typename: str

    # Is this field has to be set and/or not-null
    optional: Optional[bool]

    # Is this parameter must be externally synced
    externally_synced: Optional[bool]

    # if this is None then the entire field is externally synced
    # Otherwise the specific field represented here is externally synced
    externally_synced_field: Optional[str]

    # If the parameter is an array, it's size is defined by another
    # parameter of the command.
    size: Optional[Union[VulkanCommandParam, str]]


class RenderpassAllowance(Enum):
    UNKNOWN = 0
    INSIDE = 1
    OUTSIDE = 2
    BOTH = 3


@dataclass
class VulkanCommand:
    """The metadata defines a Vulkan Command"""
    name: str

    # if the command returns void this field will be None
    return_type: Optional[VulkanType]

    # Can this command be called inside or outside
    # of a renderpass, or both
    renderpass_allowance: RenderpassAllowance

    success_codes: Optional[List[VulkanEnumField]]
    error_codes: Optional[List[VulkanEnumField]]

    # Which queues this command can be used
    queues: Optional[List[VulkanEnumField]]

    # Which command
    command_buffer_levels: Optional[List[VulkanEnumField]]

    # These give us both what are the parameters are and their order
    parameters: Dict[str, VulkanCommandParam] = field(default_factory=dict)


@dataclass
class VulkanCommandAlias:
    """The metadata defines a Vulkan Command alias"""
    name: str
    aliased_command: VulkanCommand


@dataclass
class VulkanTypeInfo:
    """
    This class holds the information of parsed types from Vulkan XML
    This class should have all the information needed to generate code for types
    """
    # This class holds every Vulkan Type as [typename -> type]

    # Melih TODO: We probably need the map in two ways while generating code
    # both type -> alias and alias -> type
    # For now, lets store as the other types but when we do code generation,
    # We may have an extra step to convert the map to other direction.

    external_types: Dict[str, ExternalType] = field(default_factory=dict)

    basetypes: Dict[str, VulkanBaseType] = field(default_factory=dict)

    handles: Dict[str, VulkanHandle] = field(default_factory=dict)
    handle_aliases: Dict[str, VulkanHandleAlias] = field(default_factory=dict)

    enums: Dict[str, VulkanEnum] = field(default_factory=dict)
    enum_aliases: Dict[str, VulkanEnumAlias] = field(default_factory=dict)

    bitmasks: Dict[str, VulkanBitmask] = field(default_factory=dict)
    bitmask_aliases: Dict[str, VulkanBitmaskAlias] = field(default_factory=dict)

    unions: Dict[str, VulkanUnion] = field(default_factory=dict)

    structs: Dict[str, VulkanStruct] = field(default_factory=dict)
    struct_aliases: Dict[str, VulkanStructAlias] = field(default_factory=dict)

    funcpointers: Dict[str, VulkanFunctionPtr] = field(default_factory=dict)

    # This is all the types for generic search
    all_types: Dict[str, VulkanType] = field(default_factory=dict)
    all_aliases: Dict[str, VulkanTypeAlias] = field(default_factory=dict)


@dataclass
class VulkanCommandInfo:
    """
    This class holds the information of parsed commands from Vulkan XML
    This class should have all the information needed to generate code for commands
    """
    commands: Dict[str, VulkanCommand] = field(default_factory=dict)
    command_aliases: Dict[str, VulkanCommandAlias] = field(default_factory=dict)


@dataclass
class VulkanFeatureSet:
    """The structure to hold all the requirements for a version or extension of Vulkan"""
    includes: Dict[str, ExternalInclude] = field(default_factory=dict)
    defines: Dict[str, VulkanDefine] = field(default_factory=dict)

    types: Dict[str, VulkanType] = field(default_factory=dict)
    type_aliases: Dict[str, VulkanTypeAlias] = field(default_factory=dict)

    commands: Dict[str, VulkanCommand] = field(default_factory=dict)
    command_aliases: Dict[str, VulkanCommandAlias] = field(default_factory=dict)

    enum_fields: Dict[str, VulkanEnumField] = field(default_factory=dict)
    enum_field_aliases: Dict[str, VulkanEnumFieldAlias] = field(default_factory=dict)


@dataclass
class VulkanCoreVersion:
    """The structure to hold all the features required or added by a Vulkan core version"""
    name: str
    number: str

    features: VulkanFeatureSet


@dataclass
class VulkanExtensionRequirement:
    """The structure to holds a subgroup features required or added by a Vulkan extension"""
    # Sometimes a group of features depends on a core version
    required_version: Optional[VulkanCoreVersion]
    # Sometimes a group of features depends on another extension
    required_extension: Optional[VulkanExtension]

    features: VulkanFeatureSet

    # Print only the name of version and extension to prevent circular dependency
    def to_str(self) -> str:
        return f"""
        Required Version: {self.required_version.name if self.required_version else None}
        Required Extension: {self.required_extension.name if self.required_extension else None}
        Features: [{self.features}]
        """

    def __repr__(self) -> str:
        return self.to_str()

    def __str__(self) -> str:
        return self.to_str()


class ExtensionType(Enum):
    UNKNOWN = 0
    INSTANCE = 1
    DEVICE = 2


@dataclass
class VulkanExtension:
    """The structure to hold all the features required or added by a Vulkan extension"""
    name: str
    number: int

    # This extension is core in the promoted version
    promotedto: Optional[Union[VulkanCoreVersion, VulkanExtension]]

    # This extension is deprecated by another extension or core version
    deprecated_by: Optional[Union[VulkanCoreVersion, VulkanExtension]]

    # Whether this extension a device or instance extension
    extension_type: ExtensionType

    # Sometimes an extension requires another extension
    required_extensions: Optional[List[VulkanExtension]]

    # Is this extension limited to a particular platform
    platform: Optional[ExternalPlatform]

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
    version: Optional[VulkanCoreVersion]

    # Vulkan extension enabled by this Spirv extension
    vulkan_extension: VulkanExtension


@dataclass
class SpirvCapabilityFeature:
    struct: VulkanStruct
    feature: VulkanStructMember


@dataclass
class SpirvCapabilityProperty:
    struct: VulkanStruct
    group: VulkanStructMember
    value: Union[VulkanDefine, VulkanEnumField]


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
    version: Optional[VulkanCoreVersion]

    # Which Vulkan feature this capabiliy enables
    feature: Optional[SpirvCapabilityFeature]

    # Which Vulkan property this capabiliy enables
    property: Optional[SpirvCapabilityProperty]

    # Vulkan extension enabled by this Spirv extension
    extension: Optional[VulkanExtension]


@dataclass
class SpirvMetadata:
    """
    This class holds the information of Spirv features from Vulkan XML
    This class should have all the information needed to generate code related to Spirv
    """
    extensions: Dict[str, SpirvExtension] = field(default_factory=dict)
    capabilities: Dict[str, SpirvCapability] = field(default_factory=dict)


@dataclass
class VulkanInfo:
    """
    This class holds the information parsed from Vulkan XML
    This class should have all the information needed to generate code
    """
    platforms: Dict[str, ExternalPlatform]
    includes: Dict[str, ExternalInclude]
    defines: Dict[str, VulkanDefine]
    types: VulkanTypeInfo
    commands: VulkanCommandInfo
    core_versions: Dict[str, VulkanCoreVersion]
    extensions: Dict[str, VulkanExtension]
    image_formats: Dict[str, ImageFormat]
    spirv_metadata: SpirvMetadata
