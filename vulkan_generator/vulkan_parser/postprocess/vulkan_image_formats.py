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
This module responsible for postprocessing the Vulkan Image formats

All the stringly typed references will be linked during this stage.
"""


from typing import Dict
from typing import List

from vulkan_generator.vulkan_parser.internal import internal_types
from vulkan_generator.vulkan_parser.api import types


def process(internal_formats: Dict[str, internal_types.ImageFormat]) -> Dict[str, types.ImageFormat]:
    """Post process image format data for the external API"""
    image_formats: Dict[str, types.ImageFormat] = {}

    for image_format in internal_formats.values():
        new_components: Dict[str, types.ImageFormatComponent] = {}
        for component in image_format.components.values():
            new_components[component.name] = types.ImageFormatComponent(
                name=component.name,
                numeric_format=component.numeric_format,
                bits=component.bits,
                compressed=component.compressed,
            )

        new_planes: List[types.ImageFormatPlane] = []
        for plane in image_format.planes:
            new_planes.append(types.ImageFormatPlane(
                index=plane.index,
                width_divisor=plane.width_divisor,
                height_divisor=plane.height_divisor,
                compatible=plane.compatible,
            ))

        image_formats[image_format.name] = types.ImageFormat(
            name=image_format.name,
            format_class=image_format.format_class,
            block_size=image_format.block_size,
            texels_per_block=image_format.texels_per_block,
            packed=image_format.packed,
            spirv_format=image_format.spirv_format,
            components=new_components,
            planes=new_planes,
        )

    return image_formats
