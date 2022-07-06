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

""" This module is responsible for parsing Vulkan extensions and aliases of them"""

from typing import Dict
from typing import List
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import parser_utils
from vulkan_generator.vulkan_utils import parsing_utils


def parse(extensions_element: ET.Element) -> Dict[str, types.VulkanExtension]:
    """Returns Vulkan extensions and/or aliases from the XML element that defines it"""
    extensions: Dict[str, types.VulkanExtension] = {}

    for extension_element in extensions_element:
        if extension_element.attrib["supported"] == "disabled":
            # Apps should not use disabled extensions.
            # Therefore log them for now and if we need we can add them in the future
            print(f"Disabled Extension : {extension_element.attrib['name']}")
            continue

        if extension_element.attrib["supported"] != "vulkan":
            raise SyntaxError(f"Unexpected Support: {ET.tostring(extension_element, 'utf-8')!r}")

        name = extension_element.attrib["name"]
        number = int(extension_element.attrib["number"])

        # if this extension promoted to a core version which version it is
        promoted = parsing_utils.try_get_attribute(extension_element, "promotedto")

        # if another extension deprecated this extension
        deprecated = parsing_utils.try_get_attribute(extension_element, "deprecatedby")
        if deprecated == "":
            deprecated = None

        # Is this a device or instance extension
        extension_type = extension_element.attrib["type"]

        # Which extensions this extension requires
        required_extensions = parsing_utils.try_get_attribute_as_list(extension_element, "requires")

        # If this extension is limited to a platform, which platform it is
        platform = parsing_utils.try_get_attribute(extension_element, "platform")

        requirements: List[types.VulkanExtensionRequirement] = []
        for requirement_element in extension_element:
            requirements.append(parser_utils.parse_requirement(requirement_element))

        extensions[name] = types.VulkanExtension(
            name=name,
            number=number,
            promoted_version=promoted,
            deprecating_extension=deprecated,
            extension_type=extension_type,
            required_extensions=required_extensions,
            platform=platform,
            requirements=requirements,
        )

    return extensions
