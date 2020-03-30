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
_URL_LINUX = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.15-202003050155/swt-4.15-gtk-linux-x86_64.zip"
_SHA_LINUX = "b64caa701e41627b6a442898f0d8d76f995b6cf3c409569bce1c3c1f9238d381"
_URL_WIN = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.15-202003050155/swt-4.15-win32-win32-x86_64.zip"
_SHA_WIN = "df74b98238c949dd30fa2f56d762fab03e3bf78a008c580186d32e7878da89d6"
_URL_OSX = "https://download.eclipse.org/eclipse/downloads/drops4/R-4.15-202003050155/swt-4.15-cocoa-macosx-x86_64.zip"
_SHA_OSX = "23ffbcf7a4b020a3a7ce6ce3cfc16a27c7724f6b4173ee01f9285d61ad996f2d"

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
