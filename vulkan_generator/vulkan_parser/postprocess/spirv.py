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
This module responsible for postprocessing the SpirV Metadata

All the stringly typed references will be linked during this stage.
"""


from typing import Dict
from typing import Optional
from typing import Union

from vulkan_generator.vulkan_parser.internal import internal_types
from vulkan_generator.vulkan_parser.api import types
from vulkan_generator.vulkan_parser.api import query


def process(
        internal_metadata: internal_types.SpirvMetadata,
        new_defines: Dict[str, types.VulkanDefine],
        vulkan_types: types.VulkanTypeInfo,
        vulkan_versions: Dict[str, types.VulkanCoreVersion],
        vulkan_extensions: Dict[str, types.VulkanExtension]) -> types.SpirvMetadata:
    """
    Post process Spirv metadata for the public API
    This will do the linking between SpirV and Vulkan
    """
    new_extensions: Dict[str, types.SpirvExtension] = {}
    for spirv_extension in internal_metadata.extensions.values():
        vulkan_version_for_ext: Optional[types.VulkanCoreVersion] = None
        if spirv_extension.version:
            # For some reasons some Spirv extensions uses API version
            # instead of Vulkan version
            version_str = spirv_extension.version.replace("VK_API_VERSION", "VK_VERSION")
            vulkan_version_for_ext = vulkan_versions[version_str]

        new_extensions[spirv_extension.name] = types.SpirvExtension(
            name=spirv_extension.name,
            version=vulkan_version_for_ext,
            vulkan_extension=vulkan_extensions[spirv_extension.vulkan_extension]
        )

    new_capabilities: Dict[str, types.SpirvCapability] = {}
    for spirv_capability in internal_metadata.capabilities.values():
        vulkan_version_for_cap: Optional[types.VulkanCoreVersion] = None
        if spirv_capability.version:
            vulkan_version_for_cap = vulkan_versions[version_str]

        vulkan_extension: Optional[types.VulkanExtension] = None
        if spirv_capability.vulkan_extension:
            vulkan_extension = vulkan_extensions[spirv_capability.vulkan_extension]

        new_feature: Optional[types.SpirvCapabilityFeature] = None
        if spirv_capability.feature:
            feature_struct = query.get_struct_or_deducted_struct(spirv_capability.feature.struct, vulkan_types)
            new_feature = types.SpirvCapabilityFeature(
                struct=feature_struct,
                feature=feature_struct.members[spirv_capability.feature.feature],
            )

        new_property: Optional[types.SpirvCapabilityProperty] = None
        if spirv_capability.property:
            property_struct = vulkan_types.structs[spirv_capability.property.struct]
            property_group = property_struct.members[spirv_capability.property.group]

            property_value: Union[types.VulkanDefine, types.VulkanEnumField]
            if isinstance(property_group.type, types.VulkanBitmask):
                value_enum = property_group.type.field_type
                if not value_enum:
                    raise ValueError(f"Unexpected bitmask: {property_group.name} : {property_group.type.typename}")
                property_value = value_enum.fields[spirv_capability.property.value]
            elif isinstance(property_group.type, types.VulkanBaseType):
                property_value = new_defines[spirv_capability.property.value]
            else:
                raise ValueError(f"Unexpected property group: {property_group.name} : {property_group.type.typename}")

            new_property = types.SpirvCapabilityProperty(
                struct=property_struct,
                group=property_group,
                value=property_value
            )

        new_capabilities[spirv_capability.name] = types.SpirvCapability(
            name=spirv_capability.name,
            version=vulkan_version_for_cap,
            feature=new_feature,
            property=new_property,
            extension=vulkan_extension
        )
    return types.SpirvMetadata(
        extensions=new_extensions,
        capabilities=new_capabilities,
    )
