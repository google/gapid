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
This module is responsible for testing Vulkan includes

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import include_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def test_vulkan_include_with_directive() -> None:
    """""Test the case if the handle name is in an XML tag"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type name="vk_platform" category="include">#include "vk_platform.h"</type>"""

    include = include_parser.parse(ET.fromstring(xml))

    assert isinstance(include, internal_types.ExternalInclude)
    assert include.header == "vk_platform"
    assert include.directive == '#include "vk_platform.h"'


def test_vulkan_include_without_directive() -> None:
    """""Test the case if the handle name is in an XML tag"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type category="include" name="ggp_c/vulkan_types.h"/>"""

    include = include_parser.parse(ET.fromstring(xml))

    assert isinstance(include, internal_types.ExternalInclude)
    assert include.header == "ggp_c/vulkan_types.h"
