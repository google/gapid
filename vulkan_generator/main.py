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

from pathlib import Path
import sys

from vulkan_generator import generator


def main() -> None:
    """ Entry point """
    if len(sys.argv) == 4:
        generator.generate(target = sys.argv[1], output_dir = Path(sys.argv[2]), vulkan_xml_path = Path(sys.argv[3]))
    elif len(sys.argv) == 2:
        generator.generate("", "", sys.argv[1])
    else:
        print("""
            Please run this as one of the following:
            1) ./main.py <path to xml>
                This will parse the XML file and print a report.
            2) ./main.py <target to generate> <path to generate code files> <path to xml>
                This will parse the XML file and generate the requested code into a source directory.
            """)


if __name__ == "__main__":
    main()
