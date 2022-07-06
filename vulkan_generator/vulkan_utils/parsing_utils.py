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

"""This module contains the utility functions that needed elsewhere while parsing Vulkan XML"""

import xml.etree.ElementTree as ET

from dataclasses import dataclass
from typing import List
from typing import Optional
import os

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


def try_get_attribute(elem: ET.Element, attrib: str) -> Optional[str]:
    """Tries to get an attribute from XML and returns None if the attribute does not exists"""
    if attrib not in elem.attrib:
        return None

    return elem.attrib[attrib]


def try_get_attribute_as_list(elem: ET.Element, attrib: str) -> Optional[List[str]]:
    """Tries to get an attribute from XML and returns None if the attribute does not exists"""
    if attrib not in elem.attrib:
        return None

    attrib_str = elem.attrib[attrib]
    if not attrib_str:
        return None

    return attrib_str.split(",")


def clean_type_string(string: str) -> str:
    """Cleans the string from whitespace and ',' and ');'"""
    return string.replace(os.linesep, "").replace(" ", "").replace(",", "").replace(");", "")

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
