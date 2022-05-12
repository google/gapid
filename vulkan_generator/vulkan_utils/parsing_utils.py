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
from typing import Optional
import os


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


def clean_type_string(string: str) -> str:
    """Cleans the string from whitespace and ',' and ');'"""
    return string.replace(os.linesep, "").replace(" ", "").replace(",", "").replace(");", "")
