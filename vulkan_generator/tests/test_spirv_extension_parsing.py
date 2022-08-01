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

"""
This module is responsible for testing Spirv information

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import spirv_extensions_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def test_spirv_extension() -> None:
    """""Test the case with a spirv extension enables a Vulkan extension"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <spirvextension name="SPV_AMD_gcn_shader">
        <enable extension="VK_AMD_gcn_shader"/>
    </spirvextension>"""

    spirv_extension = spirv_extensions_parser.parse(ET.fromstring(xml))

    assert isinstance(spirv_extension, internal_types.SpirvExtension)
    assert spirv_extension.name == "SPV_AMD_gcn_shader"
    assert spirv_extension.vulkan_extension == "VK_AMD_gcn_shader"


def test_spirv_extension_with_version() -> None:
    """""Test the case with a spirv extension with version"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <spirvextension name="SPV_KHR_variable_pointers">
        <enable version="VK_VERSION_1_1"/>
        <enable extension="VK_KHR_variable_pointers"/>
    </spirvextension>"""

    spirv_extension = spirv_extensions_parser.parse(ET.fromstring(xml))

    assert isinstance(spirv_extension, internal_types.SpirvExtension)
    assert spirv_extension.version == "VK_VERSION_1_1"
