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

""" This module is responsible for parsing the includes used by Vulkan.

    Some of these includes are platform specific but no information given on that
"""


import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types


def parse(include_elem: ET.Element) -> internal_types.ExternalInclude:
    """Returns an include definition from the XML element that defines it.

      A sample Vulkan Include:
      <type name="vk_platform" category="include">#include "vk_platform.h"</type>
    """

    header = include_elem.attrib["name"]
    directive = include_elem.text

    return internal_types.ExternalInclude(header=header, directive=directive)
