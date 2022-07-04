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

""" This module is responsible for parsing Spirv extensions"""

from typing import Optional
import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser import types


def parse(spirv_element: ET.Element) -> types.SpirvExtension:
    """Returns a Spirv extension or alias from the XML element that defines it

    A sample Spirv extension:
    <spirvextension name="SPV_KHR_variable_pointers">
        <enable version="VK_VERSION_1_1"/>
        <enable extension="VK_KHR_variable_pointers"/>
    </spirvextension>
    """

    name = spirv_element.attrib["name"]
    version: Optional[str] = None
    extension: Optional[str] = None

    for enable in spirv_element:
        if "version" in enable.attrib:
            version = enable.attrib["version"]
        elif "extension" in enable.attrib:
            extension = enable.attrib["extension"]
        else:
            raise SyntaxError(f"Unknown Spirv capability type: {ET.tostring(spirv_element, 'utf-8')}")

    if not extension:
        raise SyntaxError(
            f"Vulkan extension for the Spirv extension could not found:{ET.tostring(spirv_element, 'utf-8')!r}")

    return types.SpirvExtension(name=name, version=version, extension=extension)
