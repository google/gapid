# Copyright (C) 2018 Google Inc.
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

# Repository rule to download and extract SWT.
_URL_LINUX = "http://download.eclipse.org/eclipse/downloads/drops4/R-4.9-201809060745/swt-4.9-gtk-linux-x86_64.zip"
_URL_WIN = "http://download.eclipse.org/eclipse/downloads/drops4/R-4.9-201809060745/swt-4.9-win32-win32-x86_64.zip"
_URL_OSX = "http://download.eclipse.org/eclipse/downloads/drops4/R-4.9-201809060745/swt-4.9-cocoa-macosx-x86_64.zip"

def _swt_impl(repository_ctx):
    url = ""
    if repository_ctx.os.name.startswith("linux"):
        url = _URL_LINUX
    elif repository_ctx.os.name.startswith("windows"):
        url = _URL_WIN
    elif repository_ctx.os.name.startswith("mac os"):
        url = _URL_OSX
    else:
        fail("No SWT available for os: " + repository_ctx.os.name)

    repository_ctx.download_and_extract(
        url = url,
        output = ".",
    )
    repository_ctx.file("BUILD.bazel",
        content = """
java_import(
    name = "swt",
    jars = ["swt.jar"],
    visibility = ["//visibility:public"]
)
""",
    )

swt = repository_rule(
    implementation = _swt_impl,
)

