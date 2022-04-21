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

"""This module contains the utility functions that needed elsewhere while parsing Vulkan XML"""

import xml.etree.ElementTree as ET


def get_text_from_tag(root: ET.Element, tag: str) -> str:
    """Gets the text of the element with the given tag"""
    for child in root:
        if tag == child.tag:
            return child.text

    # This should not happen
    raise SyntaxError(f"No name tag found in {ET.tostring(root, 'utf-8')}")
