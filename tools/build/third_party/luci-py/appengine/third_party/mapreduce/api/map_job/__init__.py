#!/usr/bin/env python
# Copyright 2015 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Map job package."""

# All the public API should be imported here.
# 1. Seasoned Python user should simply import this package.
# 2. Other users may import individual files so filenames should still have
#    "map_job" prefix. But adding the prefix won't mandate the first type
#    of user to type more.
# 3. Class names should not have "map_job" prefix.
from .input_reader import InputReader
from .map_job_config import JobConfig
from .map_job_control import Job
from .mapper import Mapper
from .output_writer import OutputWriter
