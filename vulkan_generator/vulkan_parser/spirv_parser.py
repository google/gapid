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

""" This module is responsible for parsing Spirv elements and their aliases"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types
from vulkan_generator.vulkan_parser import spirv_extensions_parser
from vulkan_generator.vulkan_parser import spirv_capabilities_parser


def process_spirv_extensions(spirv_metadata: types.SpirvMetadata, spirv_extensions_element: ET.Element) -> None:
    """Process all the spirv extensions parsed from the Vulkan XML"""
    for extension_element in spirv_extensions_element:
        spirv_extension = spirv_extensions_parser.parse(extension_element)

        if not spirv_extension:
            raise SyntaxError(f"Spirv Extension could not found: {ET.tostring(extension_element, 'utf-8')}")

        spirv_metadata.extensions[spirv_extension.name] = spirv_extension


def process_spirv_capabilities(spirv_metadata: types.SpirvMetadata, spirv_capabilities_element: ET.Element) -> None:
    """Process all the spirv capabilities parsed from the Vulkan XML"""
    for child in spirv_capabilities_element:
        spirv_capability = spirv_capabilities_parser.parse(child)

        if not spirv_capability:
            raise SyntaxError(f"Spirv Capability could not found: {ET.tostring(spirv_capabilities_element, 'utf-8')!r}")

        spirv_metadata.capabilities[spirv_capability.name] = spirv_capability


def parse(spirv_element: ET.Element) -> types.SpirvMetadata:
    """Returns Spirv metadata parsed from the XML element"""
    spirv_metadata = types.SpirvMetadata()

    if spirv_element.tag == "spirvextensions":
        process_spirv_extensions(spirv_metadata, spirv_element)
    elif spirv_element.tag == "spirvcapabilities":
        process_spirv_capabilities(spirv_metadata, spirv_element)
    else:
        raise SyntaxError(f"Unknown Spirv type: {ET.tostring(spirv_element)!r}")

    return spirv_metadata
