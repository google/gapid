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
This module responsible for postprocessing the Vulkan Versions and Extensions

All the stringly typed references will be linked during this stage.
"""


from dataclasses import dataclass
from dataclasses import field

from typing import Dict
from typing import List
from typing import Optional
from typing import Union

from vulkan_generator.vulkan_parser.internal import internal_types
from vulkan_generator.vulkan_parser.api import types


@dataclass
class RequiredInfo:
    platforms: Dict[str, types.ExternalPlatform] = field(default_factory=dict)
    includes: Dict[str, types.ExternalInclude] = field(default_factory=dict)
    defines: Dict[str, types.VulkanDefine] = field(default_factory=dict)
    type_info: types.VulkanTypeInfo = field(default=types.VulkanTypeInfo())
    command_info: types.VulkanCommandInfo = field(default=types.VulkanCommandInfo())
    versions: Dict[str, types.VulkanCoreVersion] = field(default_factory=dict)


def process_feature_set(
        features: Dict[str, internal_types.VulkanFeature],
        required_info: RequiredInfo) -> types.VulkanFeatureSet:
    """
    Post process Vulkan Features
    This will process all the linking and group the required/added features to their types
    """

    new_feature_set = types.VulkanFeatureSet()

    for feature in features.values():
        if feature.feature_type == "type":
            if feature.name in required_info.type_info.all_types:
                new_feature_set.types[feature.name] = required_info.type_info.all_types[feature.name]
            elif feature.name in required_info.type_info.all_aliases:
                new_feature_set.type_aliases[feature.name] = required_info.type_info.all_aliases[feature.name]
            elif feature.name in required_info.includes:
                new_feature_set.includes[feature.name] = required_info.includes[feature.name]
            elif feature.name in required_info.defines:
                new_feature_set.defines[feature.name] = required_info.defines[feature.name]
            else:
                raise ValueError(f"Unknown required type: {feature}")
        elif feature.feature_type == "command":
            if feature.name in required_info.command_info.commands:
                new_feature_set.commands[feature.name] = required_info.command_info.commands[feature.name]
            elif feature.name in required_info.command_info.command_aliases:
                new_feature_set.command_aliases[feature.name] = required_info.command_info.command_aliases[feature.name]
            else:
                raise ValueError(f"Unknown required type: {feature}")
        elif feature.feature_type == "enum":
            if not isinstance(feature, internal_types.VulkanFeatureExtension):
                # If there is no feature extension for an enum, then it's a define
                new_feature_set.defines[feature.name] = required_info.defines[feature.name]
                continue

            if isinstance(feature, internal_types.VulkanFeatureExtensionDefine):
                new_feature_set.defines[feature.name] = required_info.defines[feature.name]
            elif isinstance(feature, internal_types.VulkanFeatureExtensionEnum):
                extended_enum = required_info.type_info.enums[feature.basetype]
                new_feature_set.enum_fields[feature.name] = extended_enum.fields[feature.name]
            elif isinstance(feature, internal_types.VulkanFeatureExtensionEnumAlias):
                extended_enum = required_info.type_info.enums[feature.basetype]
                new_feature_set.enum_field_aliases[feature.name] = extended_enum.aliases[feature.name]
            else:
                raise ValueError(f"Unknown feature extension type: {feature}")
        else:
            raise ValueError(f"Unknown required feature: {feature}")

    return new_feature_set


def convert_extension_type(extension_type: Optional[str]) -> types.ExtensionType:
    """Convert render pass allowance information to a typed enum"""
    if not extension_type:
        return types.ExtensionType.UNKNOWN

    if extension_type == "instance":
        return types.ExtensionType.INSTANCE

    if extension_type == "device":
        return types.ExtensionType.DEVICE

    raise ValueError(f"Unexpected Extension Type: {extension_type}")


def process_requirements(
        internal_requirements: List[internal_types.VulkanExtensionRequirement],
        new_extensions: Dict[str, types.VulkanExtension],
        required_info: RequiredInfo) -> Union[str, List[types.VulkanExtensionRequirement]]:
    """
    Post Process extension requirements.
    If any linked extension cannot be found, then this function will return the name of it instead
    """

    new_requirements: List[types.VulkanExtensionRequirement] = []

    for requirement in internal_requirements:

        required_version: Optional[types.VulkanCoreVersion] = None
        if requirement.required_version:
            required_version = required_info.versions[requirement.required_version]

        required_extension: Optional[types.VulkanExtension] = None
        if requirement.required_extension:
            required_extension = new_extensions.get(requirement.required_extension)
            if not required_extension:
                return requirement.required_extension

        new_requirements.append(types.VulkanExtensionRequirement(
            required_version=required_version,
            required_extension=required_extension,
            features=process_feature_set(requirement.features, required_info),
        ))

    return new_requirements


def process_extension(
        extension: internal_types.VulkanExtension,
        new_extensions: Dict[str, types.VulkanExtension],
        required_info: RequiredInfo) -> Union[types.VulkanExtension, str]:
    """
    Post Process extension.
    If any linked extension cannot be found, then this function will return the name of it instead
    """

    promotedto: Optional[Union[types.VulkanCoreVersion, types.VulkanExtension]] = None
    if extension.promotedto:
        promotedto = required_info.versions.get(extension.promotedto)
        if not promotedto:
            promotedto = new_extensions.get(extension.promotedto)
            if not promotedto:
                return extension.promotedto

    deprecated_by: Optional[Union[types.VulkanCoreVersion, types.VulkanExtension]] = None
    if extension.deprecatedby:
        deprecated_by = required_info.versions.get(extension.deprecatedby)
        if not deprecated_by:
            deprecated_by = new_extensions.get(extension.deprecatedby)
            if not deprecated_by:
                return extension.deprecatedby

    required_extensions: Optional[List[types.VulkanExtension]] = None
    if extension.required_extensions:
        required_extensions = []
        for ext in extension.required_extensions:
            required_extension = new_extensions.get(ext)
            if not required_extension:
                return ext

            required_extensions.append(required_extension)

    platform: Optional[types.ExternalPlatform] = None
    if extension.platform:
        platform = required_info.platforms[extension.platform]

    requirements = process_requirements(extension.requirements, new_extensions, required_info)

    if isinstance(requirements, str):
        return requirements

    return types.VulkanExtension(
        name=extension.name,
        number=extension.number,
        promotedto=promotedto,
        deprecated_by=deprecated_by,
        extension_type=convert_extension_type(extension.extension_type),
        required_extensions=required_extensions,
        platform=platform,
        requirements=requirements,
    )


# These extensions are refer each other therefore cannot be added
# recursively we need to handle it specially
circular_dep_extensions: List[str] = [
    # This is the order in the XML. Let's keep it the same
    "VK_KHR_push_descriptor",
    "VK_KHR_descriptor_update_template"
]


def copy_circular_dependent_extension(
        extension: types.VulkanExtension,
        new_extensions: Dict[str, types.VulkanExtension]) -> None:
    """
    VK_KHR_push_descriptor and VK_KHR_descriptor_update_template are circular
    dependent. Special care is needed.
    """

    if extension.name not in circular_dep_extensions:
        raise ValueError(f"Unexpected extension: {extension.name}")

    # We do not want to mistakenly create a new object and assign that as reference
    # As it would create hidden bugs with updating objects
    # Let's ensure that we do not tamper the initial object
    new_extensions[extension.name].name = extension.name
    new_extensions[extension.name].number = extension.number
    new_extensions[extension.name].deprecated_by = extension.deprecated_by
    new_extensions[extension.name].extension_type = extension.extension_type
    new_extensions[extension.name].platform = extension.platform
    new_extensions[extension.name].promotedto = extension.promotedto
    new_extensions[extension.name].required_extensions = extension.required_extensions
    new_extensions[extension.name].requirements = extension.requirements


def process_extension_with_dependencies(
        extension_name: str,
        internal_extensions: Dict[str, internal_types.VulkanExtension],
        new_extensions: Dict[str, types.VulkanExtension],
        required_info: RequiredInfo) -> None:
    """Recursively processes extensions"""

    # In Vulkan XML sometimes extensions are referred before they have been defined
    # Therefore recursively add all the extension so we can link them together in order
    # In case of a circular dependency, handle it separately so that linked object references
    # are not altered

    new_extension = process_extension(internal_extensions[extension_name], new_extensions, required_info)

    if isinstance(new_extension, str):
        process_extension_with_dependencies(new_extension, internal_extensions, new_extensions, required_info)
        process_extension_with_dependencies(extension_name, internal_extensions, new_extensions, required_info)
    else:
        if extension_name in circular_dep_extensions:
            copy_circular_dependent_extension(new_extension, new_extensions)
        else:
            new_extensions[extension_name] = new_extension


def process_extensions(internal_extensions: Dict[str, internal_types.VulkanExtension],
                       new_platforms: Dict[str, types.ExternalPlatform],
                       new_includes: Dict[str, types.ExternalInclude],
                       new_defines: Dict[str, types.VulkanDefine],
                       new_types: types.VulkanTypeInfo,
                       new_commands: types.VulkanCommandInfo,
                       new_versions: Dict[str, types.VulkanCoreVersion]) -> Dict[str, types.VulkanExtension]:
    """
    Post process Vulkan extensions
    This will process all the linking and group the required/added features to their types
    """

    required_info = RequiredInfo(
        platforms=new_platforms,
        includes=new_includes,
        defines=new_defines,
        type_info=new_types,
        command_info=new_commands,
        versions=new_versions,
    )

    new_extensions: Dict[str, types.VulkanExtension] = {}

    # Create an empty extension object for circularly dependent objects so
    # they can be referenced
    for circular_extension in circular_dep_extensions:
        new_extensions[circular_extension] = types.VulkanExtension(
            name="",
            number=0,
            deprecated_by=None,
            extension_type=types.ExtensionType.UNKNOWN,
            platform=None,
            promotedto=None,
            required_extensions=None,
            requirements=[],
        )

    for extension in internal_extensions.values():
        if extension.name in new_extensions:
            if extension.name not in circular_dep_extensions:
                # This calls a recursive function that will add all the dependencies
                # Do not add again the extension if it is already added
                continue

        process_extension_with_dependencies(extension.name, internal_extensions, new_extensions, required_info)

    if len(new_extensions) != len(internal_extensions):
        raise RuntimeError(f"""Unexpected number of extensions:
        {len(new_extensions)} : {len(internal_extensions)}""")

    return new_extensions


def process_core_versions(
        core_versions: Dict[str, internal_types.VulkanCoreVersion],
        new_includes: Dict[str, types.ExternalInclude],
        new_defines: Dict[str, types.VulkanDefine],
        new_types: types.VulkanTypeInfo,
        new_commands: types.VulkanCommandInfo,) -> Dict[str, types.VulkanCoreVersion]:
    """
    Post process Vulkan core versions
    This will process all the linking and group the required/added features to their types
    """

    required_info = RequiredInfo(
        includes=new_includes,
        defines=new_defines,
        type_info=new_types,
        command_info=new_commands,
    )

    new_versions: Dict[str, types.VulkanCoreVersion] = {}

    for version in core_versions.values():
        new_versions[version.name] = types.VulkanCoreVersion(
            name=version.name,
            number=version.number,
            features=process_feature_set(version.features, required_info),
        )

    return new_versions
