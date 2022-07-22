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

""" This module is responsible for parsing Vulkan defines

    It parses all defines except some of the defines that are defined under enums as "API Constants"
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types


def parse_define_by_attribute(define_elem: ET.Element) -> internal_types.VulkanDefine:
    """Parses a define that has a name attribute

    If a define has name attribute, it's usually a preprocessor decision logic
    """
    name = define_elem.attrib["name"]
    value = define_elem.text

    if not value:
        raise SyntaxError(f"Vulkan define does not have definition: {ET.tostring(define_elem, 'utf-8')!r}")

    return internal_types.VulkanDefine(
        variable_name=name,
        value=value
    )


def parse_define_by_tag(define_elem: ET.Element) -> internal_types.VulkanDefine:
    """Parses a define that has a name tag

    If a define has a name tag, it's "usually" a macro
    """
    name = None
    value = None

    # The XML is very irregular around the defines. The only thing that we can ensure
    # that it is written in order between the text and tails of XML elements. Therefore,
    # We can just append the value while traversing.
    #
    # e.g.
    # <name>VK_API_VERSION_1_0</name> <type>VK_MAKE_API_VERSION</type>(0, 1, 0, 0)
    #  // Patch version should always be set to 0</type>
    #    <type category="define" requires="VK_MAKE_API_VERSION">// Vulkan 1.1 version number

    macro = False
    for child in define_elem:
        if child.tag == "name":
            name = child.text
            value = f"{value or ''}{child.tail}"

            if child.tail and child.tail[0] == "(":
                # If define is a macro, tail starts with "("
                # e.g
                # type category="define"> // DEPRECATED: This define is deprecated.
                #   VK_MAKE_API_VERSION should be used instead.
                # # define <name>VK_MAKE_VERSION</name>(major, minor, patch) \
                # ((((uint32_t)(major)) &lt;&lt; 22) | (((uint32_t)(minor)) &lt;&lt; 12) | ((uint32_t)(patch)))</type>
                macro = True
        elif child.tag == "type":
            value = f"{value}{child.text}{child.tail}"

    if not name:
        raise SyntaxError(f"Define name could not be parsed: {ET.tostring(define_elem, 'utf-8')}")

    if not value:
        raise SyntaxError(f"Define value could not be parsed: {ET.tostring(define_elem, 'utf-8')}")

    # If define is a macro, macro's argument are part of the text
    if macro:
        macro_value = value.split(")", 1)
        name = f"{name}{macro_value[0]})"
        value = "".join(macro_value[1:])

    # Remove the comments from value
    if "//" in value:
        value = value.split("//")[0]

    # Remove the leading whitespace
    value = value.lstrip(" ")

    return internal_types.VulkanDefine(
        variable_name=name,
        value=value
    )


def parse(define_elem: ET.Element) -> internal_types.VulkanDefine:
    """Returns a Vulkan define from the XML element that defines it.

    A sample Vulkan Define:
    <type category="define" requires="VK_MAKE_API_VERSION">// Vulkan 1.1 version number
    #define <name>VK_API_VERSION_1_1</name> <type>VK_MAKE_API_VERSION</type>
    #   (0, 1, 1, 0)// Patch version should always be set to 0</type>
    """

    if "name" in define_elem.attrib:
        return parse_define_by_attribute(define_elem)

    return parse_define_by_tag(define_elem)
