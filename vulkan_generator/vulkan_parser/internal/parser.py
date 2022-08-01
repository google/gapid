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
from typing import Dict
from typing import Optional
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types
from vulkan_generator.vulkan_parser.internal import parser_utils
from vulkan_generator.vulkan_parser.internal import version_features_parser
from vulkan_generator.vulkan_parser.internal import extensions_parser
from vulkan_generator.vulkan_parser.internal import formats_parser
from vulkan_generator.vulkan_parser.internal import type_parser
from vulkan_generator.vulkan_parser.internal import enums_parser
from vulkan_generator.vulkan_parser.internal import commands_parser
from vulkan_generator.vulkan_parser.internal import spirv_capabilities_parser
from vulkan_generator.vulkan_parser.internal import spirv_extensions_parser
from vulkan_generator.vulkan_parser.internal import platforms_parser


def process_enums(vulkan_types: internal_types.AllVulkanTypes, enum_element: ET.Element) -> None:
    """Process the parsing of Vulkan enums"""
    # Enums are not under the types tag in the XML.
    # Therefore, they have to be handled separately.
    vulkan_enums = enums_parser.parse(enum_element)

    if not vulkan_enums:
        raise SyntaxError(f"Enum could not be parsed {ET.tostring(enum_element, 'utf-8')}")

    if isinstance(vulkan_enums, internal_types.VulkanEnum):
        vulkan_types.enums[vulkan_enums.typename] = vulkan_enums
        return

    # Some Vulkan defines are under enums tag. Therefore we need to parse them here.
    if isinstance(vulkan_enums, dict):
        for define in vulkan_enums.values():
            vulkan_types.defines[define.key] = define
        return

    raise SyntaxError(f"Unknown define or enum {vulkan_enums}")


def process_core_versions(versions: Dict[str, internal_types.VulkanCoreVersion], feature_element: ET.Element) -> None:
    """Processes the parsing of Vulkan core versions"""
    features = version_features_parser.parse(feature_element)

    if not features:
        raise SyntaxError(f"Vulkan version could not be parsed {ET.tostring(feature_element, 'utf-8')!r}")

    versions[features.name] = features


def get_enum_field_for_extension(
        extension: internal_types.VulkanFeatureExtensionEnum,
        bit64: bool) -> Optional[parser_utils.EnumFieldRepresentation]:
    """Gets the enum value based on how its defined on XML"""
    if extension.value:
        return parser_utils.get_enum_field_from_value(extension.value)

    if extension.bitpos:
        return parser_utils.get_enum_field_from_bitpos(extension.bitpos, bit64)

    if extension.offset:
        return parser_utils.get_enum_field_from_offset(
            extnumber_str=extension.extnumber, offset_str=extension.offset, sign_str=extension.sign)

    return None


def append_extended_feature(
        feature: internal_types.VulkanFeature,
        vulkan_types: internal_types.AllVulkanTypes) -> None:
    """Appends an enum extension to their corresponding enum"""
    if isinstance(feature, internal_types.VulkanFeatureExtensionEnum):
        field = get_enum_field_for_extension(feature, vulkan_types.enums[feature.basetype].bit64)

        if not field:
            raise SyntaxError(f"Enum field for {feature.basetype}:{feature.name} could not be generated")

        vulkan_types.enums[feature.basetype].fields[feature.name] = internal_types.VulkanEnumField(
            name=feature.name,
            value=field.value,
            representation=field.representation,
            parent=feature.basetype,
            extension=True,
        )
    elif isinstance(feature, internal_types.VulkanFeatureExtensionEnumAlias):
        vulkan_types.enums[feature.basetype].aliases[feature.name] = internal_types.VulkanEnumFieldAlias(
            typename=feature.name,
            aliased_typename=feature.alias,
            parent=feature.basetype,
            extension=True,
        )
    elif isinstance(feature, internal_types.VulkanFeatureExtensionDefine):
        vulkan_types.defines[feature.name] = internal_types.VulkanDefine(
            key=feature.name,
            variable_name=feature.name,
            value=feature.value,
            extension=True,
        )


def append_extended_enum_and_bitfield_fields(
        core_versions: Dict[str, internal_types.VulkanCoreVersion],
        extensions: Dict[str, internal_types.VulkanExtension],
        vulkan_types: internal_types.AllVulkanTypes) -> None:
    """
    Appends the enum/bit fields that defined by a core version or a Vulkan extension
    to their corresponding enums and bitfields
    """

    for version in core_versions.values():
        for feature in version.features.values():
            if feature.feature_type != "enum":
                continue

            # Append enum field or define
            append_extended_feature(feature, vulkan_types)

    for extension in extensions.values():
        for requirement in extension.requirements:
            for feature in requirement.features.values():
                if feature.feature_type != "enum":
                    continue

                # Append enum field or define
                append_extended_feature(feature, vulkan_types)


def process_spirv_extensions(spirv_metadata: internal_types.SpirvMetadata, spirv_extensions_elem: ET.Element) -> None:
    """Process all the spirv extensions parsed from the Vulkan XML"""
    for extension_element in spirv_extensions_elem:
        spirv_extension = spirv_extensions_parser.parse(extension_element)

        if not spirv_extension:
            raise SyntaxError(f"Spirv Extension could not found: {ET.tostring(extension_element, 'utf-8')}")

        spirv_metadata.extensions[spirv_extension.name] = spirv_extension


def process_spirv_capabilities(spirv_metadata: internal_types.SpirvMetadata, spirv_cap_elem: ET.Element) -> None:
    """Process all the spirv capabilities parsed from the Vulkan XML"""
    for child in spirv_cap_elem:
        spirv_capability = spirv_capabilities_parser.parse(child)

        if not spirv_capability:
            raise SyntaxError(f"Spirv Capability could not found: {ET.tostring(spirv_cap_elem, 'utf-8')!r}")

        spirv_metadata.capabilities[spirv_capability.name] = spirv_capability


def parse(filename: Path) -> internal_types.VulkanMetadata:
    """ Parse the Vulkan XML to extract every information that is needed for code generation"""
    tree = ET.parse(filename)
    platforms: Dict[str, internal_types.ExternalPlatform] = {}
    all_types = internal_types.AllVulkanTypes()
    all_commands = internal_types.AllVulkanCommands()
    image_formats: Dict[str, internal_types.ImageFormat] = {}
    spirv_metadata = internal_types.SpirvMetadata()
    core_versions: Dict[str, internal_types.VulkanCoreVersion] = {}
    extensions: Dict[str, internal_types.VulkanExtension] = {}

    for child in tree.getroot():
        if child.tag == "platforms":
            platforms = platforms_parser.parse(child)
        if child.tag == "types":
            all_types = type_parser.parse(child)
        elif child.tag == "enums":
            process_enums(all_types, child)
        elif child.tag == "commands":
            all_commands = commands_parser.parse(child)
        elif child.tag == "feature":
            process_core_versions(core_versions, child)
        elif child.tag == "extensions":
            extensions = extensions_parser.parse(child)
        elif child.tag == "formats":
            image_formats = formats_parser.parse(child)
        elif child.tag == ("spirvextensions"):
            process_spirv_extensions(spirv_metadata, child)
        elif child.tag == ("spirvcapabilities"):
            process_spirv_capabilities(spirv_metadata, child)

    # Because extended enum fields are not part of the enum tags in XML, we need to add them later
    append_extended_enum_and_bitfield_fields(core_versions, extensions, all_types)

    return internal_types.VulkanMetadata(
        platforms=platforms,
        types=all_types,
        commands=all_commands,
        core_versions=core_versions,
        extensions=extensions,
        image_formats=image_formats,
        spirv_metadata=spirv_metadata)
