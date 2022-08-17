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
_URL_LINUX = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.24-202206070700/swt-4.24-gtk-linux-x86_64.zip"
_SHA_LINUX = "39f51ec43e750ffca909627a1167ad60faf0b263ffa7eda4cdcd708b7de17e90"
_URL_WIN = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.24-202206070700/swt-4.24-win32-win32-x86_64.zip"
_SHA_WIN = "b5b2bf94b6d411dedf8456289b25fcbe82889b01ebe7898606218dc0c5ad7968"
_URL_OSX = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.24-202206070700/swt-4.24-cocoa-macosx-x86_64.zip"
_SHA_OSX = "f4b02c8e513b798586a6605c72de1a1c0a7a0e01e9233b1482ce01428c7b7111"

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
