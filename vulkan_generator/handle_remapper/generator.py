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

"""This is the entry point for Vulkan Code Generator"""

import abc

from pathlib import Path
from typing import List

from textwrap import dedent
from vulkan_generator.vulkan_parser import types
from vulkan_generator.codegen_utils import codegen


def handle_map_name(handle: str) -> str:
    return f"""{handle[0:1].lower() + handle[1:]}_handle_map_"""


def handle_count_map_name(handle: str) -> str:
    return f"""{handle[0:1].lower() + handle[1:]}_count_map_"""


def handle_add_name(handle: str) -> str:
    return f"""Add{handle[0:1].upper() + handle[1:]}Handle"""


def handle_remove_name(handle: str) -> str:
    return f"""Remove{handle[0:1].upper() + handle[1:]}Handle"""


def handle_remap_name(handle: str) -> str:
    return f"""Remap{handle[0:1].upper() + handle[1:]}Handle"""


class HandleAccessorCodeGenerator(metaclass=abc.ABCMeta):
    ''' Abstract base class for generating accessor code implementations '''
    @abc.abstractmethod
    def handle_add_code(self, handle: str) -> str:
        pass

    @abc.abstractmethod
    def handle_remove_code(self, handle: str) -> str:
        pass

    @abc.abstractmethod
    def handle_remap_code(self, handle: str) -> str:
        pass


class DispatchableHandleAccessorCodeGenerator(HandleAccessorCodeGenerator):
    ''' Class for generating accessor code implementations for dispatchable handles'''

    def handle_add_code(self, handle: str) -> str:
        map_name = handle_map_name(handle)
        return dedent(f"""
            if({map_name}.find(captureHandle) != {map_name}.end()) throw HandleCollisionException();
            {map_name}[captureHandle] = replayHandle;"""
                      )

    def handle_remove_code(self, handle: str) -> str:
        map_name = handle_map_name(handle)
        return dedent(f"""
            if({map_name}.find(captureHandle) == {map_name}.end()) throw RemoveNonExistantHandleException();
            {map_name}.erase(captureHandle);"""
                      )

    def handle_remap_code(self, handle: str) -> str:
        map_name = handle_map_name(handle)
        return dedent(f"""
            if({map_name}.find(captureHandle) == {map_name}.end()) throw RemapNonExistantHandleException();
            return {map_name}[captureHandle];"""
                      )


class NonDispatchableHandleAccessorCodeGenerator(HandleAccessorCodeGenerator):
    ''' Class for generating accessor code implementations for non-dispatchable handles'''

    def handle_add_code(self, handle: str) -> str:
        map_name = handle_map_name(handle)
        map_count_name = handle_count_map_name(handle)
        return dedent(f"""
            auto map_iter = {map_name}.find(captureHandle);
            auto map_count_iter = {map_count_name}.find(captureHandle);

            if(map_iter == {map_name}.end()) {{
                if(map_count_iter != {map_count_name}.end()) throw InternalConsistencyException();
            }}
            else {{
                if(map_count_iter == {map_count_name}.end() || map_count_iter->second <= 0) throw InternalConsistencyException();
                if(map_iter->second != replayHandle) throw NonDispatchableHandleRedefinitionException();
            }}

            {map_name}[captureHandle] = replayHandle;
            {map_count_name}[captureHandle]++;"""
                      )

    def handle_remove_code(self, handle: str) -> str:
        map_name = handle_map_name(handle)
        map_count_name = handle_count_map_name(handle)
        return dedent(f"""
            auto map_iter = {map_name}.find(captureHandle);
            auto map_count_iter = {map_count_name}.find(captureHandle);

            if(map_iter == {map_name}.end() || map_count_iter == {map_count_name}.end()) {{
                throw RemoveNonExistantHandleException();
            }}
            else {{
                if(map_count_iter->second <= 0) throw InternalConsistencyException();
            }}

            if((--{map_count_name}[captureHandle]) <= 0){{
                {map_name}.erase(map_iter);
                {map_count_name}.erase(map_count_iter);
            }}"""
                      )

    def handle_remap_code(self, handle: str) -> str:
        map_name = handle_map_name(handle)
        map_count_name = handle_count_map_name(handle)
        return dedent(f"""
            auto map_iter = {map_name}.find(captureHandle);
            auto map_count_iter = {map_count_name}.find(captureHandle);

            if(map_iter == {map_name}.end() || map_count_iter == {map_count_name}.end()) {{
                throw RemapNonExistantHandleException();
            }}
            else {{
                if(map_count_iter->second <= 0) throw InternalConsistencyException();
            }}

            return map_iter->second;"""
                      )


def generate_handle_remapper_h(file_path: Path, vulkan_metadata: types.VulkanMetadata):
    ''' Generates handle_remapper.h '''
    with open(file_path, "w", encoding="ascii") as remapper_h:

        remapper_h.write(codegen.generated_license_header())

        remapper_h.write(dedent("""
            #include <map>
            #include <iostream>
            #include <map>

            #include "replay2/core_utils/non_copyable.h"
            #include "replay2/vulkan_base/vulkan_handle.h"

            namespace agi {
            namespace replay2 {

        """))

        private_members: List[str] = []
        public_members: List[str] = []
        public_functions: List[str] = []

        public_members.append(codegen.create_exception_declaration("InternalConsistencyException"))
        public_members.append(codegen.create_exception_declaration("HandleCollisionException"))
        public_members.append(codegen.create_exception_declaration("NonDispatchableHandleRedefinitionException",
                                                                   base_class="HandleCollisionException"))
        public_members.append(codegen.create_exception_declaration("RemoveNonExistantHandleException"))
        public_members.append(codegen.create_exception_declaration("RemapNonExistantHandleException"))

        for handle in vulkan_metadata.types.handles:

            private_members.append(f"""std::map<VulkanHandle, VulkanHandle> {handle_map_name(handle)};""")

            if not vulkan_metadata.types.handles[handle].dispatchable:
                private_members.append(f"""std::map<VulkanHandle, int> {handle_count_map_name(handle)};""")

            private_members.append("")

            public_functions.append(codegen.create_function_declaration(handle_add_name(handle),
                                                                        arguments={"captureHandle": "VulkanHandle",
                                                                                   "replayHandle": "VulkanHandle"}))
            public_functions.append(codegen.create_function_declaration(handle_remove_name(handle),
                                                                        arguments={"captureHandle": "VulkanHandle"}))
            public_functions.append(codegen.create_function_declaration(handle_remap_name(handle),
                                                                        return_type="VulkanHandle",
                                                                        arguments={"captureHandle": "VulkanHandle"}))
            public_functions.append("")

        remapper_class_def = codegen.create_class_definition("VulkanHandleRemapper",
                                                             public_inheritance=["non_copyable"],
                                                             public_functions=public_functions,
                                                             public_members=public_members,
                                                             private_members=private_members)
        remapper_h.write(remapper_class_def + "\n")

        remapper_h.write(dedent("""
            }
            }

        """))


def generate_handle_remapper_cpp(file_path: Path, vulkan_metadata: types.VulkanMetadata):
    ''' Generates handle_remapper.cc '''
    with open(file_path, "w", encoding="ascii") as remapper_cpp:

        remapper_cpp.write(codegen.generated_license_header())

        remapper_cpp.write(dedent("""
            #include "handle_remapper.h"

            namespace agi {
            namespace replay2 {

        """))

        dispatchable_implgen = DispatchableHandleAccessorCodeGenerator()
        nondispatchable_implgen = NonDispatchableHandleAccessorCodeGenerator()

        for handle in vulkan_metadata.types.handles:

            dispatchable = vulkan_metadata.types.handles[handle].dispatchable
            implgenerator = dispatchable_implgen if dispatchable else nondispatchable_implgen

            add_definition = codegen.create_function_definition(
                f"""VulkanHandleRemapper::{handle_add_name(handle)}""",
                arguments={"captureHandle": "VulkanHandle",
                           "replayHandle": "VulkanHandle"},
                code=implgenerator.handle_add_code(handle))

            remove_definition = codegen.create_function_definition(
                f"""VulkanHandleRemapper::{handle_remove_name(handle)}""",
                arguments={"captureHandle": "VulkanHandle"},
                code=implgenerator.handle_remove_code(handle))

            remap_definition = codegen.create_function_definition(
                f"""VulkanHandleRemapper::{handle_remap_name(handle)}""",
                arguments={"captureHandle": "VulkanHandle"},
                return_type="VulkanHandle",
                code=implgenerator.handle_remap_code(handle))

            remapper_cpp.write(codegen.comment_code(code=add_definition,
                                                    comment="record the creation of a new handle") + "\n")
            remapper_cpp.write(codegen.comment_code(code=remove_definition,
                                                    comment="record the destruction of a handle") + "\n")
            remapper_cpp.write(codegen.comment_code(code=remap_definition,
                                                    comment="remap a handle from the capture to replay handle") + "\n")

        remapper_cpp.write(dedent("""
            }
            }

        """))


def generate_handle_remapper_tests(file_path: Path, vulkan_metadata: types.VulkanMetadata):
    ''' Generates handle_remapper_tests.cc '''
    with open(file_path, "w", encoding="ascii") as tests_cpp:

        tests_cpp.write(codegen.generated_license_header())

        tests_cpp.write(dedent("""
            #include "handle_remapper.h"
            #include <gtest/gtest.h>

            using namespace agi::replay2;
            """))

        for handle in vulkan_metadata.types.handles:
            tests_cpp.write(dedent(f"""
            TEST(VulkanHandleRemapper, {handle}BasicRemap) {{
                VulkanHandleRemapper mapper;
                EXPECT_THROW(mapper.{handle_remap_name(handle)}(1234), VulkanHandleRemapper::RemapNonExistantHandleException);
                EXPECT_NO_THROW(mapper.{handle_add_name(handle)}(1234, 5678));
                EXPECT_NO_THROW(EXPECT_EQ(mapper.{handle_remap_name(handle)}(1234), 5678));
                EXPECT_NO_THROW(mapper.{handle_remove_name(handle)}(1234));
                EXPECT_THROW(mapper.{handle_remap_name(handle)}(1234), VulkanHandleRemapper::RemapNonExistantHandleException);
            }}"""))

            tests_cpp.write(dedent(f"""
            TEST(VulkanHandleRemapper, {handle}UnknownRemap) {{
                VulkanHandleRemapper mapper;
                EXPECT_NO_THROW(mapper.{handle_add_name(handle)}(1234, 5678));
                EXPECT_THROW(mapper.{handle_remap_name(handle)}(5678), VulkanHandleRemapper::RemapNonExistantHandleException);
            }}"""))

            if vulkan_metadata.types.handles[handle].dispatchable:
                tests_cpp.write(dedent(f"""
                TEST(VulkanHandleRemapper, Dispatchable{handle}Redefinition) {{
                    VulkanHandleRemapper mapper;
                    EXPECT_NO_THROW(mapper.{handle_add_name(handle)}(1234, 5678));
                    EXPECT_THROW(mapper.{handle_add_name(handle)}(1234, 5678), VulkanHandleRemapper::HandleCollisionException);
                    EXPECT_THROW(mapper.{handle_add_name(handle)}(1234, 8765), VulkanHandleRemapper::HandleCollisionException);
                    EXPECT_NO_THROW(EXPECT_EQ(mapper.{handle_remap_name(handle)}(1234), 5678));
                    EXPECT_NO_THROW(mapper.{handle_remove_name(handle)}(1234));
                    EXPECT_THROW(mapper.{handle_remap_name(handle)}(1234), VulkanHandleRemapper::RemapNonExistantHandleException);
                    EXPECT_THROW(mapper.{handle_remove_name(handle)}(1234), VulkanHandleRemapper::RemoveNonExistantHandleException);
                    EXPECT_THROW(mapper.{handle_remap_name(handle)}(1234), VulkanHandleRemapper::RemapNonExistantHandleException);
                    EXPECT_THROW(mapper.{handle_remove_name(handle)}(1234), VulkanHandleRemapper::RemoveNonExistantHandleException);
                    EXPECT_THROW(mapper.{handle_remap_name(handle)}(1234), VulkanHandleRemapper::RemapNonExistantHandleException);
                }}"""))
            else:
                tests_cpp.write(dedent(f"""
                TEST(VulkanHandleRemapper, NonDispatchable{handle}Redefinition) {{
                    VulkanHandleRemapper mapper;
                    EXPECT_NO_THROW(mapper.{handle_add_name(handle)}(1234, 5678));
                    EXPECT_NO_THROW(mapper.{handle_add_name(handle)}(1234, 5678));
                    EXPECT_THROW(mapper.{handle_add_name(handle)}(1234, 8765), VulkanHandleRemapper::NonDispatchableHandleRedefinitionException);
                    EXPECT_NO_THROW(EXPECT_EQ(mapper.{handle_remap_name(handle)}(1234), 5678));
                    EXPECT_NO_THROW(mapper.{handle_remove_name(handle)}(1234));
                    EXPECT_NO_THROW(EXPECT_EQ(mapper.{handle_remap_name(handle)}(1234), 5678));
                    EXPECT_NO_THROW(mapper.{handle_remove_name(handle)}(1234));
                    EXPECT_THROW(mapper.{handle_remap_name(handle)}(1234), VulkanHandleRemapper::RemapNonExistantHandleException);
                    EXPECT_THROW(mapper.{handle_remove_name(handle)}(1234), VulkanHandleRemapper::RemoveNonExistantHandleException);
                    EXPECT_THROW(mapper.{handle_remap_name(handle)}(1234), VulkanHandleRemapper::RemapNonExistantHandleException);
                }}"""))

        tests_cpp.write(dedent("""
        int main(int argc, char **argv) {
            ::testing::InitGoogleTest(&argc, argv);
            return RUN_ALL_TESTS();
        }
        """))
