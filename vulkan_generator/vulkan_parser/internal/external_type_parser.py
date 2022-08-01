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

""" This module is responsible for parsing Vulkan baseinternal_types.

    Vulkan basetypes can be considered as C++ typedefs for
    either type declaration or forward declaration
    e.g.
"""


import xml.etree.ElementTree as ET

from vulkan_generator.vulkan_parser.internal import internal_types


def parse(external_type_elem: ET.Element) -> internal_types.ExternalType:
    """Returns a C Basetype from the XML element that defines it.

      C types thin Vulkan XML are defined with no tag. They can be recognised with either
      the require attribute or having no attribute

      A sample Vulkan C type with no attribute:
      <type name="int"/>

      A sample Vulkan C type with attribute:
      <type requires="vk_platform" name="float"/>
    """

    typename = external_type_elem.get("name")
    if not typename:
        raise SyntaxError(f"No type name found for C type: {ET.tostring(external_type_elem, 'utf-8')!r}")

    source_header = external_type_elem.get("requires")
    ctype = False
    if not source_header or source_header == "vk_platform":
        ctype = True

    return internal_types.ExternalType(typename=typename, source_header=source_header, ctype=ctype)
