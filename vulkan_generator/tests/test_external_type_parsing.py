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
This module is responsible for testing Vulkan basetypes

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import external_type_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def test_ctype_external_type_with_no_require_field() -> None:
    """""Test the case if the handle name is in an XML tag"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type name="int"/>"""

    external_type = external_type_parser.parse(ET.fromstring(xml))

    assert isinstance(external_type, internal_types.ExternalType)
    assert external_type.typename == "int"
    assert not external_type.source_header
    assert external_type.ctype


def test_ctype_external_type_with_plafrom() -> None:
    """""Test the case if the handle name is in an XML tag"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type requires="vk_platform" name="float"/>"""

    external_type = external_type_parser.parse(ET.fromstring(xml))

    assert isinstance(external_type, internal_types.ExternalType)
    assert external_type.typename == "float"
    assert external_type.source_header == "vk_platform"
    assert external_type.ctype


def test_non_ctype_external_type() -> None:
    """""Test the case if the handle name is in an XML tag"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <type requires="screen/screen.h" name="_screen_context"/>"""

    external_type = external_type_parser.parse(ET.fromstring(xml))

    assert isinstance(external_type, internal_types.ExternalType)
    assert external_type.typename == "_screen_context"
    assert external_type.source_header == "screen/screen.h"
    assert not external_type.ctype
