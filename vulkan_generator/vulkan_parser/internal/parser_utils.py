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

"""This package contains common functionalities used parsing different Vulkan types"""


from dataclasses import dataclass
from typing import Dict
from typing import List
from typing import Optional

import os
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types


################################
#                              #
#           XML Utils          #
#                              #
################################


def get_text_from_tag_in_children(elem: ET.Element, tag: str, recursive: bool = False) -> str:
    """Gets the text of the element with the given tag in element

    Args:
        recursive(bool):
            If True:
                The function will search the entire subtree recursively including the root node
            If False:
                The function will search only the first level children.
    """
    value = try_get_text_from_tag_in_children(elem, tag, recursive)

    if value is None:
        # This should not happen
        raise SyntaxError(f"No {tag} tag found in {ET.tostring(elem, 'utf-8')}")

    return value


def try_get_text_from_tag_in_children(elem: ET.Element, tag: str, recursive: bool = False) -> Optional[str]:
    """Tries to Gets the text of the element with the given tag
    and returns None if the tag does not exits

    Args:
        recursive(bool):
            If True:
                The function will search the entire subtree recursively including the root node
            If False:
                The function will search only the first level children.
    """

    search_base = elem.iter() if recursive else elem

    for child in search_base:
        if tag == child.tag:
            return child.text

    return None


def get_tail_from_tag_in_children(elem: ET.Element, tag: str, recursive: bool = False) -> str:
    """Gets the tail of the element with the given tag recursively

    Args:
        recursive(bool):
            If True:
                The function will search the entire subtree recursively including the root node
            If False:
                The function will search only the first level children.
    """
    value = try_get_tail_from_tag_in_children(elem, tag, recursive)

    if value is None:
        # This should not happen
        raise SyntaxError(f"No tail found in {ET.tostring(elem, 'utf-8')}")

    return value


def try_get_tail_from_tag_in_children(elem: ET.Element, tag: str, recursive: bool = False) -> Optional[str]:
    """Tries to Gets the text of the element with the given tag
    and returns None if the tag does not exits

    Args:
        recursive(bool):
            If True:
                The function will search the entire subtree recursively including the root node
            If False:
                The function will search only the first level children.
    """
    search_base = elem.iter() if recursive else elem

    for child in search_base:
        if tag == child.tag:
            return child.tail

    return None


def try_get_attribute_as_list(elem: ET.Element, attrib: str) -> Optional[List[str]]:
    """Tries to get an attribute from XML and returns None if the attribute does not exists"""
    if attrib not in elem.attrib:
        return None

    attrib_str = elem.attrib[attrib]
    if not attrib_str:
        return None

    return attrib_str.split(",")


def clean_type_string(string: str) -> str:
    """
    Cleans the string from whitespace and ',' and ');'

    if the type string has a struct modifier, it ensures that they are separated
    """
    return string.replace(os.linesep, "").replace(" ", "").replace(",", "").replace(");", "").replace(
        "conststruct", "const struct")


def clean_size_reference(string: str) -> str:
    # Remove mathematical signs and numbers
    # Keep _ because it's needed for Vulkan types e.g. VK_UUID_SIZE
    return "".join(filter(lambda s: s.isalpha() or s == "_", string))


def get_plain_typename(string: str) -> str:
    """
    It removes type modifiers from a type e.g:

    const void* -> void
    const VkCreateDeviceInfo* -> VkCreateDeviceInfo
    """
    return clean_type_string(string.replace("const", "").replace("*", "").replace("struct", ""))

################################
#                              #
#          Enum Utils          #
#                              #
################################


@dataclass
class EnumFieldRepresentation:
    value: int
    representation: str


def get_enum_field_from_value(value_str: str) -> EnumFieldRepresentation:
    """
    Returns an enum field representation from given value
    """
    representation = value_str
    value = int(representation, 0)

    return EnumFieldRepresentation(value=value, representation=representation)


def get_enum_field_from_bitpos(bitpos_str: str, bit64: bool) -> EnumFieldRepresentation:
    """
    Returns an enum field representation from bit position
    """
    bitpos = int(bitpos_str)
    value = 1 << bitpos
    representation = f"0x{value:08x}"
    if bit64:
        representation = f"{representation}ULL"

    return EnumFieldRepresentation(value=value, representation=representation)


def get_enum_field_from_offset(
        extnumber_str: Optional[str],
        offset_str: str,
        sign_str: Optional[str]) -> EnumFieldRepresentation:
    """
    Returns an enum field representation from an offset based on extension number and sign
    """
    # Representation format for extension enums:
    # 1000EEEOOO
    # e.g. Extension Number: 123, offset:4 => 1000123004

    if not extnumber_str:
        # Sometimes extension enums does not have extnumber
        # In that case extension number in the enum fied should be 0
        # but because all extension numbers are off by 1 in the enum
        # we set it to 1
        extnumber_str = "1"

    sign = sign_str or ""
    extnumber = int(extnumber_str)
    offset = int(offset_str)

    representation = f"{sign}1000{(extnumber- 1):03}{offset:03}"
    value = int(representation)

    return EnumFieldRepresentation(value=value, representation=representation)


################################
#                              #
#       Requirement Utils      #
#                              #
################################


@dataclass
class FieldAliasInfo:
    alias: str


def parse_enum_alias_extension(element: ET.Element) -> FieldAliasInfo:
    return FieldAliasInfo(element.attrib["alias"])


@dataclass
class FieldValueInfo:
    extnumber: Optional[str]
    offset: Optional[str]
    sign: Optional[str]
    bitpos: Optional[str]
    value: Optional[str]


def parse_enum_extension(element: ET.Element) -> FieldValueInfo:
    return FieldValueInfo(
        extnumber=element.get("extnumber"),
        offset=element.get("offset"),
        sign=element.get("dir"),
        bitpos=element.get("bitpos"),
        value=element.get("value"),
    )


def parse_requirement(require_element: ET.Element) -> internal_types.VulkanExtensionRequirement:
    """Parses requirement for Vulkan Core versions and extensions"""
    features: Dict[str, internal_types.VulkanFeature] = {}

    version = require_element.get("feature")
    extension = require_element.get("extension")

    for required_feature_element in require_element:
        if required_feature_element.tag == "comment":
            continue

        feature_name = required_feature_element.attrib["name"]
        feature_type = required_feature_element.tag

        if required_feature_element.tag != "enum":
            # If the tag is not enum, then it's just a required feature
            features[feature_name] = internal_types.VulkanFeature(name=feature_name, feature_type=feature_type)
            continue

        basetype = required_feature_element.get("extends")
        alias = required_feature_element.get("alias")

        if not basetype:
            # If there is no basetype, then it's a new Vulkan define which is under enum tag
            # They either define a new define with value field or alias an existing define
            # with alias tag(which practically is another define)
            # If they don't define a new value then it's just a required annotation

            value = required_feature_element.get("value")
            if not value:
                value = required_feature_element.get("alias")

            if value:
                features[feature_name] = internal_types.VulkanFeatureExtensionDefine(
                    name=feature_name,
                    feature_type=feature_type,
                    value=value,
                )
            else:
                features[feature_name] = internal_types.VulkanFeature(name=feature_name, feature_type=feature_type)
            continue

        alias = required_feature_element.get("alias")
        if alias:
            # if there is alias then it's a field alias
            field_alias_info = parse_enum_alias_extension(required_feature_element)
            features[feature_name] = internal_types.VulkanFeatureExtensionEnumAlias(
                name=feature_name,
                feature_type=feature_type,
                basetype=basetype,
                alias=field_alias_info.alias
            )
            continue

        # if the feature extension is not define or alias, then it's an enum field
        field_info = parse_enum_extension(required_feature_element)
        features[feature_name] = internal_types.VulkanFeatureExtensionEnum(
            name=feature_name,
            feature_type=feature_type,
            basetype=basetype,
            extnumber=field_info.extnumber,
            offset=field_info.offset,
            sign=field_info.sign,
            bitpos=field_info.bitpos,
            value=field_info.value
        )

    return internal_types.VulkanExtensionRequirement(
        required_version=version,
        required_extension=extension,
        features=features,
    )
