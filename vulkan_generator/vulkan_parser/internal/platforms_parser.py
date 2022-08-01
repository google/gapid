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

""" This module is responsible for parsing Vulkan platforms"""


from typing import Dict

import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types


def parse(platforms_elem: ET.Element) -> Dict[str, internal_types.ExternalPlatform]:
    """Parses the platforms that Vulkan is targeting"""

    platforms: Dict[str, internal_types.ExternalPlatform] = {}

    for platform in platforms_elem:
        if platform.tag != "platform":
            raise SyntaxError(f"Unexpected tag in platforms: {ET.tostring(platform, 'utf-8')!r}")

        name = platform.attrib["name"]
        protect = platform.attrib["protect"]

        platforms[name] = internal_types.ExternalPlatform(
            name=name,
            protect=protect,
        )

    return platforms
