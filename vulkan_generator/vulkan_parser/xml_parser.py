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

from vulkan_generator.vulkan_parser import parser_utils
from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import version_features_parser
from vulkan_generator.vulkan_parser import extensions_parser
from vulkan_generator.vulkan_parser import formats_parser
from vulkan_generator.vulkan_parser import type_parser
from vulkan_generator.vulkan_parser import enums_parser
from vulkan_generator.vulkan_parser import commands_parser
from vulkan_generator.vulkan_parser import spirv_parser


def process_enums(vulkan_types: types.AllVulkanTypes, enum_element: ET.Element) -> None:
    """Process the parsing of Vulkan enums"""
    # Enums are not under the types tag in the XML.
    # Therefore, they have to be handled separately.
    vulkan_enums = enums_parser.parse(enum_element)

    if not vulkan_enums:
        raise SyntaxError(f"Enum could not be parsed {ET.tostring(enum_element, 'utf-8')}")

    if isinstance(vulkan_enums, types.VulkanEnum):
        vulkan_types.enums[vulkan_enums.typename] = vulkan_enums
        return

    # Some Vulkan defines are under enums tag. Therefore we need to parse them here.
    if isinstance(vulkan_enums, dict):
        for define in vulkan_enums.values():
            vulkan_types.defines[define.variable_name] = define
        return

    raise SyntaxError(f"Unknown define or enum {vulkan_enums}")


def process_core_versions(core_versions: Dict[str, types.VulkanCoreVersion], feature_element: ET.Element) -> None:
    """Processes the parsing of Vulkan core versions"""
    features = version_features_parser.parse(feature_element)

    if not features:
        raise SyntaxError(f"Vulkan version could not be parsed {ET.tostring(feature_element, 'utf-8')!r}")

    core_versions[features.name] = features


def get_enum_field_for_extension(
        extension: types.VulkanFeatureExtensionEnum,
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


def append_field_to_enum_or_bitfield(feature: types.VulkanFeature, vulkan_types: types.AllVulkanTypes) -> None:
    """Appends an enum extension to their corresponding enum"""
    if feature.feature_type != "enum":
        raise SyntaxError(f"Feature is not an enum {feature.name}:{feature.feature_type}")

    if not feature.feature_extension:
        raise SyntaxError(f"Enum feature does not have extension{feature.name}")

    feature_extension = feature.feature_extension
    if not isinstance(feature_extension, types.VulkanFeatureExtensionEnum):
        raise SyntaxError(f"Enum feauture should have enum extension {feature.feature_extension}")

    enum_name = feature_extension.basetype
    field_name = feature.name

    if feature_extension.alias:
        vulkan_types.enums[enum_name].aliases[field_name] = feature_extension.alias
        return

    field = get_enum_field_for_extension(feature_extension, vulkan_types.enums[enum_name].bit64)
    if not field:
        raise SyntaxError(f"Enum field for {enum_name}:{field_name} could not be generated")

    vulkan_types.enums[enum_name].fields[field_name] = types.VulkanEnumField(
        name=field_name,
        value=field.value,
        representation=field.representation,
    )


def append_extended_enum_and_bitfield_fields(
        core_versions: Dict[str, types.VulkanCoreVersion],
        extensions: Dict[str, types.VulkanExtension],
        vulkan_types: types.AllVulkanTypes) -> None:
    """
    Appends the enum/bit fields that defined by a core version or a Vulkan extension
    to their corresponding enums and bitfields
    """

    for version in core_versions.values():
        for feature in version.features.values():
            if feature.feature_type != "enum" or not feature.feature_extension:
                continue

            append_field_to_enum_or_bitfield(feature, vulkan_types)

    for extension in extensions.values():
        for requirement in extension.requirements:
            for feature in requirement.features.values():
                if feature.feature_type != "enum" or not feature.feature_extension:
                    continue

                append_field_to_enum_or_bitfield(feature, vulkan_types)


def parse(filename: Path) -> types.VulkanMetadata:
    """ Parse the Vulkan XML to extract every information that is needed for code generation"""
    tree = ET.parse(filename)
    all_types = types.AllVulkanTypes()
    all_commands = types.AllVulkanCommands()
    format_metadata = types.ImageFormatMetadata()
    spirv_metadata = types.SpirvMetadata()
    core_versions: Dict[str, types.VulkanCoreVersion] = {}
    extensions: Dict[str, types.VulkanExtension] = {}

    for child in tree.getroot():
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
            format_metadata = formats_parser.parse(child)
        elif child.tag.startswith("spirv"):
            spirv_metadata = spirv_parser.parse(child)

    # Because extended enum fields are not part of the enum tags in XML, we need to add them later
    append_extended_enum_and_bitfield_fields(core_versions, extensions, all_types)

    return types.VulkanMetadata(
        types=all_types,
        commands=all_commands,
        core_versions=core_versions,
        extensions=extensions,
        image_format_metadata=format_metadata,
        spirv_metadata=spirv_metadata)
