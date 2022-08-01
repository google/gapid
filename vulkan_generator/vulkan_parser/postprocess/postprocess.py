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
This module responsible for postprocessing the Vulkan Metadata

All the stringly typed references will be linked during this stage.
"""


from vulkan_generator.vulkan_parser.internal import internal_types
from vulkan_generator.vulkan_parser.postprocess import spirv
from vulkan_generator.vulkan_parser.postprocess import vulkan_defines
from vulkan_generator.vulkan_parser.postprocess import vulkan_commands
from vulkan_generator.vulkan_parser.postprocess import vulkan_image_formats
from vulkan_generator.vulkan_parser.postprocess import vulkan_includes
from vulkan_generator.vulkan_parser.postprocess import vulkan_platforms
from vulkan_generator.vulkan_parser.postprocess import vulkan_types
from vulkan_generator.vulkan_parser.postprocess import vulkan_versions_and_extensions
from vulkan_generator.vulkan_parser.api import types


def process(internal_metadata: internal_types.VulkanMetadata) -> types.VulkanInfo:
    """
    Post process all the internal structure for public API
    This will link all the referred type each other for preventing a stringly typed API
    """
    all_platforms = vulkan_platforms.process(internal_metadata.platforms)

    # Process includes and defines here separate than Vulkan types.
    all_includes = vulkan_includes.process(internal_metadata.types.includes)
    all_defines = vulkan_defines.process(internal_metadata.types.defines)

    all_types = vulkan_types.process(internal_metadata.types, all_defines)
    all_commands = vulkan_commands.process(internal_metadata.commands, all_types)

    core_versions = vulkan_versions_and_extensions.process_core_versions(
        internal_metadata.core_versions,
        all_includes,
        all_defines,
        all_types,
        all_commands,
    )

    extensions = vulkan_versions_and_extensions.process_extensions(
        internal_metadata.extensions,
        all_platforms,
        all_includes,
        all_defines,
        all_types,
        all_commands,
        core_versions,
    )

    image_formats = vulkan_image_formats.process(internal_metadata.image_formats)
    spirv_metadata = spirv.process(
        internal_metadata.spirv_metadata, all_defines, all_types, core_versions, extensions)

    return types.VulkanInfo(
        platforms=all_platforms,
        includes=all_includes,
        defines=all_defines,
        types=all_types,
        commands=all_commands,
        core_versions=core_versions,
        extensions=extensions,
        image_formats=image_formats,
        spirv_metadata=spirv_metadata,
    )
