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
_URL_LINUX = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.21-202109060500/swt-4.21-gtk-linux-x86_64.zip"
_SHA_LINUX = "3e35a4ababf504bcf64df864e4a957aaa2f0dec9696a922d936c7ee224fa4c5f"
_URL_WIN = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.21-202109060500/swt-4.21-win32-win32-x86_64.zip"
_SHA_WIN = "efad710dd60476c60bcef2bd5f855d9784cb825ecef1fb8d4c73fd470402ad32"
_URL_OSX = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.21-202109060500/swt-4.21-cocoa-macosx-x86_64.zip"
_SHA_OSX = "d1c9afbef014488cc4d768edcdfb726e90228927b7555672199d7931f140ff5d"

def _swt_impl(repository_ctx):
    url = ""
    sha = ""
    if repository_ctx.os.name.startswith("linux"):
        url = _URL_LINUX
        sha = _SHA_LINUX
    elif repository_ctx.os.name.startswith("windows"):
        url = _URL_WIN
        sha = _SHA_WIN
    elif repository_ctx.os.name.startswith("mac os"):
        url = _URL_OSX
        sha = _SHA_OSX
    else:
        fail("No SWT available for os: " + repository_ctx.os.name)

    repository_ctx.download_and_extract(
        url = url,
        output = ".",
        sha256 = sha,
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
