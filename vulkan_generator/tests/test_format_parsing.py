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
This module is responsible for testing Vulkan image formats

Examples in this files stems from vk.xml that relesed by Khronos.
Anytime the particular xml updated, test should be checked
if they reflect the new XML
"""

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import formats_parser
from vulkan_generator.vulkan_parser.internal import internal_types


def test_vulkan_image_format() -> None:
    """"Test an image format"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <formats>
        <format name="VK_FORMAT_R8_USCALED" class="8-bit" blockSize="1" texelsPerBlock="1">
            <component name="R" bits="8" numericFormat="USCALED"/>
        </format>
    </formats>"""

    image_format_metadata = formats_parser.parse(ET.fromstring(xml))
    assert "VK_FORMAT_R8_USCALED" in image_format_metadata.formats

    image_format = image_format_metadata.formats["VK_FORMAT_R8_USCALED"]

    assert isinstance(image_format, internal_types.ImageFormat)
    assert image_format.name == "VK_FORMAT_R8_USCALED"
    assert image_format.format_class == "8-bit"
    assert image_format.block_size == 1
    assert image_format.texels_per_block == 1


def test_vulkan_image_format_with_spirv_format() -> None:
    """"Test an image format with spirv format"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <formats>
        <format name="VK_FORMAT_R8_UNORM" class="8-bit" blockSize="1" texelsPerBlock="1">
            <component name="R" bits="8" numericFormat="UNORM"/>
            <spirvimageformat name="R8"/>
        </format>
    </formats>"""

    image_format_metadata = formats_parser.parse(ET.fromstring(xml))
    image_format = image_format_metadata.formats["VK_FORMAT_R8_UNORM"]
    assert image_format.spirv_format == "R8"


def test_vulkan_image_format_with_packed() -> None:
    """"Test an image format with packed"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <formats>
        <format name="VK_FORMAT_R4G4_UNORM_PACK8" class="8-bit" blockSize="1" texelsPerBlock="1" packed="8">
            <component name="R" bits="4" numericFormat="UNORM"/>
            <component name="G" bits="4" numericFormat="UNORM"/>
        </format>
    </formats>"""

    image_format_metadata = formats_parser.parse(ET.fromstring(xml))
    image_format = image_format_metadata.formats["VK_FORMAT_R4G4_UNORM_PACK8"]
    assert image_format.packed == 8


def test_vulkan_image_format_components() -> None:
    """"Test an image format with components"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <formats>
       <format name="VK_FORMAT_A1R5G5B5_UNORM_PACK16" class="16-bit" blockSize="2" texelsPerBlock="1" packed="16">
            <component name="A" bits="1" numericFormat="UNORM"/>
            <component name="R" bits="5" numericFormat="UNORM"/>
            <component name="G" bits="5" numericFormat="UNORM"/>
            <component name="B" bits="5" numericFormat="UNORM"/>
        </format>
    </formats>"""

    image_format_metadata = formats_parser.parse(ET.fromstring(xml))
    image_format = image_format_metadata.formats["VK_FORMAT_A1R5G5B5_UNORM_PACK16"]

    components = list(image_format.components.values())

    assert components[0].name == "A"
    assert components[0].bits == 1
    assert components[0].numeric_format == "UNORM"

    assert components[1].name == "R"
    assert components[1].bits == 5
    assert components[1].numeric_format == "UNORM"

    assert components[2].name == "G"
    assert components[2].bits == 5
    assert components[2].numeric_format == "UNORM"

    assert components[3].name == "B"
    assert components[3].bits == 5
    assert components[3].numeric_format == "UNORM"


def test_vulkan_image_format_uncompressed_components() -> None:
    """"Test an image format with uncompressed components"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <formats>
       <format name="VK_FORMAT_A1R5G5B5_UNORM_PACK16" class="16-bit" blockSize="2" texelsPerBlock="1" packed="16">
            <component name="A" bits="1" numericFormat="UNORM"/>
            <component name="R" bits="5" numericFormat="UNORM"/>
            <component name="G" bits="5" numericFormat="UNORM"/>
            <component name="B" bits="5" numericFormat="UNORM"/>
        </format>
    </formats>"""

    image_format_metadata = formats_parser.parse(ET.fromstring(xml))
    image_format = image_format_metadata.formats["VK_FORMAT_A1R5G5B5_UNORM_PACK16"]

    components = list(image_format.components.values())
    assert not components[0].compressed
    assert not components[1].compressed
    assert not components[2].compressed
    assert not components[3].compressed


def test_vulkan_image_format_compressed_components() -> None:
    """"Test an image format with compressed components"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <formats>
       <format name="VK_FORMAT_BC5_SNORM_BLOCK" class="BC5" blockSize="16" texelsPerBlock="16"
            blockExtent="4,4,1" compressed="BC">
            <component name="R" bits="compressed" numericFormat="SRGB"/>
            <component name="G" bits="compressed" numericFormat="SRGB"/>
        </format>
    </formats>"""

    image_format_metadata = formats_parser.parse(ET.fromstring(xml))
    image_format = image_format_metadata.formats["VK_FORMAT_BC5_SNORM_BLOCK"]

    components = list(image_format.components.values())

    assert components[0].compressed
    assert components[1].compressed


def test_vulkan_image_format_planes() -> None:
    """"Test an image format with planes"""

    xml = """<?xml version="1.0" encoding="UTF-8"?>
    <formats>
       <format name="VK_FORMAT_G10X6_B10X6R10X6_2PLANE_444_UNORM_3PACK16" class="10-bit 2-plane 444" blockSize="6"
            texelsPerBlock="1" packed="16" chroma="444">
            <component name="G" bits="10" numericFormat="UNORM" planeIndex="0"/>
            <component name="B" bits="10" numericFormat="UNORM" planeIndex="1"/>
            <component name="R" bits="10" numericFormat="UNORM" planeIndex="1"/>
            <plane index="0" widthDivisor="1" heightDivisor="1" compatible="VK_FORMAT_R10X6_UNORM_PACK16"/>
            <plane index="1" widthDivisor="1" heightDivisor="1" compatible="VK_FORMAT_R10X6G10X6_UNORM_2PACK16"/>
        </format>
    </formats>"""

    image_format_metadata = formats_parser.parse(ET.fromstring(xml))
    image_format = image_format_metadata.formats["VK_FORMAT_G10X6_B10X6R10X6_2PLANE_444_UNORM_3PACK16"]

    assert image_format.planes[0].index == 0
    assert image_format.planes[0].width_divisor == 1
    assert image_format.planes[0].height_divisor == 1
    assert image_format.planes[0].compatible == "VK_FORMAT_R10X6_UNORM_PACK16"

    assert image_format.planes[1].index == 1
    assert image_format.planes[1].width_divisor == 1
    assert image_format.planes[1].height_divisor == 1
    assert image_format.planes[1].compatible == "VK_FORMAT_R10X6G10X6_UNORM_2PACK16"
